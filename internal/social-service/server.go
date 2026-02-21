package socialservice

import (
	//"context"
	"context"
	//"errors"
	//"fmt"
	//socialservice "microBloggingAPP/internal/social-service
	"microBloggingAPP/internal/pubsub"
	pb "microBloggingAPP/internal/social-service/socialpb"

	//userservice "microBloggingAPP/internal/user-service"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO Need to decouple or may be it should be just part of userService
type FollowServiceServer struct {
	pb.UnimplementedFollowServiceServer
	Client    *mongo.Client
	FollowCol *mongo.Collection
	UserCol   *mongo.Collection
	amqpConn  *amqp.Connection
	amqpChan  *amqp.Channel
}

type FollowServiceEventMsg struct {
	follow FollowDoc
}

const (
	ExchangeSocialFanOut = "SocialFanOut"
)

func NewServer(Client *mongo.Client,
	connStr string,
	FollowCol *mongo.Collection,
	UserCol *mongo.Collection) (*FollowServiceServer, error) {

	amqpConn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, err
	}

	amqpChan, err := amqpConn.Channel()
	if err != nil {
		amqpConn.Close()
		return nil, err
	}

	err = amqpChan.ExchangeDeclare(ExchangeSocialFanOut, "fanout", true, false, false, false, nil)
	if err != nil {
		amqpConn.Close()
		return nil, err
	}

	return &FollowServiceServer{
		Client:    Client,
		FollowCol: FollowCol,
		UserCol:   UserCol,
		amqpConn:  amqpConn,
		amqpChan:  amqpChan,
	}, nil
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

	doc, err := FollowUserReq(ctx, ser.UserCol, ser.Client, ser.FollowCol, req)

	pubsub.PublishJSON(ctx, ser.amqpChan, ExchangeSocialFanOut, "social.follow", doc)

	if err != nil {
		return nil, err
	}
	return &pb.FollowUserResponse{
		Success: true,
	}, nil
}

func (ser *FollowServiceServer) UnfollowUser(ctx context.Context, req *pb.UnfollowUserRequest) (*pb.UnfollowUserResponse, error) {
	err := ser.checkServer()
	if err != nil {
		return nil, err
	}

	doc, err := UnfollowUserReq(ctx, ser.UserCol, ser.Client, ser.FollowCol, req)

	pubsub.PublishJSON(ctx, ser.amqpChan, ExchangeSocialFanOut, "social.unfollow", doc)

	if err != nil {
		return nil, err
	}
	return &pb.UnfollowUserResponse{
		Success: true,
	}, nil
}

func (ser *FollowServiceServer) GetFollowing(ctx context.Context, req *pb.GetFollowingRequest) (*pb.GetFollowingResponse, error) {
	// TODO what to do of these events
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
