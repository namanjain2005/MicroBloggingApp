# Usage Examples

## Server Examples

### Start Server with Default Settings
```bash
cd cmd/server
go build -o server
./server
```

**Output:**
```
2026/01/14 10:30:00 Starting User Service
2026/01/14 10:30:00 Environment: development
2026/01/14 10:30:00 MongoDB URI: mongodb://localhost:27017
2026/01/14 10:30:00 Database: microBlogging, Collection: users
2026/01/14 10:30:00 gRPC Server: 0.0.0.0:50051
2026/01/14 10:30:01 Successfully connected to MongoDB
2026/01/14 10:30:01 User Service listening on [::]:50051
```

### Start Server with Custom MongoDB URI
```bash
# Windows PowerShell
$env:MONGO_URI = "mongodb://mongodb-server.example.com:27017"
./server

# Linux/macOS
export MONGO_URI="mongodb://mongodb-server.example.com:27017"
./server

# Command line
MONGO_URI="mongodb://mongodb-server.example.com:27017" ./server
```

### Start Server with Custom Port
```bash
GRPC_PORT=9000 ./server
```

### Start Server in Production Mode
```bash
APP_ENV=production LOG_LEVEL=warn ./server
```

### Start Server with All Custom Settings
```bash
MONGO_URI="mongodb://root:password@mongo:27017" \
MONGO_DB_NAME="myBlog" \
MONGO_COLLECTION_NAME="accounts" \
GRPC_PORT=50051 \
GRPC_HOST=0.0.0.0 \
APP_ENV=production \
LOG_LEVEL=info \
./server
```

## Client Examples

### Build Client
```bash
cd cmd/client
go build -o client
```

### Create User (Basic)
```bash
./client -cmd=create -name="John Doe" -password="SecurePassword123"
```

**Output:**
```
User created successfully!
ID: 8
Name: John Doe
Mail: 
Bio: 
```

### Create Multiple Users
```bash
./client -cmd=create -name="Alice Johnson" -password="alice123"
./client -cmd=create -name="Bob Smith" -password="bob456"
./client -cmd=create -name="Carol Williams" -password="carol789"
```

### Get User by ID
```bash
./client -cmd=get -id="8"
```

**Output:**
```
User retrieved successfully!
ID: 8
Name: John Doe
Mail: 
Bio: 
Follower Count: 0
```

### Connect to Remote Server
```bash
# Connect to server on different machine
./client -server="192.168.1.100:50051" -cmd=create -name="Remote User" -password="pass"

# Connect to server in Docker
./client -server="localhost:50051" -cmd=get -id="8"
```

### Using Environment Variable for Server
```bash
# Set environment variable
export GRPC_SERVER="remote-server.example.com:50051"

# Client uses it automatically
./client -cmd=create -name="Test" -password="test123"
```

### Script: Create Multiple Users
```bash
#!/bin/bash
# create_users.sh

USERS=(
  "alice:password1"
  "bob:password2"
  "carol:password3"
  "dave:password4"
  "eve:password5"
)

for user in "${USERS[@]}"; do
  IFS=':' read -r name password <<< "$user"
  echo "Creating user: $name"
  ./client -cmd=create -name="$name" -password="$password"
  sleep 1  # Small delay between requests
done
```

## Docker Examples

### Build Docker Image
```bash
docker build -t microblogging-server:v1.0.0 .
```

### Run Server in Docker
```bash
# Using local MongoDB
docker run -p 50051:50051 \
  -e MONGO_URI=mongodb://host.docker.internal:27017 \
  microblogging-server:v1.0.0

# Using remote MongoDB
docker run -p 50051:50051 \
  -e MONGO_URI=mongodb://mongodb.example.com:27017 \
  microblogging-server:v1.0.0

# With authentication
docker run -p 50051:50051 \
  -e MONGO_URI=mongodb://root:password@mongo:27017 \
  microblogging-server:v1.0.0
```

### Run Complete Stack with Docker Compose
```bash
# Start
docker-compose up -d

# View logs
docker-compose logs -f server
docker-compose logs -f mongo

# Stop
docker-compose down

# Remove volumes
docker-compose down -v
```

## Make/Makefile Examples

### Build Everything
```bash
make build
# Output:
# Building server...
# ✓ Server built: cmd/server/server
# Building client...
# ✓ Client built: cmd/client/client
# Build complete!
```

