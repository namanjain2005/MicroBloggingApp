#!/bin/bash
# Quick Start Guide for Micro Blogging App

echo "=== Micro Blogging App - Quick Start ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed. Please install Go 1.25.2 or higher.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Go is installed$(go version)${NC}"
echo ""

# Check if MongoDB is accessible
echo "Checking MongoDB connection..."
if timeout 2 bash -c "cat < /dev/null > /dev/tcp/localhost/27017" 2>/dev/null; then
    echo -e "${GREEN}✓ MongoDB is accessible on localhost:27017${NC}"
else
    echo -e "${YELLOW}⚠ MongoDB is not accessible on localhost:27017${NC}"
    echo "  Please ensure MongoDB is running or set MONGO_URI environment variable"
fi
echo ""

# Build server
echo "Building server..."
cd cmd/server
go build -o server
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Server built successfully${NC}"
else
    echo -e "${RED}✗ Failed to build server${NC}"
    exit 1
fi
cd ../..
echo ""

# Build client
echo "Building client..."
cd cmd/client
go build -o client
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Client built successfully${NC}"
else
    echo -e "${RED}✗ Failed to build client${NC}"
    exit 1
fi
cd ../..
echo ""

echo -e "${GREEN}=== Setup Complete ===${NC}"
echo ""
echo "Next steps:"
echo ""
echo "1. Start the server in one terminal:"
echo "   cd cmd/server && ./server"
echo ""
echo "2. Create a user in another terminal:"
echo "   cd cmd/client && ./client -cmd=create -name=\"Your Name\" -password=\"YourPassword\""
echo ""
echo "3. Retrieve the user:"
echo "   cd cmd/client && ./client -cmd=get -id=\"<user-id>\""
echo ""
echo "For more information, see README.md"
