package main

import (
	"context"
	"fmt"
	"log"
	"microBloggingAPP/internal/config"
	postservice "microBloggingAPP/internal/post-service"
	postpb "microBloggingAPP/internal/post-service/postpb"
	socialservice "microBloggingAPP/internal/social-service"
	socialpb "microBloggingAPP/internal/social-service/socialpb"
	userservice "microBloggingAPP/internal/user-service"
	userpb "microBloggingAPP/internal/user-service/userpb"
	"net"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	defer cfg.Mongo.Client.Disconnect(context.Background())

	log.Println("Starting MicroBlogging Service")
	log.Printf("Environment: %s", cfg.App.Env)
	log.Printf("MongoDB URI: %s", cfg.Mongo.URI)
	log.Printf(
		"Database: %s | UserCollection: %s | FollowCollection: %s | PostCollection: %s",
		cfg.Mongo.DBName,
		cfg.Mongo.UserCollection.Name(),
		cfg.Mongo.FollowCollection.Name(),
		cfg.Mongo.PostCollection.Name(),
	)
	log.Printf("gRPC Server: %s", cfg.GRPC.Address())

	grpcServer := grpc.NewServer()

	// Register User Service
	userServerConnStr := "amqp://guest:guest@localhost:5672/"
	userServer,err := userservice.NewServer(cfg.Mongo.UserCollection,userServerConnStr)
	if err != nil{
		fmt.Printf("%v",err)
		return
	}
	userpb.RegisterUserServiceServer(grpcServer, userServer)

	// Register Follow Service
	followServer := socialservice.NewServer(
		cfg.Mongo.Client,
		cfg.Mongo.FollowCollection,
		cfg.Mongo.UserCollection,
	)
	socialpb.RegisterFollowServiceServer(grpcServer, followServer)

	// Register Post Service
	postServer := postservice.NewServer(
		cfg.Mongo.PostCollection,
		cfg.Mongo.UserCollection,
	)
	postpb.RegisterPostServiceServer(grpcServer, postServer)

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
