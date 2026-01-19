package socialservice

import (
	//"context"
	"context"
	//"errors"
	//"fmt"
	//socialservice "microBloggingAPP/internal/social-service
	pb "microBloggingAPP/internal/social-service/socialpb"

	//userservice "microBloggingAPP/internal/user-service"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FollowServiceServer struct {
	pb.UnimplementedFollowServiceServer
	Client    *mongo.Client
	FollowCol *mongo.Collection
	UserCol   *mongo.Collection
}

func NewServer(Client *mongo.Client, FollowCol *mongo.Collection, UserCol *mongo.Collection) *FollowServiceServer {
	return &FollowServiceServer{
		Client:    Client,
		FollowCol: FollowCol,
		UserCol:   UserCol,
	}
}

func (ser *FollowServiceServer) checkServer() error {

	if ser.Client == nil {
		return status.Errorf(codes.NotFound, "Client not found")
	}

	if ser.FollowCol == nil {
		return status.Errorf(codes.NotFound, "FollowCol not found")
	}

	if ser.UserCol == nil {
		return status.Errorf(codes.NotFound, "UserCol not found")
	}
	return nil
}

func (ser *FollowServiceServer) FollowUser(ctx context.Context, req *pb.FollowUserRequest) (*pb.FollowUserResponse, error) {
	err := ser.checkServer()
	if err != nil {
		return nil, err // may want to do this before this func and assume it exist for it
	}
	return FollowUserReq(ctx, ser.UserCol, ser.Client, ser.FollowCol, req)
}

func (ser *FollowServiceServer) UnfollowUser(ctx context.Context, req *pb.UnfollowUserRequest) (*pb.UnfollowUserResponse, error) {
	err := ser.checkServer()
	if err != nil {
		return nil, err
	}
	return UnfollowUserReq(ctx, ser.UserCol, ser.Client, ser.FollowCol, req)
}

func (ser *FollowServiceServer) GetFollowing(ctx context.Context, req *pb.GetFollowingRequest) (*pb.GetFollowingResponse, error) {
	err := ser.checkServer()
	if err != nil {
		return nil, err
	}
	return GetFollowingReq(ctx, ser.FollowCol, req)
}

func (ser *FollowServiceServer) GetFollowers(ctx context.Context, req *pb.GetFollowersRequest) (*pb.GetFollowersResponse, error) {
	err := ser.checkServer()
	if err != nil {
		return nil, err
	}
	return GetFollowersReq(ctx, ser.FollowCol, req)
}
