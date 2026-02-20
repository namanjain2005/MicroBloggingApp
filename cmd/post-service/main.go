package main

import (
	"log"
	"microBloggingAPP/internal/config"
	postservice "microBloggingAPP/internal/post-service"
	"microBloggingAPP/internal/post-service/postpb"
	"net"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	grpcServer := grpc.NewServer()
	AmpqConnStr := "amqp://guest:guest@localhost:5672/"

	cfg := config.Load()

	redisOpts := &redis.Options{
		Addr:         cfg.Redis.Addr,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	}
	bigPersonalityThreshold := cfg.Timeline.BigPersonalityThreshold

	PostServer, err := postservice.NewServer(
		cfg.Mongo.PostCollection,
		cfg.Mongo.UserCollection,
		cfg.Mongo.FollowCollection,
		AmpqConnStr,
		redisOpts,
		bigPersonalityThreshold,
	)
	if err != nil {
		log.Fatalf("failed to create Server: %v", err)
	}

	postpb.RegisterPostServiceServer(grpcServer, PostServer)

	listener, err := net.Listen("tcp", "localhost:50055")
	if err != nil {
		log.Fatalf("failed to listen on: %v", "localhost:50055")
	}
	defer listener.Close()

	log.Printf("Post Service listening on %s", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc serve failed: %v", err)
	}
}
