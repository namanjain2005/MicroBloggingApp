package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config holds all configuration for the application
type Config struct {
	Mongo Mongo
	GRPC  GRPC
	App   App
}

// MongoDB holds MongoDB configuration
type Mongo struct {
	URI              string
	DBName           string
	DB               *mongo.Database
	UserCollection   *mongo.Collection
	FollowCollection *mongo.Collection
	Client           *mongo.Client
	Timeout          int
}

// GRPC holds gRPC server configuration
type GRPC struct {
	Port string
	Host string
}

// App holds application configuration
type App struct {
	Env      string
	LogLevel string
}

// Load loads configuration from environment variables and fails fast when required variables are missing
func Load() *Config {
	_ = godotenv.Load()
	_ = godotenv.Load("internal/config/.env")

	required := []string{
		"MONGO_URI",
		"MONGO_DB_NAME",
		"MONGO_USER_COLLECTION",
		"MONGO_FOLLOW_COLLECTION",
		"MONGO_TIMEOUT",
		"GRPC_HOST",
		"GRPC_PORT",
		"APP_ENV",
		"LOG_LEVEL",
	}

	vals := map[string]string{}
	var missing []string

	for _, k := range required {
		v := os.Getenv(k)
		if v == "" {
			missing = append(missing, k)
		}
		vals[k] = v
	}

	if len(missing) > 0 {
		log.Fatalf("missing env vars: %s", strings.Join(missing, ", "))
	}

	timeout, err := strconv.Atoi(vals["MONGO_TIMEOUT"])
	if err != nil {
		log.Fatalf("invalid MONGO_TIMEOUT: %v", err)
	}

	cfg := &Config{
		Mongo: Mongo{
			URI:     vals["MONGO_URI"],
			DBName:  vals["MONGO_DB_NAME"],
			Timeout: timeout,
		},
		GRPC: GRPC{
			Host: vals["GRPC_HOST"],
			Port: vals["GRPC_PORT"],
		},
		App: App{
			Env:      vals["APP_ENV"],
			LogLevel: vals["LOG_LEVEL"],
		},
	}

	cfg.initMongo(
		vals["MONGO_USER_COLLECTION"],
		vals["MONGO_FOLLOW_COLLECTION"],
	)

	return cfg
}

func (c *Config) initMongo(userCol, followCol string) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(c.Mongo.Timeout)*time.Second,
	)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(c.Mongo.URI))
	if err != nil {
		log.Fatalf("mongo connect failed: %v", err)
	}

	db := client.Database(c.Mongo.DBName)

	c.Mongo.Client = client
	c.Mongo.DB = db
	c.Mongo.UserCollection = db.Collection(userCol)
	c.Mongo.FollowCollection = db.Collection(followCol)

	c.ensureIndexes(ctx)
}

func (c *Config) ensureIndexes(ctx context.Context) {
	_, err := c.Mongo.FollowCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "followerId", Value: 1},
			{Key: "followeeId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalf("follow index creation failed: %v", err)
	}

	_, err = c.Mongo.UserCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalf("user index creation failed: %v", err)
	}
}

// ConnectionString returns the MongoDB connection string
func (m Mongo) ConnectionString() string {
	return m.URI
}

// Address returns the gRPC server address
func (g GRPC) Address() string {
	return fmt.Sprintf("%s:%s", g.Host, g.Port)
}
