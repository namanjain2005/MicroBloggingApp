package postservice

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"microBloggingAPP/internal/events"
	pb "microBloggingAPP/internal/post-service/postpb"
	"microBloggingAPP/internal/pubsub"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PostServiceServer struct {
	pb.UnimplementedPostServiceServer
	postCol     *mongo.Collection
	userCol     *mongo.Collection
	amqpConn    *amqp.Connection
	amqpChan    *amqp.Channel
	redisClient *redis.Client
}

func NewServer(postCol *mongo.Collection, userCol *mongo.Collection, connStr string, redisOpts *redis.Options) (*PostServiceServer, error) {
	amqpConn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, err
	}

	amqpChan, err := amqpConn.Channel()
	if err != nil {
		amqpConn.Close()
		return nil, err
	}

	if err := amqpChan.ExchangeDeclare(events.PostFanOutExchange, "fanout", true, false, false, false, nil); err != nil {
		amqpConn.Close()
		return nil, err
	}

	var redisClient *redis.Client
	if redisOpts != nil {
		redisClient = redis.NewClient(redisOpts)
	}

	return &PostServiceServer{
		postCol:     postCol,
		userCol:     userCol,
		amqpConn:    amqpConn,
		amqpChan:    amqpChan,
		redisClient: redisClient,
	}, nil
}

func (s *PostServiceServer) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	if s.postCol == nil || s.userCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Creating post for author: %s\n", req.AuthorId)
	resp, err := PostUserReq(ctx, s.userCol, s.postCol, req)
	if err != nil {
		return nil, err
	}

	if resp.Post != nil && resp.Post.ParentPostId == "" {
		s.publishPostCreated(ctx, resp.Post)
	}

	return resp, nil
}

func (s *PostServiceServer) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostResponse, error) {
	if s.postCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Deleting post: %s\n", req.PostId)
	return DeletePostReq(ctx, s.postCol, req)
}

func (s *PostServiceServer) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.GetPostResponse, error) {
	if s.postCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Getting post: %s\n", req.PostId)
	return GetPostReq(ctx, s.postCol, req)
}

func (s *PostServiceServer) GetReplies(ctx context.Context, req *pb.GetRepliesRequest) (*pb.GetRepliesResponse, error) {
	if s.postCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Getting replies for post: %s\n", req.PostId)
	return GetRepliesReq(ctx, s.postCol, req)
}

func (s *PostServiceServer) GetThread(ctx context.Context, req *pb.GetThreadRequest) (*pb.GetThreadResponse, error) {
	if s.postCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Getting thread for root post: %s\n", req.RootPostId)
	return GetThreadReq(ctx, s.postCol, req)
}

func (s *PostServiceServer) LikePost(ctx context.Context, req *pb.LikePostRequest) (*pb.LikePostResponse, error) {
	if s.postCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Liking post: %s by user: %s\n", req.PostId, req.UserId)
	return LikePostReq(ctx, s.postCol, req)
}

func (s *PostServiceServer) UnlikePost(ctx context.Context, req *pb.UnlikePostRequest) (*pb.UnlikePostResponse, error) {
	if s.postCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Unliking post: %s by user: %s\n", req.PostId, req.UserId)
	return UnlikePostReq(ctx, s.postCol, req)
}

