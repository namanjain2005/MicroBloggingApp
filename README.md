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

*** Begin README update ***

# MicroBlogging App

This repository contains multiple small services (user, post, social, search...) used by a sample micro-blogging application. The README below focuses on how to build, run and test the code locally (Linux/macOS and Windows).

Prerequisites
- Go (1.17+) installed and on your PATH
- MongoDB running or reachable via `MONGO_URI`

Build & run (Linux / macOS)

```bash
# Build server and client
cd cmd/server && go build -o server && cd -
cd cmd/client && go build -o client && cd -

# Start server in background
./cmd/server/server &

# Use the client
./cmd/client -cmd=create -name="Alice" -password="<your-password>"
```

Build & run (Windows / PowerShell)

```powershell
cd cmd\server; go build -o server.exe; cd ..\..
cd cmd\client; go build -o client.exe; cd ..\..

start cmd\server\server.exe
.
```

Helper scripts
- `quickstart.sh` / `quickstart.bat` — build common binaries used during development.
- `run-all.sh` / `run-all.bat` — start the server and timeline consumer (requires built binaries).
- `run-test.sh` / `run-test.bat` — run the repository tests (see below).

Testing and per-test timings

Each service's tests include a small `runTimed` helper that logs subtest durations in milliseconds via `t.Logf(...)`. To see the per-test timings, run tests with the `-v` flag:

```bash
go test -v ./...
```

You will see lines like:

```
user_test.go:16: duration: 0.709ms
```

If you prefer machine-readable output, you can use:

```bash
go test -json ./... | jq -r 'select(.Test) | "\(.Package) \(.Test) \(.Elapsed)"'
```

Notes
- Tests are intentionally fast; many assertions will show `0.000ms`. For demonstrations you can aggregate operations inside a test, but avoid adding sleeps in CI.
- This README now instructs using `run-test.sh` / `run-test.bat` to run the full test suite across packages.

If you want a per-test timing CSV/JSON report, I can add a small reporter that parses `go test -json` and writes a summary.

*** End README update ***
2026/01/14 10:30:00 MongoDB URI: mongodb://localhost:27017
