package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	MongoDB MongoDB
	GRPC    GRPC
	App     App
}

// MongoDB holds MongoDB configuration
type MongoDB struct {
	URI            string
	DBName         string
	CollectionName string
	Timeout        int
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
	// Try to auto-load .env from project root first, then internal/config if present
	if err := godotenv.Load(); err != nil {
		// attempt internal path
		if err2 := godotenv.Load("internal/config/.env"); err2 == nil {
			log.Println("Loaded environment from internal/config/.env")
		} else {
			log.Println("No .env file found in project root or internal/config; relying on process environment variables")
		}
	} else {
		log.Println("Loaded environment from .env")
	}

	required := []string{
		"MONGO_URI",
		"MONGO_DB_NAME",
		"MONGO_COLLECTION_NAME",
		"MONGO_TIMEOUT",
		"GRPC_PORT",
		"GRPC_HOST",
		"APP_ENV",
		"LOG_LEVEL",
	}

	vals := make(map[string]string, len(required))
	missing := make([]string, 0)
	for _, k := range required {
		v := os.Getenv(k)
		if v == "" {
			missing = append(missing, k)
		}
		vals[k] = v
	}

	if len(missing) > 0 {
		log.Fatalf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	timeout, err := strconv.Atoi(vals["MONGO_TIMEOUT"])
	if err != nil {
		log.Fatalf("invalid MONGO_TIMEOUT: %v", err)
	}

	return &Config{
		MongoDB: MongoDB{
			URI:            vals["MONGO_URI"],
			DBName:         vals["MONGO_DB_NAME"],
			CollectionName: vals["MONGO_COLLECTION_NAME"],
			Timeout:        timeout,
		},
		GRPC: GRPC{
			Port: vals["GRPC_PORT"],
			Host: vals["GRPC_HOST"],
		},
		App: App{
			Env:      vals["APP_ENV"],
			LogLevel: vals["LOG_LEVEL"],
		},
	}
}

// ConnectionString returns the MongoDB connection string
func (m MongoDB) ConnectionString() string {
	return m.URI
}

// Address returns the gRPC server address
func (g GRPC) Address() string {
	return fmt.Sprintf("%s:%s", g.Host, g.Port)
}
