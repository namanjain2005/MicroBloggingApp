package timelineconsumer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"microBloggingAPP/internal/events"
	"microBloggingAPP/internal/pubsub"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TimeLineConsumerServer struct {
	amqpConn    *amqp.Connection // do i need it ??
	amqpChan    *amqp.Channel
	RedisClient *redis.Client
	FollowCol   *mongo.Collection // TODO Is There Some Reason because of which rather than
	// Using this directly you might want to ask follow service to give me the list ??
	ctx             context.Context
	maxTimelineSize int64
	postTTL         time.Duration // why this ?? 
}

type UserMessage struct {
	// it just user
	// TODO convert this into proper and also change it in publishing part also
	Id             string
	Name           string
	Email          string
	Bio            string
	HashedPassword string
	FollowerCount  uint64
	CreatedAt      time.Time
}
func NewServer(ctx context.Context, connStr string, redisOpts *redis.Options, followCol *mongo.Collection, timelineMax int64, postTTL time.Duration) (*TimeLineConsumerServer, error) {
	amqpConn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, err
	}

	amqpChan, err := amqpConn.Channel()
	if err != nil {
		return nil, err
	}

	err = amqpChan.ExchangeDeclare(events.PostFanOutExchange, "fanout", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	_, err = amqpChan.QueueDeclare("SocialTimeLineService", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	err = amqpChan.QueueBind("SocialTimeLineService", "", events.PostFanOutExchange, false, nil)
	if err != nil {
		return nil, err
	}

	if redisOpts == nil {
		redisOpts = &redis.Options{
			Addr:         "localhost:6379",
			DB:           0,
			PoolSize:     50,
			MinIdleConns: 10,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}
	}
	RedisClient := redis.NewClient(redisOpts)

	if err = RedisClient.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	if timelineMax <= 0 {
		timelineMax = 1000
	}
	if postTTL == 0 {
		postTTL = 7 * 24 * time.Hour
	}

	return &TimeLineConsumerServer{
		amqpConn:        amqpConn,
		amqpChan:        amqpChan,
		RedisClient:     RedisClient,
		FollowCol:       followCol,
		ctx:             ctx,
		maxTimelineSize: timelineMax,
		postTTL:         postTTL,
	}, nil
}

// func (s *TimeLineConsumerServer) userHandler(userMsg UserMessage) pubsub.AckType {
// 	// score := float64(userMsg.CreatedAt.Unix())
// 	// follwers :=

// 	return pubsub.Ack
// }

func (s *TimeLineConsumerServer) Subscribe() error {
	return pubsub.SubscribeJSON(s.amqpChan, "SocialTimeLineService", s.postCreatedHandler)
}

type followDoc struct {
	FollowerId string `bson:"followerId"`
	FolloweeId string `bson:"followeeId"`
}

func (s *TimeLineConsumerServer) postCreatedHandler(event events.PostCreatedEvent) pubsub.AckType {
	if event.PostID == "" || event.AuthorID == "" {
		return pubsub.Ack
	}
	if event.ParentPostID != "" {
		return pubsub.Ack
	}
	if s.RedisClient == nil {
		return pubsub.NackRequeue
	}
	if s.FollowCol == nil {
		return pubsub.NackRequeue
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return pubsub.NackDiscard
	}

	postKey := postCacheKey(event.PostID)
	score := float64(event.CreatedAt.UnixMilli())

	pipe := s.RedisClient.Pipeline()
	if s.postTTL > 0 {
		pipe.Set(s.ctx, postKey, payload, s.postTTL)
	} else {
		pipe.Set(s.ctx, postKey, payload, 0)
	}

	followerIDs, err := s.loadFollowerIDs(event.AuthorID)
	if err != nil {
		return pubsub.NackRequeue
	}
	followerIDs = append(followerIDs, event.AuthorID)

	for _, followerID := range followerIDs {
		// can be very heavy if too many users are there so covert that to be just Fanout read 
		tKey := timelineKey(followerID)
		pipe.ZAdd(s.ctx, tKey, redis.Z{Score: score, Member: event.PostID})
		if s.maxTimelineSize > 0 {
			pipe.ZRemRangeByRank(s.ctx, tKey, 0, -(s.maxTimelineSize + 1))
		}
	}

	if _, err := pipe.Exec(s.ctx); err != nil {
		log.Printf("timeline fanout failed: %v", err)
		return pubsub.NackRequeue
	}

	return pubsub.Ack
}

func (s *TimeLineConsumerServer) loadFollowerIDs(authorID string) ([]string, error) {
	if authorID == "" {
		return nil, errors.New("authorID is empty")
	}

	cur, err := s.FollowCol.Find(s.ctx, bson.M{"followeeId": authorID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(s.ctx)

	ids := make([]string, 0)
	for cur.Next(s.ctx) {
		var doc followDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		if doc.FollowerId != "" {
			ids = append(ids, doc.FollowerId)
		}
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func timelineKey(userID string) string {
	return "timeline:" + userID
}

func postCacheKey(postID string) string {
	return "post:" + postID
}
