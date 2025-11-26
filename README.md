# PR Reviewer Service

[🇷🇺 Русский](README.ru.md) | [🇬🇧 English](README.md)

Service for automatic assignment of reviewers to Pull Requests for internal teams.

## Table of Contents

- [Key Features](#key-features)
- [API Documentation](#-api-documentation)
  - [Main Endpoints](#main-endpoints)
  - [Additional Endpoints](#additional-endpoints)
- [Architecture](#architecture)
  - [Technology Stack](#technology-stack)
  - [Project Structure](#project-structure)
  - [Application Layers](#application-layers)
- [Quick Start](#quick-start)
  - [Docker Compose (Recommended)](#docker-compose-recommended)
  - [Local Run](#local-run)
  - [Configuration and Environment Variables](#configuration-and-environment-variables)
- [Development](#development)
  - [Makefile Commands](#makefile-commands)
  - [Manual Run](#manual-run)
  - [Testing](#testing)
    - [Unit Tests](#unit-tests)
    - [Integration Tests](#integration-tests)
- [Load Testing](#load-testing)
- [Observability and Metrics](#observability-and-metrics)
  - [Prometheus Metrics](#prometheus-metrics)
  - [Monitoring](#monitoring)
  - [Logging](#logging)
- [Assumptions](#assumptions)
- [Useful Links](#useful-links)

## Key Features

- Team and member management (`/team/add`, `/team/get`, `/users/setIsActive`).
- PR creation with automatic selection of up to two active reviewers from the author's team.
- Review reassignment (replacement with a random active member from the replaced reviewer's team).
- Idempotent merge and listing of PRs by reviewer.
- Additional features:
  - Assignment statistics (`/stats/assignments`).
  - Mass deactivation of team members with safe reassignment (`/team/deactivate`).
  - Load testing (see `load/load-test-report.md`).
  - Integration test (package `test/integration`).
  - Linter configuration (`.golangci.yml`).

## 📚 API Documentation

Live documentation is available in Swagger UI: `http://localhost:8080/swagger` (uses `openapi.yml` from the project root).

### Main Endpoints

| Method | Path                  | Description                                                              |
| ----- | --------------------- | --------------------------------------------------------------------- |
| POST  | `/team/add`           | Create a team with members (creates/updates users)       |
| GET   | `/team/get`           | Get a team with members                                        |
| POST  | `/users/setIsActive`  | Set user activity flag                               |
| GET   | `/users/getReview`    | Get PRs where the user is assigned as a reviewer                    |
| POST  | `/pullRequest/create` | Create a PR and automatically assign up to 2 reviewers from the author's team |
| POST  | `/pullRequest/merge`  | Mark PR as MERGED (idempotent operation)                       |
| POST  | `/pullRequest/reassign` | Reassign a specific reviewer to another from their team          |

### Additional Endpoints

| Method | Path                | Description                                                          |
| ----- | ------------------- | ----------------------------------------------------------------- |
| POST  | `/team/deactivate`  | Mass deactivation of team members with safe reassignment |
| GET   | `/stats/assignments` | Get assignment statistics by users and PRs              |
| GET   | `/health`           | Health check endpoint                                             |
| GET   | `/metrics`           | Prometheus metrics                                                |
| GET   | `/swagger`           | Swagger UI for interactive API documentation                    |

## Architecture

### Technology Stack

- **Go 1.24.10** — programming language
- **Chi v5** — HTTP router and middleware
- **PostgreSQL 16** — main data storage
- **pgx/v5** — PostgreSQL driver
- **golang-migrate** — database migration management
- **go-transaction-manager** — transaction management
- **Prometheus** — metrics collection
- **slog** — structured logging
- **Squirrel** — SQL query builder

### Project Structure

The project follows Clean Architecture principles with layer separation:

```
pr-reviewer-service_Avito/
├── cmd/run/              # Application entry point
├── config/               # YAML configuration
├── internal/
│   ├── app/              # Application initialization, migrations, graceful shutdown
│   ├── config/           # Configuration loading and parsing (YAML + ENV)
│   ├── domain/           # Domain models and errors
│   ├── repository/       # Database layer (SQL operations, transactions)
│   ├── service/          # Business logic, validation, reviewer selection
│   ├── http/
│   │   ├── handler/      # Feature-based HTTP handlers
│   │   │   ├── add_team/
│   │   │   ├── get_team/
│   │   │   ├── pull_request_create/
│   │   │   ├── pull_request_merge/
│   │   │   ├── pull_request_reassign/
│   │   │   ├── team_deactivate/
│   │   │   ├── user_set_activity/
│   │   │   ├── user_get_review/
│   │   │   ├── stats_assignments/
│   │   │   └── common/    # Common utilities (response, mappers)
│   │   ├── middleware/    # HTTP middleware (logging, metrics, panic recovery)
│   │   ├── router/       # Route registration
│   │   └── swagger/       # Swagger UI integration
│   ├── infrastructure/   # Infrastructure dependencies
│   │   ├── nower/         # Time abstraction (for testing)
│   │   └── randomizer/   # Thread-safe randomizer
│   ├── logging/          # Structured logging with context
│   └── metrics/          # Business and technical metrics
├── migrations/           # Database SQL migrations
├── load/                 # Load testing (Vegeta)
├── test/integration/     # Integration tests (Testcontainers)
├── deploy/               # Deployment configurations (Prometheus)
└── metrics/              # Grafana dashboards
```

### Application Layers

1. **`internal/repository`** — database layer:
   - All SQL operations via Squirrel and pgx
   - Transaction management via `go-transaction-manager`
   - Separation into `Storage` (main) and `txStorage` (transactional)

2. **`internal/service`** — business logic:
   - Input data validation
   - Random reviewer selection (Fisher-Yates shuffle)
   - Bulk operations in transactions
   - Operation timeout management

3. **`internal/http/handler`** — HTTP layer:
   - Feature-based structure (each feature in a separate package)
   - Centralized error handling via `WithErrorHandling`
   - Mapping between API and domain models

4. **`internal/http/middleware`** — HTTP middleware:
   - `PanicMiddleware` — panic recovery
   - `LoggerMiddleware` — structured logging with request ID
   - `MetricsMiddleware` — Prometheus metrics collection

5. **`internal/app`** — initialization:
   - Migration application on startup
   - Database connection with retries
   - Graceful shutdown with timeout

## Quick Start

### Docker Compose (Recommended)

```bash
# Start all services (app, postgres, prometheus, grafana)
docker compose up --build
```

The service will be available at `http://localhost:8080`, DB — at `localhost:5432`.  
Migrations from the `migrations` directory are automatically applied on startup.

### Local Run

```bash
# 1. Start PostgreSQL
docker compose up db -d

# 2. Build and run the application
make build
./bin/pr-reviewer

# Or directly
make run
```

### Configuration and Environment Variables

Base config is stored in `config/config.yaml`. You can specify another file via `CONFIG_PATH`.  
Any value from YAML can be overridden with environment variables.

| Variable | Default Value | Description |
|-----------|-----------------------|----------|
| `CONFIG_PATH` | `config/config.yaml` | Path to YAML config file |
| `HTTP_PORT` | `8080` | HTTP server port |
| `HTTP_READ_TIMEOUT` | `5s` | HTTP read timeout |
| `HTTP_WRITE_TIMEOUT` | `5s` | HTTP write timeout |
| `HTTP_IDLE_TIMEOUT` | `5m` | Keep-alive idle timeout |
| `DATABASE_URL` | `postgres://...` | PostgreSQL connection string |
| `DB_MAX_CONNECTIONS` | `50` | Maximum connections in pool |
| `DB_MIN_CONNECTIONS` | `5` | Minimum connections |
| `DB_MAX_CONN_IDLE_TIME` | `5m` | Connection idle timeout |
| `DB_MAX_CONN_LIFETIME` | `30m` | Connection TTL |
| `MIGRATIONS_PATH` | `migrations` | Path to SQL migrations |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown time |
| `OPERATION_TIMEOUT` | `30s` | Regular operation timeout |
| `LONG_OPERATION_TIMEOUT` | `60s` | Long operation timeout |
| `LOG_LEVEL` | `info` | Logging level (debug/info/warn/error) |
| `LOG_OUTPUT` | `stdout` | `stdout`, `stderr` or file path |
| `SWAGGER_SPEC_PATH` | `openapi.yml` | Path to OpenAPI file |

## Development

### Makefile Commands

| Command | Description |
|---------|----------|
| `make fmt` | Format application code (go fmt) |
| `make lint` | Run linters to find errors and bugs |
| `make lint-fix` | Run linters with automatic fixes |
| `make test` | Run all tests |
| `make test-clean` | Clear test cache and run again |
| `make build` | Build application binary |
| `make run` | Run application locally |
| `make compose-up` | Start docker containers |
| `make compose-down` | Stop docker containers |
| `make quick-setup` | Quick setup (only start containers) |
| `make full-setup` | Full setup (formatting, linters, containers) |
| `make load-test` | Run load testing |
| `make load-test-setup` | Only prepare environment for load test |
| `make load-test-report` | Show report from saved results |
| `make load-test-plot` | Show instructions for graph generation |
| `make tidy` | Sync dependencies (go mod tidy) |
| `make generate` | Regenerate DTOs from `openapi.yml` (oapi-codegen) |
| `make go-generate` | Run all go:generate commands |
| `make install-tools` | Install development tools (golangci-lint, oapi-codegen, goimports) |

### Manual Run

```bash
# Formatting
go fmt ./...

# Tests
go test ./...

# Linter
golangci-lint run ./...

# Build
go build -o bin/pr-reviewer ./cmd/run
```

### Testing

#### Unit Tests

```bash
# All tests
make test

# Specific package
go test ./internal/service/...

# With coverage
go test -cover ./...
```

#### Integration Tests

The `test/integration/integration_test.go` file spins up PostgreSQL via Testcontainers (skipped on Windows) and runs an end-to-end scenario: team → PR → reassign → merge → mass deactivation.

```bash
go test ./test/integration -run TestHappyPath -v
```

## Load Testing

The [Vegeta](https://github.com/tsenart/vegeta) library is used for load testing.  
Implemented as a Go program in `load/cli/`, details — in `load/README.md`.  
Brief results — in `load/load-test-report.md`. At RPS≈5, the service confidently maintains SLA 300 ms.

**Quick start:**
```bash
make load-test
# or
go run ./load/cli
```

For more details, see `load/README.md`.

## Observability and Metrics

### Prometheus Metrics

The `/metrics` endpoint provides:

**Technical metrics:**
- `http_requests_total` — total number of HTTP requests (by method, endpoint, status)
- `http_request_duration_seconds` — request duration histogram
- `http_request_size_bytes` — request body size

**Business metrics:**
- `teams_created_total` — number of teams created
- `users_processed_total` — number of users processed
- `pull_requests_created_total` — number of PRs created
- `reviewer_reassignments_total` — number of reviewer reassignments

### Monitoring

`docker compose up` starts:
- **Prometheus** (`http://localhost:9090`). Config — `deploy/prometheus/prometheus.yml`.
- **Grafana** (`http://localhost:3000`). Credentials: `admin/admin`. Import the dashboard `metrics/grafana/pr-reviewer-dashboard.json`.

### Logging

Logs are structured (slog) and contain:
- Request ID (unique for each request)
- Request path
- HTTP method
- Response status code
- Processing duration

Behavior is controlled by `LOG_LEVEL` and `LOG_OUTPUT` variables.

## Assumptions

- `team/add` returns `TEAM_EXISTS` error if the team already exists, but users can still be updated via separate endpoints.
- During mass deactivation, if no active reviewers remain in the team, the slot is freed (according to the rule "can assign less than two").
- Randomization of reviewer selection uses `math/rand` generator, sufficient for uniform load distribution within a team. For cryptographic security, can be replaced with `crypto/rand`.
- Integration test is skipped on Windows, where basic Docker rootless mode is unavailable. In CI/Linux, the test runs fully.
- All database operations are performed via transactions to ensure data consistency.

## Useful Links

- [OpenAPI specification](openapi.yml)
- [Database migrations](migrations)
- [Load testing report](load/load-test-report.md)
- [Load testing documentation](load/README.md)
