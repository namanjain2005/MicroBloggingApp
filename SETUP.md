# Environment Setup Guide

## MongoDB Setup

### Option 1: Local MongoDB (Development)

#### Windows
1. **Download MongoDB:**
   - Visit https://www.mongodb.com/try/download/community
   - Download the Windows installer

2. **Install MongoDB:**
   - Run the installer
   - Accept defaults
   - MongoDB installs as Windows Service

3. **Start MongoDB:**
   ```powershell
   # Already running as service, or start manually:
   "C:\Program Files\MongoDB\Server\7.0\bin\mongod.exe"
   ```

4. **Verify Installation:**
   ```powershell
   "C:\Program Files\MongoDB\Server\7.0\bin\mongo.exe"
   # In mongo shell:
   db.version()
   ```

#### macOS (Homebrew)
```bash
# Install
brew tap mongodb/brew
brew install mongodb-community

# Start service
brew services start mongodb-community

# Verify
brew services list
```

#### Linux (Ubuntu)
```bash
# Install
sudo apt-get update
sudo apt-get install -y mongodb

# Start service
sudo systemctl start mongodb
sudo systemctl status mongodb

# Verify
mongo --version
```

### Option 2: Docker MongoDB

```bash
# Run MongoDB in Docker (use a strong password; replace <your-password>)
docker run -d \
    --name mongodb \
    -p 27017:27017 \
    -e MONGO_INITDB_ROOT_USERNAME=root \
    -e MONGO_INITDB_ROOT_PASSWORD=<your-password> \
    mongo:7.0

# Verify connection
docker exec mongodb mongosh -u root -p <your-password> --authenticationDatabase admin --eval "db.version()"
```

### Option 3: Docker Compose (Complete Stack)

Already configured in `docker-compose.yml`:

```bash
docker-compose up -d
# Automatically starts MongoDB and gRPC server
```

## Go Installation

### Windows
1. Download: https://golang.org/dl/
2. Run installer (go1.25.2.windows-amd64.msi)
3. Verify:
   ```powershell
   go version
   ```

### macOS
```bash
# Using Homebrew
brew install go@1.25

# Or download from https://golang.org/dl/
```

### Linux
```bash
# Ubuntu
sudo apt-get install golang-go

# Or download from https://golang.org/dl/
```

## Environment Variables

### Setting Locally

#### Windows PowerShell
```powershell
$env:MONGO_URI = "mongodb://localhost:27017"
$env:MONGO_DB_NAME = "microBlogging"
$env:GRPC_PORT = "50051"
```

#### Windows Command Prompt
```cmd
set MONGO_URI=mongodb://localhost:27017
set MONGO_DB_NAME=microBlogging
set GRPC_PORT=50051
```

#### Linux/macOS
```bash
export MONGO_URI=mongodb://localhost:27017
export MONGO_DB_NAME=microBlogging
export GRPC_PORT=50051
```

### Persistent Environment Variables

#### Windows
1. Right-click "This PC" → Properties
2. Click "Advanced system settings"
3. Click "Environment Variables"
4. Add new user/system variables
5. Restart terminal

#### macOS/Linux
Add to `~/.bashrc` or `~/.zshrc`:
```bash
export MONGO_URI=mongodb://localhost:27017
export MONGO_DB_NAME=microBlogging
export GRPC_PORT=50051
```

Then run: `source ~/.bashrc`

### Using .env File

Create `.env` in project root:
```bash
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=microBlogging
MONGO_COLLECTION_NAME=users
GRPC_PORT=50051
GRPC_HOST=0.0.0.0
APP_ENV=development
LOG_LEVEL=info
```

The application automatically loads from environment variables with defaults.

## MongoDB Connection Strings

### Local Default
```
mongodb://localhost:27017
```

### With Authentication
```
mongodb://username:password@localhost:27017
```

### Remote MongoDB Atlas
```
mongodb+srv://username:password@cluster0.mongodb.net/database?retryWrites=true&w=majority
```

### Docker Container
```
mongodb://root:password@mongo:27017
```

## Verification Checklist

### MongoDB Setup
    -e MONGO_INITDB_ROOT_PASSWORD=<your-password> \
