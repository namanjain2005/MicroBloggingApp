package main

import (
	"context"
	"log"
	"net"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"

	"microBloggingAPP/internal/config"
	userservice "microBloggingAPP/internal/user-service"
	pb "microBloggingAPP/internal/user-service/userpb"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting User Service\n")
	log.Printf("Environment: %s\n", cfg.App.Env)
	log.Printf("MongoDB URI: %s\n", cfg.MongoDB.URI)
	log.Printf("Database: %s, Collection: %s\n", cfg.MongoDB.DBName, cfg.MongoDB.CollectionName)
	log.Printf("gRPC Server: %s\n", cfg.GRPC.Address())

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MongoDB.Timeout)*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err = client.Disconnect(context.Background()); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	//verifying by pinging
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Successfully connected to MongoDB")

	collection := client.Database(cfg.MongoDB.DBName).Collection(cfg.MongoDB.CollectionName)

	grpcServer := grpc.NewServer()
	userServer := userservice.NewServer(collection)

	pb.RegisterUserServiceServer(grpcServer, userServer)

	listener, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPC.Address(), err)
	}
	defer listener.Close()
	log.Printf("User Service listening on %s\n", listener.Addr().String())

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

}
