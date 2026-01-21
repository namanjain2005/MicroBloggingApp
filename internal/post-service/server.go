package postservice

import (
	"context"
	"errors"
	"log"
	pb "microBloggingAPP/internal/post-service/postpb"

	"go.mongodb.org/mongo-driver/mongo"
)

type PostServiceServer struct {
	pb.UnimplementedPostServiceServer
	postCol *mongo.Collection
	userCol *mongo.Collection
}

func NewServer(postCol *mongo.Collection, userCol *mongo.Collection) *PostServiceServer {
	return &PostServiceServer{
		postCol: postCol,
		userCol: userCol,
	}
}

func (s *PostServiceServer) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	if s.postCol == nil || s.userCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Creating post for author: %s\n", req.AuthorId)
	return PostUserReq(ctx, s.userCol, s.postCol, req)
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
