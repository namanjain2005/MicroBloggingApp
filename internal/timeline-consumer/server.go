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
	amqpConn                *amqp.Connection
	amqpChan                *amqp.Channel
	RedisClient             *redis.Client
	FollowCol               *mongo.Collection
	UserCol                 *mongo.Collection // Check follower counts
	ctx                     context.Context
	maxTimelineSize         int64
	postTTL                 time.Duration
	bigPersonalityThreshold uint64 // Follower count threshold for fanout-read
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

func NewServer(ctx context.Context, connStr string, redisOpts *redis.Options, followCol *mongo.Collection, userCol *mongo.Collection, timelineMax int64, postTTL time.Duration, bigPersonalityThreshold uint64) (*TimeLineConsumerServer, error) {
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
		amqpConn:                amqpConn,
		amqpChan:                amqpChan,
		RedisClient:             RedisClient,
		FollowCol:               followCol,
		UserCol:                 userCol,
		ctx:                     ctx,
		maxTimelineSize:         timelineMax,
		postTTL:                 postTTL,
		bigPersonalityThreshold: bigPersonalityThreshold,
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
	// Always cache the post itself
	if s.postTTL > 0 {
		pipe.Set(s.ctx, postKey, payload, s.postTTL)
	} else {
		pipe.Set(s.ctx, postKey, payload, 0)
	}

	// Check if author is a big personality
	isBig, err := s.isBigPersonality(event.AuthorID)
	if err != nil {
		log.Printf("failed to check big personality status: %v", err)
		// Continue with fanout on error to be safe
		isBig = false
	}

	if isBig {
		// For big personalities, ONLY cache the post, skip fanout-write
		log.Printf("Skipping fanout for big personality: %s", event.AuthorID)
		if _, err := pipe.Exec(s.ctx); err != nil {
			log.Printf("post cache failed: %v", err)
			return pubsub.NackRequeue
		}
		return pubsub.Ack
	}

	// For normal users, perform fanout-write
	followerIDs, err := s.loadFollowerIDs(event.AuthorID)
	if err != nil {
		return pubsub.NackRequeue
	}
	followerIDs = append(followerIDs, event.AuthorID)

	for _, followerID := range followerIDs {
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

func (s *TimeLineConsumerServer) isBigPersonality(userID string) (bool, error) {
	if s.UserCol == nil {
		return false, errors.New("userCol not initialized")
	}
	if s.bigPersonalityThreshold == 0 {
		return false, nil // Threshold not set, treat all as normal users
	}

	var result struct {
		FollowerCount uint64 `bson:"followerCount"`
	}

	err := s.UserCol.FindOne(s.ctx, bson.M{"_id": userID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // User not found, treat as normal
		}
		return false, err
	}

	return result.FollowerCount >= s.bigPersonalityThreshold, nil
}

func timelineKey(userID string) string {
	return "timeline:" + userID
}

func postCacheKey(postID string) string {
	return "post:" + postID
}
