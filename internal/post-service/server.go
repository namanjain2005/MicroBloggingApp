package postservice

import (
	"context"
	"errors"
	"log"
	"microBloggingAPP/internal/events"
	pb "microBloggingAPP/internal/post-service/postpb"
	"microBloggingAPP/internal/pubsub"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

type PostServiceServer struct {
	pb.UnimplementedPostServiceServer
	postCol                 *mongo.Collection
	userCol                 *mongo.Collection
	followCol               *mongo.Collection // For querying following relationships
	amqpConn                *amqp.Connection
	amqpChan                *amqp.Channel
	redisClient             *redis.Client
	bigPersonalityThreshold uint64 // Follower count threshold for fanout-read
}

func NewServer(postCol *mongo.Collection, userCol *mongo.Collection, followCol *mongo.Collection, connStr string, redisOpts *redis.Options, bigPersonalityThreshold uint64) (*PostServiceServer, error) {
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
		postCol:                 postCol,
		userCol:                 userCol,
		followCol:               followCol,
		amqpConn:                amqpConn,
		amqpChan:                amqpChan,
		redisClient:             redisClient,
		bigPersonalityThreshold: bigPersonalityThreshold,
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

func (s *PostServiceServer) GetUserTimeline(req *pb.GetUserTimelineRequest, stream pb.PostService_GetUserTimelineServer) error {
	return GetUserTimelineStream(stream.Context(), s.redisClient, s.followCol, s.userCol, s.postCol, req, stream, s.bigPersonalityThreshold)
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