- [ ] Can connect: `mongosh` or `mongo` command works
- [ ] MONGO_URI environment variable is set (or using default)

### Go Setup
docker exec mongodb mongosh -u root -p <your-password> --authenticationDatabase admin --eval "db.version()"
- [ ] Go is installed: `go version` shows 1.25.2+
- [ ] GOPATH is set correctly
- [ ] Can run: `go run hello.go` works

### Application Ready
- [ ] Clone/navigate to project directory
- [ ] Run: `go mod download` (dependencies installed)
- [ ] Run: `make build` (builds successfully)
- [ ] MongoDB is running
- [ ] MONGO_URI points to running MongoDB instance

## Quick Verification Script

### Windows PowerShell
```powershell
# Check Go
Write-Host "Go Version:" (go version)

# Check MongoDB
Write-Host "Testing MongoDB connection..."
$mongoTest = & mongosh --version 2>$null
if ($mongoTest) {
    Write-Host "MongoDB: Available"
} else {
    Write-Host "MongoDB: Not found in PATH"
}

# Check environment
Create `.env` in project root from `.env.example` (do NOT commit `.env`):
```bash
# copy the example and fill in secrets
cp .env.example .env
# Edit .env and set values like:
# MONGO_URI=mongodb://localhost:27017
# MONGO_DB_NAME=microBlogging
# MONGO_COLLECTION_NAME=users
# GRPC_PORT=50051
# GRPC_HOST=0.0.0.0
# APP_ENV=development
# LOG_LEVEL=info
```
```

### Linux/macOS
```bash
#!/bin/bash

echo "Go Version:"
go version

echo ""
echo "Testing MongoDB connection..."
if command -v mongo &> /dev/null; then
    echo "MongoDB CLI: Available"
else
    echo "MongoDB CLI: Not in PATH"
fi

echo ""
echo "Environment Variables:"
echo "MONGO_URI: $MONGO_URI"
echo "GRPC_PORT: $GRPC_PORT"

echo ""
echo "Project Status:"
if [ -f "go.mod" ]; then
    echo "Project: Found"
else
    echo "Project: Not in current directory"
fi
```

## Troubleshooting

### MongoDB Won't Start
**Windows:**
```powershell
# Check service
Get-Service MongoDB
# If not running:
Start-Service MongoDB
```

**Linux:**
```bash
sudo systemctl status mongodb
sudo systemctl start mongodb
sudo systemctl enable mongodb  # Enable on boot
```

### MongoDB Connection Refused
1. Verify MongoDB is running: `netstat -tuln | grep 27017`
2. Check connection string: `echo $MONGO_URI`
3. Try connecting directly: `mongosh $MONGO_URI`

### Go Command Not Found
1. Verify installation: Check installation directory
2. Add to PATH: 
   - Windows: System Properties → Environment Variables
   - Linux/macOS: Add `export PATH=$PATH:/usr/local/go/bin` to .bashrc/.zshrc

### Port 27017 Already in Use
```bash
# Find and kill process
lsof -i :27017 | grep LISTEN | awk '{print $2}' | xargs kill -9

# Or use different port:
MONGO_URI=mongodb://localhost:27018
```

## Environment Variables Reference

| Variable | Example | Notes |
|----------|---------|-------|
| MONGO_URI | mongodb://localhost:27017 | Must point to running MongoDB |
| MONGO_DB_NAME | microBlogging | Database will be created if needed |
| MONGO_COLLECTION_NAME | users | Collection will be created if needed |
| MONGO_TIMEOUT | 10 | Seconds, increase for slow networks |
| GRPC_PORT | 50051 | Must be available |
| GRPC_HOST | 0.0.0.0 | 0.0.0.0 = all interfaces, localhost = local only |
| APP_ENV | development | Used for logging/behavior |
| LOG_LEVEL | info | debug, info, warn, error |

## Next Steps

1. ✅ Install MongoDB (if not already done)
2. ✅ Install Go (if not already done)
3. ✅ Set environment variables
4. ✅ Navigate to project directory
5. ✅ Run `make setup` to build
6. ✅ Run `make run-server` to start server
7. ✅ In another terminal, run client command

See README.md for usage examples.
