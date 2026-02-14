package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	Mongo    Mongo
	GRPC     GRPC
	App      App
	Redis    Redis
	Timeline Timeline
}

// MongoDB holds MongoDB configuration
type Mongo struct {
	URI              string
	DBName           string
	DB               *mongo.Database
	UserCollection   *mongo.Collection
	FollowCollection *mongo.Collection
	PostCollection   *mongo.Collection
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

// Redis holds Redis configuration
type Redis struct {
	Addr            string
	DB              int
	PoolSize        int
	MinIdleConns    int
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	TimelineMaxSize int64
	PostTTL         time.Duration
}

// Timeline holds timeline fanout configuration
type Timeline struct {
	BigPersonalityThreshold uint64  // Follower count threshold for big personalities
	RedisRatio              float64 // Ratio of posts from Redis cache (0.0 - 1.0)
	BigPersonalityRatio     float64 // Ratio of posts from big personalities (0.0 - 1.0)
}

// Load loads configuration from environment variables and fails fast when required variables are missing
func Load() *Config {
	_ = godotenv.Load()
	_ = godotenv.Load("internal/config/.env")
	loadProjectEnv()

	required := []string{
		"MONGO_URI",
		"MONGO_DB_NAME",
		"MONGO_USER_COLLECTION",
		"MONGO_FOLLOW_COLLECTION",
		"MONGO_POST_COLLECTION",
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
		Redis:    loadRedisConfig(),
		Timeline: loadTimelineConfig(),
	}

	cfg.initMongo(
		vals["MONGO_USER_COLLECTION"],
		vals["MONGO_FOLLOW_COLLECTION"],
		vals["MONGO_POST_COLLECTION"],
	)

	return cfg
}

func loadProjectEnv() {
	root := findProjectRoot()
	if root == "" {
		return
	}
	_ = godotenv.Load(filepath.Join(root, ".env"))
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func loadRedisConfig() Redis {
	addr := getEnvDefault("REDIS_ADDR", "localhost:6379")
	db := getEnvIntDefault("REDIS_DB", 0)
	poolSize := getEnvIntDefault("REDIS_POOL_SIZE", 50)
	minIdle := getEnvIntDefault("REDIS_MIN_IDLE_CONNS", 10)
	dialTimeout := time.Duration(getEnvIntDefault("REDIS_DIAL_TIMEOUT_SEC", 5)) * time.Second
	readTimeout := time.Duration(getEnvIntDefault("REDIS_READ_TIMEOUT_SEC", 3)) * time.Second
	writeTimeout := time.Duration(getEnvIntDefault("REDIS_WRITE_TIMEOUT_SEC", 3)) * time.Second
	timelineMax := int64(getEnvIntDefault("REDIS_TIMELINE_MAX", 1000))
	postTTLDays := getEnvIntDefault("REDIS_POST_TTL_DAYS", 7)

	return Redis{
		Addr:            addr,
		DB:              db,
		PoolSize:        poolSize,
		MinIdleConns:    minIdle,
		DialTimeout:     dialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		TimelineMaxSize: timelineMax,
		PostTTL:         time.Duration(postTTLDays) * 24 * time.Hour,
	}
}

func loadTimelineConfig() Timeline {
	threshold := uint64(getEnvIntDefault("TIMELINE_BIG_PERSONALITY_THRESHOLD", 10000))
	redisRatio := getEnvFloatDefault("TIMELINE_REDIS_RATIO", 0.7)
	bigPersonalityRatio := getEnvFloatDefault("TIMELINE_BIG_PERSONALITY_RATIO", 0.3)

	// Normalize ratios if they don't sum to ~1.0
	total := redisRatio + bigPersonalityRatio
	if total > 0 {
		redisRatio = redisRatio / total
		bigPersonalityRatio = bigPersonalityRatio / total
	}

	return Timeline{
		BigPersonalityThreshold: threshold,
		RedisRatio:              redisRatio,
		BigPersonalityRatio:     bigPersonalityRatio,
	}
}

func getEnvDefault(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func getEnvIntDefault(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvFloatDefault(key string, def float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return def
	}
	return parsed
}

func (c *Config) initMongo(userCol, followCol, postCol string) {
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
	c.Mongo.PostCollection = db.Collection(postCol)

	c.ensureIndexes(ctx)
}

func (c *Config) ensureIndexes(ctx context.Context) { //do testing with removing index and adding to see effects
	// New PostCollection indexes
	_, err := c.Mongo.PostCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "_id", Value: 1},
		},
	})
	if err != nil {
		log.Fatalf("post index _id creation failed: %v", err)
	}

	_, err = c.Mongo.PostCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "parentId", Value: 1},
		},
	})
	if err != nil {
		log.Fatalf("post index parentId creation failed: %v", err)
	}

	_, err = c.Mongo.PostCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "rootId", Value: 1},
		},
	})
	if err != nil {
		log.Fatalf("post index rootId creation failed: %v", err)
	}

	// Index for querying posts by author and creation time (for big personality queries)
	_, err = c.Mongo.PostCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "authorId", Value: 1},
			{Key: "createdAt", Value: -1},
		},
	})
	if err != nil {
		log.Fatalf("post index authorId+createdAt creation failed: %v", err)
	}

	// Existing FollowCollection indexes
	_, err = c.Mongo.FollowCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "followerId", Value: 1},
			{Key: "followeeId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalf("follow index creation failed: %v", err)
	}

	_, err = c.Mongo.FollowCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "followeeId", Value: 1},
		},
	})
	if err != nil {
		log.Fatalf("follow index creation failed: %v", err)
	}

	// Existing UserCollection index
	_, err = c.Mongo.UserCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "_id", Value: 1},
		},
		//should not this also set to unique
	})
	if err != nil {
		log.Fatalf("user index creation failed: %v", err)
	}

	// Index for followerCount to identify big personalities
	_, err = c.Mongo.UserCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "followerCount", Value: 1},
		},
	})
	if err != nil {
		log.Fatalf("user index followerCount creation failed: %v", err)
	}
}

func (m Mongo) ConnectionString() string {
	return m.URI
}

func (g GRPC) Address() string {
	return fmt.Sprintf("%s:%s", g.Host, g.Port)
}
