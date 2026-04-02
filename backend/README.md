# StrataHQ Backend

Go backend for [StrataHQ](https://stratahq.com) — property management software for South African sectional title schemes.

## Tech Stack

| Tool | Purpose |
|------|---------|
| [Go 1.24](https://go.dev/) | Language |
| [Chi](https://github.com/go-chi/chi) | HTTP router |
| [PostgreSQL 17](https://www.postgresql.org/) | Primary database |
| [pgx/v5](https://github.com/jackc/pgx) | Postgres driver |
| [sqlc](https://sqlc.dev/) | Type-safe SQL codegen |
| [goose](https://github.com/pressly/goose) | Database migrations |
| [Redis 7](https://redis.io/) | Caching, rate limiting, sessions |
| [Stripe](https://stripe.com/) | Payment processing |
| [Resend](https://resend.com/) | Transactional email |
| [DeepSeek](https://deepseek.com/) | AI features (OpenAI-compatible) |
| [Prometheus](https://prometheus.io/) | Metrics |
| [Docker](https://www.docker.com/) | Containerization |

## Quick Start

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)
- [goose](https://github.com/pressly/goose#install)
- [golangci-lint](https://golangci-lint.run/welcome/install/)

### Setup

```bash
# Clone the repo
git clone <repo-url>
cd stratahq-app/backend

# Copy environment config
cp .env.example .env

# Start Postgres + Redis
make docker-up

# Run database migrations
make migrate-up

# Generate sqlc code
make generate

# Seed demo users + a scheme
make seed

# Start the server
make run
```

The server starts on `http://localhost:8080`. Verify with:

```bash
curl http://localhost:8080/healthz
# {"data":{"status":"ok"}}
```

## Project Structure

```
backend/
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── config/          # Environment variable loading & validation
│   ├── server/          # HTTP server setup, router, graceful shutdown
│   ├── middleware/       # Request middleware (auth, logging, CORS, metrics, etc.)
│   ├── auth/            # Authentication domain (JWT, login, register)
│   ├── scheme/          # Scheme management domain
│   ├── levy/            # Levy & payments domain
│   ├── maintenance/     # Maintenance requests domain
│   ├── billing/         # Stripe billing domain
│   ├── notification/    # Email notifications (Resend)
│   ├── ai/              # AI features (DeepSeek via OpenAI SDK)
│   └── platform/        # Shared infrastructure (database, cache, health, response)
├── db/
│   ├── migrations/      # Goose SQL migrations
│   ├── queries/         # sqlc query files
│   └── gen/             # Generated sqlc code (gitignored)
├── tests/
│   ├── integration/     # Integration tests (require Docker services)
│   └── fixtures/        # Test data
├── scripts/             # Development scripts
├── Dockerfile           # Multi-stage production build
├── docker-compose.yml   # Local development services
└── Makefile             # Development commands
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make run` | Run the server locally |
| `make build` | Build binary to `bin/server` |
| `make test` | Run unit tests |
| `make test-integration` | Run integration tests (requires Docker services) |
| `make test-all` | Run all tests |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code |
| `make generate` | Generate sqlc code |
| `make migrate-up` | Run pending migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-create name=<name>` | Create a new migration |
| `make migrate-status` | Show migration status |
| `make docker-up` | Start Postgres + Redis containers |
| `make docker-down` | Stop containers |
| `make docker-all` | Build and start all services |
| `make seed` | Seed the development database |

## Adding a New Domain

1. Create a new package under `internal/`:
   ```
   internal/mydomain/
   ├── handler.go    # HTTP handlers
   ├── service.go    # Business logic
   └── routes.go     # Chi route definitions
   ```

2. Add sqlc queries in `db/queries/mydomain.sql`

3. Run `make generate` to regenerate the Go code

4. Wire the handler into `internal/server/router.go`

5. Wire the service into `cmd/server/main.go`

6. Add tests next to your code (`*_test.go`) and in `tests/integration/`

## API

## Demo Seed Data

Running `make seed` creates one managing-agent admin, one trustee, one resident, and a demo scheme with representative units. The command is idempotent. Set `SEED_DEMO_PASSWORD` if you want a fixed local password; otherwise the first run generates one and prints it, and later runs keep the existing credentials unchanged.

All API routes are prefixed with `/api/v1`.

### Response Format

```json
// Success
{ "data": { ... }, "meta": { "page": 1, "per_page": 20, "total": 42 } }

// Error
{ "error": { "code": "VALIDATION_ERROR", "message": "..." } }
```

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/healthz` | No | Liveness probe |
| `GET` | `/readyz` | No | Readiness probe (checks DB + Redis) |
| `GET` | `/metrics` | No | Prometheus metrics |
| `POST` | `/api/v1/auth/register` | No | Register a new user |
| `POST` | `/api/v1/auth/login` | No | Login |
| `POST` | `/api/v1/auth/refresh` | No | Refresh access token |
| `POST` | `/api/v1/auth/logout` | No | Logout |
| `GET` | `/api/v1/schemes` | Yes | List schemes |
| `POST` | `/api/v1/schemes` | Yes | Create a scheme |
| `GET` | `/api/v1/schemes/:id` | Yes | Get a scheme |
| `PUT` | `/api/v1/schemes/:id` | Yes | Update a scheme |
| `DELETE` | `/api/v1/schemes/:id` | Yes | Delete a scheme |
| `GET` | `/api/v1/levies` | Yes | List levies |
| `POST` | `/api/v1/levies` | Yes | Create a levy |
| `GET` | `/api/v1/levies/:id` | Yes | Get a levy |
| `PUT` | `/api/v1/levies/:id` | Yes | Update a levy |
| `GET` | `/api/v1/maintenance` | Yes | List maintenance requests |
| `POST` | `/api/v1/maintenance` | Yes | Create a maintenance request |
| `GET` | `/api/v1/maintenance/:id` | Yes | Get a maintenance request |
| `PUT` | `/api/v1/maintenance/:id` | Yes | Update a maintenance request |
| `POST` | `/api/v1/billing/checkout` | Yes | Create Stripe checkout session |
| `GET` | `/api/v1/billing/subscription` | Yes | Get subscription status |
| `POST` | `/api/v1/billing/webhooks/stripe` | No | Stripe webhook |

## License

Proprietary. See LICENSE for details.
