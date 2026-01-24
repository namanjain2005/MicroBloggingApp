package userservice

import (
	"context"
	"errors"
	"log"
	"microBloggingAPP/internal/pubsub"
	pb "microBloggingAPP/internal/user-service/userpb"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO these server should be open independely
// TODO make a make close function that cann be defered for this struct
// TODO make appropriate events for if you want in it

// ServiceUserServer implements the UserServiceServer interface
type ServiceUserServer struct {
	pb.UnimplementedUserServiceServer
	UserCol  *mongo.Collection
	amqpConn *amqp.Connection
	amqpChan *amqp.Channel
	//connStr string
}



const (
	UserService       = "UserService"
	ExchangeUserTopic = "UserTopic"
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

	_, err = amqpChan.QueueDeclare(UserService, true, false, false, false, nil)
	if err != nil {
		amqpConn.Close()
		return nil, err
	}

	// TODO may be this should fanout 
	err = amqpChan.ExchangeDeclare(ExchangeUserTopic, "topic", true, false, false, false, nil)
	if err != nil {
		amqpConn.Close()
		return nil, err
	}

	err = amqpChan.QueueBind(UserService, "User.*", ExchangeUserTopic, false, nil)
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

/*
func (s *ServiceUserServer) serviceVibeCheck() error{
	if(s.col == nil){
		return errors.New("database collection is not initialized")
	}
	if(req == nil){
		return errors.New("request cannot be nil")
	}
	return nil
}*/

//func (s *ServiceUserServer)

func (s *ServiceUserServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
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

	pubsub.PublishJSON(ctx, s.amqpChan, ExchangeUserTopic,"User.create", user)

	return &pb.User{
		Id:             user.Id,
		Name:           user.Name,
		Email:          user.Email,
		Hashedpassword: user.HashedPassword,
		FollowerCount:  user.FollowerCount,
		CreatedAt:      timestamppb.New(user.CreatedAt),
	}, nil
}

func (s *ServiceUserServer) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.User, error) {
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
	return &pb.User{
		Id:            user.Id,
		Name:          user.Name,
		Email:         user.Email,
		FollowerCount: user.FollowerCount,
		Bio:           user.Bio,
		CreatedAt:     timestamppb.New(user.CreatedAt),
	}, nil
}

func (s *ServiceUserServer) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.User, error) {
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

	return &pb.User{
		Id:            user.Id,
		Name:          user.Name,
		Email:         user.Email,
		FollowerCount: user.FollowerCount,
		Bio:           user.Bio,
		CreatedAt:     timestamppb.New(user.CreatedAt),
	}, nil
}

func (s *ServiceUserServer) ModifyBio(ctx context.Context, req *pb.ModifyBioRequest) (*pb.User, error) {
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
	return &pb.User{
		Id:            user.Id,
		Name:          user.Name,
		Email:         user.Email,
		FollowerCount: user.FollowerCount,
		Bio:           user.Bio,
		CreatedAt:     timestamppb.New(user.CreatedAt),
	}, nil
}