### Run Server
```bash
make run-server
# Builds and runs server
```

### Clean Binaries
```bash
make clean
```

### Format and Lint
```bash
make fmt      # Format code
make lint     # Run linter
make test     # Run tests
```

## Testing Scenarios

### Scenario 1: Local Development
```bash
# Terminal 1: Start server
make run-server

# Terminal 2: Create user
./cmd/client/client -cmd=create -name="Dev User" -password="dev123"

# Terminal 2: Get user
./cmd/client/client -cmd=get -id="8"
```

### Scenario 2: Docker Local Development
```bash
# Start everything
docker-compose up -d

# Create user
GRPC_SERVER=localhost:50051 ./cmd/client/client -cmd=create -name="Docker User" -password="docker123"

# Verify
GRPC_SERVER=localhost:50051 ./cmd/client/client -cmd=get -id="6"
```

### Scenario 3: Multiple Servers
```bash
# Terminal 1: Server on port 50051
GRPC_PORT=50051 ./server

# Terminal 2: Server on port 50052
GRPC_PORT=50052 ./server

# Terminal 3: Client operations
./client -server=localhost:50051 -cmd=create -name="Server1 User" -password="pass1"
./client -server=localhost:50052 -cmd=create -name="Server2 User" -password="pass2"
```

### Scenario 4: Load Testing
```bash
#!/bin/bash
# load_test.sh

echo "Creating 100 users..."
for i in {1..100}; do
  ./client -cmd=create -name="User$i" -password="password$i" &
  
  # Limit concurrent requests
  if [ $((i % 10)) -eq 0 ]; then
    wait
    echo "Created $i users..."
  fi
done

wait
echo "Load test complete!"
```

## Error Scenarios and Solutions

### Connection Refused
```bash
# Error: connection refused
./client -cmd=create -name="Test" -password="test"

# Solution: Check server is running
make run-server

# Or connect to specific server
./client -server=192.168.1.100:50051 -cmd=create -name="Test" -password="test"
```

### Database Connection Failed
```bash
# Server output: Failed to connect to MongoDB

# Solution: Ensure MongoDB is running
docker run -d -p 27017:27017 mongo:7.0

# Or set correct URI
MONGO_URI=mongodb://localhost:27017 ./server
```

### User Not Found
```bash
# Error: user not found
./client -cmd=get -id="999"

# Solution: Use valid user ID
./client -cmd=get -id="8"
```

### Invalid Password
```bash
# Error on create: password is required
./client -cmd=create -name="Test" -password=""

# Solution: Provide password
./client -cmd=create -name="Test" -password="validpass123"
```

## Performance Testing

### Measure Response Time
```bash
# Unix/Linux/macOS
time ./client -cmd=create -name="Perf Test" -password="test"

# Windows PowerShell
Measure-Command { ./client.exe -cmd=create -name="Perf Test" -password="test" }
```

### Concurrent Users
```bash
#!/bin/bash
# concurrent_test.sh

concurrent_count=$1
iterations=$2

for ((i=1; i<=iterations; i++)); do
  for ((j=1; j<=concurrent_count; j++)); do
    ./client -cmd=create -name="User_${i}_${j}" -password="pass" &
  done
  wait
  echo "Iteration $i completed"
done
```

Usage:
```bash
./concurrent_test.sh 10 5  # 10 concurrent users, 5 iterations
```

## Debugging

### Enable Verbose Logging
```bash
# Server side
LOG_LEVEL=debug ./server

# Shows all connections and operations
```

### Check MongoDB Collections
```bash
# In mongo shell
use microBlogging
db.users.find()
db.users.count()
db.users.findOne({_id: "8"})
```

### Network Debugging
```bash
# Check port is listening
netstat -tuln | grep 50051  # Linux/macOS
netstat -aon | findstr :50051  # Windows

# Test connection
telnet localhost 50051
nc -zv localhost 50051  # nc (netcat)
```

## CI/CD Example

### GitHub Actions
```yaml
name: Build and Test

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    
    services:
      mongo:
        image: mongo:7.0
        options: >-
          --health-cmd mongosh
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v2
    - uses: actions/setup-go@v2
      with:
        go-version: 1.25.2
    
    - name: Build
      run: make build
    
    - name: Test
      run: make test
      env:
        MONGO_URI: mongodb://localhost:27017
```
