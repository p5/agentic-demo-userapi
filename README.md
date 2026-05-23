# webhookd

A high-performance webhook processing server that receives, verifies, and dispatches webhook events from GitHub and GitLab.

## Overview

webhookd is a lightweight, extensible webhook receiver built with Go's standard library. It provides secure webhook processing with signature verification, event dispatching to multiple handlers, and in-memory event storage.

## Architecture

The project is organized into the following packages:

### Core Packages

- **crypto/** - HMAC-SHA256 signature generation and verification
  - Provides cryptographic functions for webhook signature validation
  - Used by GitHub webhook verification

- **webhook/** - Webhook parsing and event modeling
  - Defines the `WebhookEvent` structure
  - Implements parsers for GitHub and GitLab webhook formats
  - Extracts event types and normalizes payload data

- **store/** - Event persistence layer
  - Defines `EventStore` interface for event storage
  - Implements in-memory store with filtering capabilities
  - Supports querying by source, event type, and time range

- **dispatcher/** - Event routing and worker pool
  - Manages concurrent event processing with configurable worker pools
  - Routes events to registered handlers based on event types
  - Provides back-pressure with bounded queue and error handling

- **server/** - HTTP server and API endpoints
  - Exposes REST API for webhook ingestion and event retrieval
  - Handles signature/token verification per webhook source
  - Provides health checks and event listing

- **cmd/webhookd/** - CLI entry point
  - Command-line interface with configuration flags
  - Graceful shutdown on SIGINT/SIGTERM

## Building

Build the webhookd binary:

```bash
go build -o bin/webhookd ./cmd/webhookd
```

## Running

Start the webhook server with default settings:

```bash
./bin/webhookd start --port 8080
```

With secret verification enabled:

```bash
./bin/webhookd start \
  --port 8080 \
  --github-secret "your-github-secret" \
  --gitlab-secret "your-gitlab-secret" \
  --workers 8 \
  --queue-size 200
```

### Configuration Options

- `--port` - HTTP server port (default: 8080)
- `--github-secret` - Secret for GitHub webhook signature verification (optional)
- `--gitlab-secret` - Secret token for GitLab webhook verification (optional)
- `--workers` - Number of worker goroutines for event processing (default: 4)
- `--queue-size` - Maximum size of the event queue (default: 100)

## API Endpoints

### POST /webhook/{source}

Receive and process webhook events. Supported sources: `github`, `gitlab`.

**Request:**
```bash
curl -X POST http://localhost:8080/webhook/github \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=<signature>" \
  -H "X-GitHub-Event: push" \
  -d @payload.json
```

**Response (202 Accepted):**
```json
{
  "id": "evt_1234567890",
  "status": "accepted"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid payload or parsing error
- `401 Unauthorized` - Signature verification failed
- `404 Not Found` - Unknown webhook source
- `405 Method Not Allowed` - Non-POST request
- `503 Service Unavailable` - Queue is full

### GET /health

Health check endpoint.

**Request:**
```bash
curl http://localhost:8080/health
```

**Response (200 OK):**
```json
{
  "status": "ok",
  "time": "2024-01-15T12:34:56Z"
}
```

### GET /events

List stored webhook events with optional filtering.

**Query Parameters:**
- `source` - Filter by webhook source (e.g., `github`, `gitlab`)
- `event_type` - Filter by event type (e.g., `push`, `merge_request`)
- `limit` - Maximum number of events to return (default: 50, max: 1000)

**Request:**
```bash
curl "http://localhost:8080/events?source=github&event_type=push&limit=10"
```

**Response (200 OK):**
```json
[
  {
    "id": "evt_1234567890",
    "source": "github",
    "event_type": "push",
    "payload": { ... },
    "headers": {
      "X-GitHub-Event": "push",
      "X-GitHub-Delivery": "12345"
    },
    "created_at": "2024-01-15T12:34:56Z"
  }
]
```

## Webhook Examples

### GitHub Webhook

Configure your GitHub repository webhook:
- **Payload URL:** `http://your-server:8080/webhook/github`
- **Content type:** `application/json`
- **Secret:** Your configured `--github-secret`
- **Events:** Choose events to send (push, pull_request, etc.)

Test with curl:
```bash
# Without signature verification
curl -X POST http://localhost:8080/webhook/github \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -d '{
    "ref": "refs/heads/main",
    "repository": {
      "name": "my-repo",
      "full_name": "user/my-repo"
    }
  }'
```

### GitLab Webhook

Configure your GitLab project webhook:
- **URL:** `http://your-server:8080/webhook/gitlab`
- **Secret Token:** Your configured `--gitlab-secret`
- **Trigger:** Choose events (Push events, Merge request events, etc.)

Test with curl:
```bash
# With token verification
curl -X POST http://localhost:8080/webhook/gitlab \
  -H "Content-Type: application/json" \
  -H "X-Gitlab-Event: Push Hook" \
  -H "X-Gitlab-Token: your-gitlab-secret" \
  -d '{
    "event_name": "push",
    "ref": "refs/heads/main",
    "project": {
      "name": "my-project"
    }
  }'
```

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

Run tests for a specific package:

```bash
go test ./server
go test ./dispatcher
```

## Package Documentation

### crypto

Provides cryptographic utilities for webhook signature verification.

**Key Functions:**
- `SignPayload(secret, payload []byte) string` - Generate HMAC-SHA256 signature
- `VerifySignature(secret, payload []byte, signature string) error` - Verify signature

### webhook

Defines webhook event structure and parsers for different platforms.

**Key Types:**
- `WebhookEvent` - Normalized webhook event structure
- `Parser` - Interface for platform-specific parsers

**Implementations:**
- `GitHubParser` - Parses GitHub webhook payloads
- `GitLabParser` - Parses GitLab webhook payloads

### store

Event persistence with in-memory storage.

**Key Types:**
- `EventStore` - Interface for event storage operations
- `MemoryStore` - In-memory implementation with filtering
- `EventFilter` - Query filter for listing events

### dispatcher

Concurrent event processing with worker pools.

**Key Types:**
- `Dispatcher` - Manages workers and routes events to handlers
- `Handler` - Interface for event processing

**Features:**
- Configurable worker pool size
- Bounded queue with back-pressure
- Event type filtering
- Graceful shutdown

### server

HTTP server with REST API for webhook processing.

**Key Types:**
- `Server` - HTTP server with webhook endpoints
- `Config` - Server configuration

**Features:**
- Signature verification (GitHub HMAC-SHA256, GitLab token)
- JSON request/response handling
- Health checks
- Event listing with filtering

## Design Decisions

### Standard Library Only

webhookd is built exclusively with Go's standard library to:
- Minimize dependencies and attack surface
- Ensure long-term compatibility
- Simplify deployment and maintenance
- Demonstrate idiomatic Go patterns

### In-Memory Storage

The default `MemoryStore` keeps events in RAM for simplicity. For production use with persistence requirements, implement the `EventStore` interface with your preferred database.

### Worker Pool Pattern

The dispatcher uses a worker pool to:
- Limit concurrent processing and prevent resource exhaustion
- Provide back-pressure when the system is overloaded
- Enable graceful shutdown without losing in-flight events

### Signature Verification

- **GitHub:** Uses HMAC-SHA256 with the `X-Hub-Signature-256` header
- **GitLab:** Uses simple token comparison with the `X-Gitlab-Token` header

Both are optional but highly recommended for production deployments.

## Security Considerations

1. **Always use secrets** in production to verify webhook authenticity
2. **Use HTTPS** in production; this server doesn't handle TLS (use a reverse proxy)
3. **Rate limiting** is not built-in; use a reverse proxy or API gateway
4. **Input validation** is performed on all webhook payloads
5. **Memory limits** - MemoryStore is bounded only by system RAM; monitor usage

## License

This project is provided as-is for educational and production use.
