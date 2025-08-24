# PVZ Service

A pickup point service for managing orders: acceptance, storage, issuance, and returns.

## Technologies

- **Language:** Go
- **Database:** PostgreSQL
- **Communication:** gRPC, HTTP Gateway
- **Monitoring:** Prometheus, Grafana, Jaeger
- **Message Queue:** Kafka, Outbox Pattern
- **Documentation:** Swagger
- **Testing:** Testcontainers, Minimock
- **Integrations:** Telegram Bot

## Project Structure

```
homework/
├── cmd/             # Service entrypoints (main entrypoint, outbox, notifier, gateway, swagger)
├── internal/
│   ├── adapter/     # Adapters for CLI, gRPC, etc.
│   ├── app/         # Application logic and services
│   ├── config/      # Configuration files and loaders
│   ├── domain/      # Domain models and business logic
│   ├── infra/       # Infrastructure utilities
│   │   ├── kafka/   # Kafka producer/consumer implementation
│   │   ├── telegram/# Telegram bot integration
│   │   ├── admin/   # Admin HTTP endpoints for management
│   │   └── shutdown/# Graceful shutdown handling
│   ├── repository/  # Data access layer (postgres)
│   └── workerpool/  # Worker pool implementation
├── migrations/      # Database migration scripts
├── pkg/             # Shared packages
│   ├── api/         # Generated gRPC code from protobuf
│   ├── cache/       # Custom in-memory cache implementation
│   └── db/          # Database client and transaction helpers
└── tests/           # Integration and E2E tests
```

## Quick Start

1. **Run with Docker Compose:**
   ```sh
   docker-compose up
   ```

2. **Run tests:**
   ```sh
   make test
   ```