func (s *PostServiceServer) GetUserTimeline(ctx context.Context, req *pb.GetUserTimelineRequest) (*pb.GetUserTimelineResponse, error) {
	if s.redisClient == nil {
		return nil, errors.New("redis not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if req.UserId == "" {
		return nil, errors.New("user_id cannot be empty")
	}

	limit := int64(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	timelineKey := timelineKey(req.UserId)
	var ids []string
	var scores []int64

	if strings.TrimSpace(req.Cursor) == "" {
		zs, err := s.redisClient.ZRevRangeWithScores(ctx, timelineKey, 0, limit-1).Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}
		ids = make([]string, 0, len(zs))
		scores = make([]int64, 0, len(zs))
		for _, z := range zs {
			id := fmtRedisMember(z.Member)
			ids = append(ids, id)
			scores = append(scores, int64(z.Score))
		}
	} else {
		cursorScore, cursorID, err := parseTimelineCursor(req.Cursor)
		if err != nil {
			return nil, err
		}
		zs, err := s.redisClient.ZRevRangeByScoreWithScores(ctx, timelineKey, &redis.ZRangeBy{
			Min:    "-inf",
			Max:    strconv.FormatInt(cursorScore, 10),
			Offset: 0,
			Count:  limit + 1,
		}).Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}

		ids = make([]string, 0, len(zs))
		scores = make([]int64, 0, len(zs))
		afterCursor := false
		for _, z := range zs {
			id := fmtRedisMember(z.Member)
			score := int64(z.Score)
			if !afterCursor {
				if score < cursorScore || (score == cursorScore && id == cursorID) {
					afterCursor = true
				}
				continue
			}
			ids = append(ids, id)
			scores = append(scores, score)
			if int64(len(ids)) >= limit {
				break
			}
		}
		if !afterCursor {
			for _, z := range zs {
				id := fmtRedisMember(z.Member)
				score := int64(z.Score)
				ids = append(ids, id)
				scores = append(scores, score)
				if int64(len(ids)) >= limit {
					break
				}
			}
		}
	}

	if len(ids) == 0 {
		return &pb.GetUserTimelineResponse{
			Posts:      []*pb.Post{},
			NextCursor: "",
		}, nil
	}

	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, postCacheKey(id))
	}

	values, err := s.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.Post, 0, len(values))
	for _, v := range values {
		if v == nil {
			continue
		}
		raw, ok := v.(string)
		if !ok {
			continue
		}
		post, err := decodeCachedPost(raw)
		if err != nil {
			continue
		}
		posts = append(posts, post)
	}

	nextCursor := ""
	if int64(len(ids)) == limit {
		lastIdx := len(ids) - 1
		nextCursor = formatTimelineCursor(scores[lastIdx], ids[lastIdx])
	}

	return &pb.GetUserTimelineResponse{
		Posts:      posts,
		NextCursor: nextCursor,
	}, nil
}

func (s *PostServiceServer) publishPostCreated(ctx context.Context, post *pb.Post) {
	if s.amqpChan == nil || post == nil {
		return
	}
	event := events.PostCreatedEvent{
		PostID:       post.Id,
		AuthorID:     post.AuthorId,
		Text:         post.Text,
		ParentPostID: post.ParentPostId,
		RootPostID:   post.RootPostId,
		ReplyCount:   uint64(post.ReplyCount),
		LikeCount:    uint64(post.LikeCount),
		ViewCount:    uint64(post.ViewCount),
		RepostCount:  uint64(post.RepostCount),
		IsDeleted:    post.IsDeleted,
		CreatedAt:    post.CreatedAt.AsTime(),
		UpdatedAt:    post.UpdatedAt.AsTime(),
	}
	if err := pubsub.PublishJSON(ctx, s.amqpChan, events.PostFanOutExchange, "Post.created", event); err != nil {
		log.Printf("failed to publish Post.created: %v", err)
	}
}

func timelineKey(userID string) string {
	return "timeline:" + userID
}

func postCacheKey(postID string) string {
	return "post:" + postID
}

func formatTimelineCursor(score int64, postID string) string {
	return strconv.FormatInt(score, 10) + ":" + postID
}

func parseTimelineCursor(cursor string) (int64, string, error) {
	parts := strings.SplitN(cursor, ":", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("invalid cursor")
	}
	score, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", err
	}
	if parts[1] == "" {
		return 0, "", errors.New("invalid cursor")
	}
	return score, parts[1], nil
}

func fmtRedisMember(member interface{}) string {
	switch v := member.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func decodeCachedPost(raw string) (*pb.Post, error) {
	var event events.PostCreatedEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return nil, err
	}
	return &pb.Post{
		Id:           event.PostID,
		AuthorId:     event.AuthorID,
		Text:         event.Text,
		ParentPostId: event.ParentPostID,
		RootPostId:   event.RootPostID,
		ReplyCount:   int64(event.ReplyCount),
		LikeCount:    int64(event.LikeCount),
		ViewCount:    int64(event.ViewCount),
		RepostCount:  int64(event.RepostCount),
		IsDeleted:    event.IsDeleted,
		CreatedAt:    timestamppb.New(event.CreatedAt),
		UpdatedAt:    timestamppb.New(event.UpdatedAt),
	}, nil
}
