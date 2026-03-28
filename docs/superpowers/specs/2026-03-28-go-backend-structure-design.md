# Go Backend Structure Design

**Date:** 2026-03-28
**Status:** Approved
**Scope:** Project scaffolding and structure only — no business logic implementation

## Overview

A Go backend for StrataHQ, living inside the existing monorepo at `backend/`. Serves a REST API consumed by the Next.js frontend (deployed on Vercel). Deployed on Render as a single Docker container.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Location | `backend/` inside monorepo | Single clone, shared Docker Compose, easy local dev |
| API style | REST only | Chi is a REST router, sqlc generates Go — no need for GraphQL/gRPC |
| Auth | Self-issued JWTs | Full control over auth layer, Redis for refresh token storage |
| Structure | Flat domain packages | Idiomatic Go, contributor-friendly, scales to ~15 domains |
| Deployment | Render (backend), Vercel (frontend) | Existing deployment targets |
| Observability | Structured logging (slog) + Prometheus metrics | Production-ready without over-engineering |
| Stripe pattern | Hybrid | Stripe.js on frontend, webhooks on backend |
| AI | OpenAI-compatible client → DeepSeek | Swap providers by changing env vars |

## Tech Stack

| Tool | Purpose |
|------|---------|
| Go 1.24 | Language |
| Chi | HTTP router |
| PostgreSQL 17 | Primary database |
| pgx/v5 | Postgres driver |
| sqlc | Type-safe SQL → Go codegen |
| goose | SQL migrations |
| Redis 7 | Caching, rate limiting, refresh tokens |
| Stripe Go SDK | Payment processing |
| Resend | Transactional email |
| OpenAI Go SDK | AI features (DeepSeek-compatible) |
| slog | Structured logging |
| Prometheus | Metrics |
| Docker | Containerization |
| golangci-lint | Linting |
| GitHub Actions | CI |

## Directory Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                  # entrypoint — wires everything, starts server
│
├── internal/
│   ├── config/
│   │   └── config.go                # env var loading, validation
│   │
│   ├── server/
│   │   ├── server.go                # HTTP server setup, graceful shutdown
│   │   └── router.go                # Chi router, mounts all domain routes
│   │
│   ├── middleware/
│   │   ├── auth.go                  # JWT validation, context injection
│   │   ├── logging.go               # structured request logging (slog)
│   │   ├── cors.go                  # CORS for Vercel frontend
│   │   ├── ratelimit.go             # Redis-backed rate limiting
│   │   ├── metrics.go               # Prometheus request metrics
│   │   └── recover.go               # panic recovery
│   │
│   ├── auth/
│   │   ├── handler.go               # login, register, refresh, logout endpoints
│   │   ├── service.go               # business logic — hashing, token issuance
│   │   ├── tokens.go                # JWT create/validate helpers
│   │   └── routes.go                # auth route group
│   │
│   ├── scheme/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── routes.go
│   │
│   ├── levy/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── routes.go
│   │
│   ├── maintenance/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── routes.go
│   │
│   ├── billing/
│   │   ├── handler.go               # Stripe checkout session creation
│   │   ├── webhook.go               # Stripe webhook handler
│   │   ├── service.go               # subscription state management
│   │   └── routes.go
│   │
│   ├── notification/
│   │   ├── email.go                 # Resend email client
│   │   └── templates.go             # email template definitions
│   │
│   ├── ai/
│   │   ├── client.go                # OpenAI-compatible client (DeepSeek)
│   │   └── config.go                # AI_BASE_URL, AI_API_KEY, AI_MODEL
│   │
│   └── platform/
│       ├── database/
│       │   ├── database.go          # pgx pool setup + health check
│       │   └── tx.go                # transaction helper
│       ├── cache/
│       │   └── redis.go             # Redis client setup + helpers
│       ├── health/
│       │   └── handler.go           # /healthz, /readyz endpoints
│       └── response/
│           └── json.go              # standard JSON response helpers
│
├── db/
│   ├── migrations/                  # goose SQL migration files
│   │   └── 00001_init.sql
│   ├── queries/                     # sqlc .sql files per domain
│   │   ├── auth.sql
│   │   ├── scheme.sql
│   │   ├── levy.sql
│   │   └── maintenance.sql
│   ├── sqlc.yaml                    # sqlc config
│   └── gen/                         # sqlc generated code (gitignored)
│
├── tests/
│   ├── integration/                 # integration tests (hit real DB)
│   │   ├── auth_test.go
│   │   └── testhelpers_test.go      # shared test setup, DB seeding
│   └── fixtures/                    # test data (JSON, CSV)
│
├── scripts/
│   ├── migrate.sh                   # run goose migrations
│   ├── generate.sh                  # run sqlc generate
│   └── seed.sh                      # seed dev database
│
├── .github/
│   └── workflows/
│       └── ci.yml                   # lint, test, build on PR
│
├── Dockerfile                       # multi-stage build
├── docker-compose.yml               # Postgres + Redis + backend for local dev
├── Makefile                         # dev commands
├── go.mod
├── .env.example                     # documented env vars
├── .golangci.yml                    # linter config
├── .gitignore
├── README.md                        # contributor-focused docs
└── CONTRIBUTING.md                  # how to contribute
```

## Configuration

All configuration via environment variables. Validated at startup — fail fast on missing required vars.

```bash
# Server
PORT=8080                    # Render sets this automatically
ENV=development              # development | staging | production

