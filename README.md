# API Gateway

A REST API gateway that routes requests to backend microservices in the Release Orchestrator platform.

## Tech Stack

- **Language:** Go 1.25
- **Framework:** Gin
- **No database** — stateless routing only

## Routes

| Path | Forwards To |
|------|-------------|
| GET /health | Health check |
| /api/v1/users/* | user-service |
| /api/v1/orders/* | order-service |
| /api/v1/payments/* | payment-service |

## Quick Start

### Prerequisites

- Go 1.25+
- The three backend services running (user-service, order-service, payment-service)

### Run locally

```bash
# Start all backend services via docker-compose
docker compose up -d

# Or run just the gateway
go run ./cmd
```

### Build Docker image

```bash
make docker-build VERSION=0.1.0
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP port |
| USER_SERVICE_URL | http://localhost:8081 | Backend user service URL |
| ORDER_SERVICE_URL | http://localhost:8082 | Backend order service URL |
| PAYMENT_SERVICE_URL | http://localhost:8083 | Backend payment service URL |

## Project Structure

```
.
├── cmd/main.go              # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── handler/             # HTTP health handler
│   └── proxy/               # Reverse proxy for backend services
├── helm/                    # Kubernetes deployment
└── Dockerfile               # Multi-stage build
```
