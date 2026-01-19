package main

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"microBloggingAPP/internal/config"
	socialservice "microBloggingAPP/internal/social-service"
	socialpb "microBloggingAPP/internal/social-service/socialpb"
	userservice "microBloggingAPP/internal/user-service"
	userpb "microBloggingAPP/internal/user-service/userpb"
	"net"
)

func main() {
	cfg := config.Load()
	defer cfg.Mongo.Client.Disconnect(context.Background())

	log.Println("Starting MicroBlogging Service")
	log.Printf("Environment: %s", cfg.App.Env)
	log.Printf("MongoDB URI: %s", cfg.Mongo.URI)
	log.Printf(
		"Database: %s | UserCollection: %s | FollowCollection: %s",
		cfg.Mongo.DBName,
		cfg.Mongo.UserCollection.Name(),
		cfg.Mongo.FollowCollection.Name(),
	)
	log.Printf("gRPC Server: %s", cfg.GRPC.Address())

	grpcServer := grpc.NewServer()

	// Register User Service
	userServer := userservice.NewServer(cfg.Mongo.UserCollection)
	userpb.RegisterUserServiceServer(grpcServer, userServer)

	// Register Follow Service
	followServer := socialservice.NewServer(
		cfg.Mongo.Client,
		cfg.Mongo.FollowCollection,
		cfg.Mongo.UserCollection,
	)
	socialpb.RegisterFollowServiceServer(grpcServer, followServer)

	listener, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPC.Address(), err)
	}
	defer listener.Close()

	log.Printf("Service listening on %s", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc serve failed: %v", err)
	}
}