# Database
DATABASE_URL=postgres://stratahq:stratahq@localhost:5432/stratahq?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# Auth
JWT_SECRET=change-me-in-production
JWT_EXPIRY=15m
REFRESH_EXPIRY=168h          # 7 days

# Stripe
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Resend
RESEND_API_KEY=re_...

# AI (DeepSeek - OpenAI-compatible)
AI_BASE_URL=https://api.deepseek.com/v1
AI_API_KEY=sk-...
AI_MODEL=deepseek-chat

# CORS
ALLOWED_ORIGINS=http://localhost:3000
```

## API Design

### Response Format

```json
// Success
{ "data": { ... }, "meta": { "page": 1, "per_page": 20, "total": 42 } }

// Error
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [...] } }
```

### Middleware Chain

```
Request → Recover → Metrics → Logging → CORS → RateLimit → [Auth] → Handler
```

- Recover outermost — catches panics
- Metrics wraps all requests for Prometheus
- Auth applied per-route group — public routes skip it

### Route Structure

All routes prefixed with `/api/v1`.

**Public:**
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/billing/webhooks/stripe`
- `GET /healthz`, `GET /readyz`

**Protected (JWT required):**
- `/api/v1/schemes/*`
- `/api/v1/levies/*`
- `/api/v1/maintenance/*`
- `/api/v1/billing/*`

## Database

### sqlc Configuration

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "migrations/"
    gen:
      go:
        package: "dbgen"
        out: "gen"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
```

- sqlc reads goose migrations as schema source — single source of truth
- Generated code in `db/gen/` is gitignored — generated in CI and locally via `make generate`
- SQL-only goose migrations with `-- +goose Up` / `-- +goose Down` sections
- All monetary values stored as `bigint` (cents)
- All IDs are UUIDs

### Connection Pattern

- pgx connection pool created at startup
- Pool injected into services via constructors
- Transaction helper wraps `pgx.BeginTxFunc`

## Testing

### Strategy

- **Unit tests** — next to the code (`*_test.go`), mocked dependencies, table-driven
- **Integration tests** — in `tests/integration/`, hit real Postgres + Redis via Docker Compose
- **Test isolation** — each integration test runs in a transaction that rolls back

### CI Pipeline

```yaml
jobs:
  lint:
    - golangci-lint run

  test:
    services: [postgres, redis]
    steps:
      - make generate
      - make test
      - make test-integration

  build:
    - docker build .
```

Three parallel jobs. Lint fails fast without waiting for DB services.

## Docker

### Dockerfile (multi-stage)

- **Stage 1:** `golang:1.24-alpine` — build static binary
- **Stage 2:** `alpine:3.21` — copy binary, expose `$PORT`, health check

### Docker Compose (local dev)

Three services:
- `backend` — the Go server, depends on postgres + redis
- `postgres:17-alpine` — persistent volume, port 5432
- `redis:7-alpine` — port 6379

## Contributor Experience

### Quick Start

```bash
git clone <repo>
cd stratahq-app/backend
cp .env.example .env
make docker-up
make migrate-up
make run
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make run` | Run the server locally |
| `make build` | Build binary to `bin/server` |
| `make test` | Unit tests |
| `make test-integration` | Integration tests (needs Docker) |
| `make lint` | golangci-lint |
| `make generate` | sqlc generate |
| `make migrate-up` | Run migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-create` | Create new migration (`name=xxx`) |
| `make docker-up` | Start Postgres + Redis |
| `make docker-down` | Stop containers |
| `make seed` | Seed dev database |

### Linting

golangci-lint with: `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`, `ineffassign`. Not aggressive — won't frustrate contributors.

### Documentation

- `README.md` — overview, quick start, structure, how to add a domain
- `CONTRIBUTING.md` — fork/branch workflow, conventional commits, PR template, testing expectations
- `.env.example` — every variable documented with comments

## Out of Scope

This spec covers **project structure and scaffolding only**. The following are explicitly NOT included:

- Business logic implementation
- Database schema design (beyond the init migration placeholder)
- API endpoint implementations (handlers will have placeholder routes)
- Frontend integration
- Deployment configuration (Render-specific)
