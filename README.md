# MicroBlogging App — Services & Features

This repository is a compact microservice example implementing a small micro-blogging platform. It is intentionally minimal and focused on demonstrating service boundaries, event-driven timeline behavior, and gRPC-based RPCs.

This README documents each microservice, the features they provide, configuration and run instructions, testing and diagnostics, and security recommendations.

**Repository layout (top-level)**
- `cmd/` — entrypoints for service binaries (server, client, gateway, service mains)
- `internal/` — service implementations and shared packages (`config`, `pubsub`, service packages)
- `*.md` — documentation and examples
- `.env.example` — example environment variables (do not commit `.env`)

## Microservices and responsibilities

- **User Service** (`internal/user-service`, `cmd/user-service`) — user CRUD and profile management. Exposes gRPC RPCs for creating users, fetching user profiles, and basic user metadata used by other services.

- **Post Service** (`internal/post-service`, `cmd/post-service`) — handles creating posts and persisting them to MongoDB. Produces post events used by downstream services (timeline, search).

- **Social Service** (`internal/social-service`, `cmd/social-service`) — manages social graph operations (follow/unfollow) and follower counts. This service is used to decide fanout behavior and to expose social-related RPCs.

- **Timeline Consumer** (`timeline-consumer`, `internal/timeline-consumer`) — consumes post/fanout events and assembles user timelines. Implements hybrid fanout: writes recent posts to Redis for fast access and falls back to MongoDB for large/celebrity fanout.

- **Search Service** (`internal/search-service`, `cmd/search-service`) — indexes posts and provides search APIs (text search over posts). Uses protobuf-generated searchpb interfaces.

- **Gateway / Server** (`cmd/gateway`, `cmd/server`) — central process wrapping and wiring services together for local development. May provide a combined gRPC entry or an HTTP gateway depending on the build.

- **CLI Client** (`cmd/client`) — command-line client for basic operations (create user, create post, fetch timeline) used in examples and smoke tests.

Each service has its own package under `internal/` with protobuf (`*.proto`) and generated `*_pb.go` files where applicable.

## Key Features

- gRPC-first design: all primary RPCs are defined in `.proto` files alongside generated Go code in `*pb/` packages.
- Event-driven timeline: post creation emits events that are consumed by `timeline-consumer` to implement hybrid fanout (Redis + MongoDB chunks).
- Simple CLI tooling: `cmd/client` for quick manual tests and smoke checks.
- Per-test timing helpers: tests include `runTimed` subtests which log durations (ms) when running `go test -v`.

## Configuration

Configuration is via environment variables. Copy `.env.example` to `.env` for local development and set secrets there. Important vars:

```
MONGO_URI             # e.g. mongodb://localhost:27017
MONGO_DB_NAME         # e.g. microBlogging
MONGO_COLLECTION_NAME # e.g. users
GRPC_PORT             # default 50051
GRPC_HOST             # default 0.0.0.0
APP_ENV               # development|staging|production
LOG_LEVEL             # debug|info|warn|error
```

Never commit `.env` to the repository. Use `.env.example` as a template.

## Build & Run (local)

Build all service binaries:

```bash
cd $(git rev-parse --show-toplevel)
go build ./cmd/...
```

Run a single service (example: user service):

```bash
MONGO_URI=mongodb://localhost:27017 GRPC_PORT=50051 ./cmd/user-service/user-service
```

Suggested local development sequence (simplified):

1. Start MongoDB (local or Docker).
2. Start `timeline-consumer` (it needs to listen for events).
3. Start `post-service`, `social-service`, and `user-service`.
4. Run `cmd/client` to create users/posts and observe timeline behavior.

For a quick local stack, the included `docker-compose.yml` can start MongoDB alongside services that are containerized.

## Testing

Run tests with verbose timing output:

```bash
go test -v ./...
```

To get machine-readable per-test timing output, use `go test -json` and post-process with `jq`:

```bash
go test -json ./... | jq -r 'select(.Test) | "\(.Package) \(.Test) \(.Elapsed)"'
```

If you want a consolidated timing report, I can add a small reporter that parses the JSON output and produces CSV/JSON summaries.

## Observability & Debugging

- Logs: controlled by `LOG_LEVEL`; set to `debug` for verbose output.
- Timeline consumer and fanout components log chunk origin (Redis vs MongoDB) — useful for validating hybrid-fanout behavior.
- Use the included test-fanout script in `cmd/test/test-fanout` for end-to-end timeline validation.

## Security & Incident Guidance

- If secrets were ever committed, rotate them immediately (credentials, DB users, API keys).
- After rewriting history to remove secrets, communicate with collaborators to re-clone the repository. Example workflow:

```bash
# After you force-push cleaned history
git fetch origin --prune
# Everyone should reclone to avoid old refs
git clone <repo-url>
```

- Keep `.env` in `.gitignore`. Use credential managers or environment-specific secret stores in production.

## Where to look in the code

- Service implementations: `internal/*-service` (e.g. `internal/user-service`, `internal/post-service`, `internal/social-service`, `internal/search-service`).
- Protobufs and generated code: `internal/*/ *pb` and the top-level `postpb/`, `socialpb/`, `userpb/` folders.
- CLI and binaries: `cmd/*` (server, client, services, test helpers).
- Shared utilities: `internal/config`, `pubsub`, and other helper packages.

## Contributing & Next steps

- If you'd like, I can:
	- Add a compact architecture diagram to `README.md`.
	- Add a `docs/Security.md` with step-by-step incident response and rotation commands.
	- Produce a `run-local.sh` that orchestrates starting services in the correct order for development.

---

If you want, I will now add a short Security section to the top-level `README.md` (with the incident-response checklist), and produce a small `run-local.sh` script that starts services in recommended order. Which should I do next? 
