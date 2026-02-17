package userservice

import (
	"context"
	"errors"
	"log"
	"microBloggingAPP/internal/pubsub"
	"microBloggingAPP/userpb"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO these server should be open independely
// TODO make a make close function that cann be defered for this struct
// TODO make appropriate events for if you want in it

// ServiceUserServer implements the UserServiceServer interface
type ServiceUserServer struct {
	userpb.UnimplementedUserServiceServer
	UserCol  *mongo.Collection
	amqpConn *amqp.Connection
	amqpChan *amqp.Channel
	//connStr string
}

type userEventLog struct {
	user      User
	EventName string
}

const (
	UserService        = "UserService"
	ExchangeUserFanOut = "UserFanOut"
)

func NewServer(col *mongo.Collection, connStr string) (*ServiceUserServer, error) {
	amqpConn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, err
	}

	amqpChan, err := amqpConn.Channel()
	if err != nil {
		amqpConn.Close()
		return nil, err
	}

	err = amqpChan.ExchangeDeclare(ExchangeUserFanOut, "fanout", true, false, false, false, nil)
	if err != nil {
		amqpConn.Close()
		return nil, err
	}

	return &ServiceUserServer{
		UserCol:  col,
		amqpConn: amqpConn,
		amqpChan: amqpChan,
	}, nil
}

func (s *ServiceUserServer) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	if s.UserCol == nil {
		return nil, errors.New("database collection not initialized")
	}

	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	log.Printf("Creating user: %s\n", req.Name)
	user, err := CreateUser(ctx, s.UserCol, req)
	if err != nil {
		return nil, err
	}

	userMsg := &userEventLog{
		// TODO should it be value or address i think this is a
		// feature of may json.Marshal but in concept it is still struct as value
		user:      *user,
		EventName: "create",
	}

	pubsub.PublishJSON(ctx, s.amqpChan, ExchangeUserFanOut, "User.create", userMsg)

	return &userpb.CreateUserResponse{
		User: &userpb.User{
			Id:             user.Id,
			Name:           user.Name,
			Email:          user.Email,
			Hashedpassword: user.HashedPassword,
			FollowerCount:  user.FollowerCount,
			CreatedAt:      timestamppb.New(user.CreatedAt),
		},
	}, nil
}

func (s *ServiceUserServer) GetUserByID(ctx context.Context, req *userpb.GetUserByIDRequest) (*userpb.GetUserResponse, error) {
	if s.UserCol == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	log.Printf("Getting user by ID: %s\n", req.Id)
	user, err := GetUserByID(ctx, s.UserCol, req)
	if err != nil {
		return nil, err
	}
	return &userpb.GetUserResponse{
		User: &userpb.User{
			Id:            user.Id,
			Name:          user.Name,
			Email:         user.Email,
			FollowerCount: user.FollowerCount,
			Bio:           user.Bio,
			CreatedAt:     timestamppb.New(user.CreatedAt),
		},
	}, nil
}

func (s *ServiceUserServer) GetUserByEmail(ctx context.Context, req *userpb.GetUserByEmailRequest) (*userpb.GetUserResponse, error) {
	if s.UserCol == nil {
		return nil, errors.New("database collection is not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	user, err := GetUserByEmail(ctx, s.UserCol, req)

	if err != nil {
		return nil, err
	}

	return &userpb.GetUserResponse{
		User: &userpb.User{
			Id:            user.Id,
			Name:          user.Name,
			Email:         user.Email,
			FollowerCount: user.FollowerCount,
			Bio:           user.Bio,
			CreatedAt:     timestamppb.New(user.CreatedAt),
		},
	}, nil
}

func (s *ServiceUserServer) ModifyBio(ctx context.Context, req *userpb.ModifyBioRequest) (*userpb.ModifyBioResponse, error) {
	if s.UserCol == nil {
		return nil, errors.New("database collection is not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	user, err := ModifyBio(ctx, s.UserCol, req)
	if err != nil {
		return nil, err
	}

	userMsg := &userEventLog{
		user:      *user,
		EventName: "BioModification",
	}

	pubsub.PublishJSON(ctx, s.amqpChan, ExchangeUserFanOut, "User.Bio", userMsg)

	return &userpb.ModifyBioResponse{
		User: &userpb.User{
			Id:            user.Id,
			Name:          user.Name,
			Email:         user.Email,
			FollowerCount: user.FollowerCount,
			Bio:           user.Bio,
			CreatedAt:     timestamppb.New(user.CreatedAt),
		},
	}, nil
}
