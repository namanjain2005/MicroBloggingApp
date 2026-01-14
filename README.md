# Micro Blogging App - User Service

## Overview

This is a gRPC-based user service for a micro blogging application. It provides user creation and retrieval functionality with MongoDB as the backend database.

## Components

- **Server**: gRPC server that handles user operations
- **Client**: Command-line client to interact with the server
- **Config**: Centralized configuration management via environment variables
- **User Service**: Core business logic for user operations

## Prerequisites

- Go 1.25.2 or higher
- MongoDB instance running (local or remote)
- gRPC and Protocol Buffers dependencies (included in go.mod)

## Configuration

Configuration is managed through environment variables. Create a `.env` file or set the following variables:

### MongoDB Configuration

```bash
# Connection URI to MongoDB
MONGO_URI=mongodb://localhost:27017

# Database name
MONGO_DB_NAME=microBlogging

# Collection name
MONGO_COLLECTION_NAME=users

# Connection timeout in seconds
MONGO_TIMEOUT=10
```

### gRPC Server Configuration

```bash
# Server port
GRPC_PORT=50051

# Server host (0.0.0.0 for all interfaces, localhost for local only)
GRPC_HOST=0.0.0.0
```

### Application Configuration

```bash
# Environment (development, staging, production)
APP_ENV=development

# Log level (debug, info, warn, error)
LOG_LEVEL=info
```

## Running the Server

### 1. Build the server

```bash
cd cmd/server
go build -o server
```

### 2. Run the server

With default configuration (MongoDB at localhost:27017):

```bash
./server
```

With custom configuration via environment variables:

```bash
MONGO_URI=mongodb://mongodb-host:27017 GRPC_PORT=50051 ./server
```

### 3. Verify server is running

The server will output:

```
2026/01/14 10:30:00 Starting User Service
2026/01/14 10:30:00 Environment: development
2026/01/14 10:30:00 MongoDB URI: mongodb://localhost:27017
2026/01/14 10:30:00 Database: microBlogging, Collection: users
2026/01/14 10:30:00 gRPC Server: 0.0.0.0:50051
2026/01/14 10:30:01 Successfully connected to MongoDB
2026/01/14 10:30:01 User Service listening on [::]:50051
```

## Running the Client

### 1. Build the client

```bash
cd cmd/client
go build -o client
```

### 2. Create a user

```bash
./client -cmd=create -name="John Doe" -password="securePassword123"
```

Output:
```
User created successfully!
ID: 9
Name: John Doe
Mail: 
Bio: 
```

### 3. Get a user

```bash
./client -cmd=get -id="9"
```

Output:
```
User retrieved successfully!
ID: 9
Name: John Doe
Mail: 
Bio: 
Follower Count: 0
```

### 4. Connect to remote server

If the server is not on localhost:50051:

```bash
./client -cmd=create -server="10.0.0.5:50051" -name="Jane Doe" -password="pass123"
```

Or use environment variable:

```bash
GRPC_SERVER=10.0.0.5:50051 ./client -cmd=create -name="Jane Doe" -password="pass123"
```

## API Reference

### CreateUser

Creates a new user with the provided name and password.

**Request:**
```protobuf
message CreateUserRequest {
  string name = 1;
  string password = 2;
}
```

**Response:**
```protobuf
message User {
    string id = 1;
    string name = 2;
    string mail = 3;
    string bio = 4;
    string Hashedpassword = 5;
    uint64 followerCount = 6;
    google.protobuf.Timestamp createdAt = 7;
}
```

### GetUserByID

Retrieves a user by their ID.

**Request:**
```protobuf
message GetUserByIDRequest {
  string id = 1;
}
```

**Response:** `User` message

## Project Structure

```
.
├── cmd/
│   ├── server/           # Server entry point
│   │   └── main.go
│   └── client/           # Client entry point
│       └── main.go
├── internal/
│   ├── config/           # Configuration management
│   │   └── config.go
│   └── user-service/     # User service implementation
│       ├── server.go     # gRPC server implementation
│       ├── user.go       # User logic and database operations
│       ├── user.proto    # Protocol buffer definitions
│       └── userpb/       # Generated protobuf code
├── go.mod
└── go.sum
```

## Development

### Regenerate Protocol Buffers

If you modify `user.proto`, regenerate the Go code:

```bash
protoc --go_out=. --go-grpc_out=. internal/user-service/user.proto
```

### Testing

Run tests for user service:

```bash
cd internal/user-service
go test -v
```

## Troubleshooting

### MongoDB Connection Failed

- Ensure MongoDB is running: `mongod`
- Check `MONGO_URI` environment variable
- Verify network connectivity to MongoDB host

### gRPC Server Port in Use

- Change `GRPC_PORT` environment variable
- Or kill the process using the port: `lsof -i :50051` (Linux/Mac)

### Client Cannot Connect

- Verify server is running
- Check `GRPC_SERVER` environment variable or use `-server` flag
- Ensure firewall allows gRPC port

## Environment Variables Summary

| Variable | Default | Description |
|----------|---------|-------------|
| MONGO_URI | mongodb://localhost:27017 | MongoDB connection URI |
| MONGO_DB_NAME | microBlogging | Database name |
| MONGO_COLLECTION_NAME | users | Collection name |
| MONGO_TIMEOUT | 10 | MongoDB connection timeout (seconds) |
| GRPC_PORT | 50051 | gRPC server port |
| GRPC_HOST | 0.0.0.0 | gRPC server host |
| APP_ENV | development | Application environment |
| LOG_LEVEL | info | Logging level |
