package userservice

import (
	"context"
	"errors"
	"log"
	pb "microBloggingAPP/internal/user-service/userpb"

	"go.mongodb.org/mongo-driver/mongo"
)

// ServiceUserServer implements the UserServiceServer interface
type ServiceUserServer struct {
	pb.UnimplementedUserServiceServer
	col *mongo.Collection
}

func NewServer(col *mongo.Collection) *ServiceUserServer {
	return &ServiceUserServer{
		col: col,
	}
}

func (s *ServiceUserServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	if s.col == nil {
		return nil, errors.New("database collection not initialized")
	}

	log.Printf("Creating user: %s\n", req.Name)
	return CreateUser(ctx, s.col, req)
}

func (s *ServiceUserServer) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.User, error) {
	if s.col == nil {
		return nil, errors.New("database collection not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	log.Printf("Getting user by ID: %s\n", req.Id)
	return GetUserByID(ctx, s.col, req.Id)
}
