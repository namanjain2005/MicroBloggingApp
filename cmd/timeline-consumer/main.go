package main

import (
	"context"
	"log"
	"microBloggingAPP/internal/config"
	timelineconsumer "microBloggingAPP/internal/timeline-consumer"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	defer cfg.Mongo.Client.Disconnect(context.Background())

	redisOpts := &redis.Options{
		Addr:         cfg.Redis.Addr,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	}

	connStr := "amqp://guest:guest@localhost:5672/"
	server, err := timelineconsumer.NewServer(
		context.Background(),
		connStr,
		redisOpts,
		cfg.Mongo.FollowCollection,
		cfg.Redis.TimelineMaxSize,
		cfg.Redis.PostTTL,
	)
	if err != nil {
		log.Fatalf("timeline consumer init failed: %v", err)
	}

	if err := server.Subscribe(); err != nil {
		log.Fatalf("timeline consumer subscribe failed: %v", err)
	}

	log.Printf("timeline consumer running")
	select {}
}
