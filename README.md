# StrataHQ

Property management software for South African sectional title schemes. Built for managing agents and trustees who run body corporates under the Sectional Titles Schemes Management Act (STSMA).

## What It Does

- **Levy management** — create levy periods, track payments, reconcile bank statements
- **Maintenance** — log and track maintenance requests from submission to resolution
- **AGM & voting** — manage meetings, resolutions, and proxy assignments
- **Document vault** — store and access scheme documents
- **Financial reporting** — levy collection rates, expenditure tracking
- **Communications** — send notices and updates to owners and residents

## Monorepo Structure

```
stratahq-app/
├── app/                 # Next.js App Router — frontend
│   ├── app/[schemeId]/  # Scheme-scoped views (levy, maintenance, AGM, etc.)
│   ├── agent/           # Managing agent portfolio views
│   ├── auth/            # Login, register, onboarding
│   └── api/             # Next.js API routes (AI copilot, etc.)
├── backend/             # Go REST API
│   ├── cmd/server/      # Server entrypoint
│   ├── internal/        # Domain packages (auth, scheme, levy, maintenance, billing)
│   ├── db/              # Migrations, sqlc queries
│   └── README.md        # Backend-specific docs
├── components/          # Shared React components
├── hooks/               # Shared React hooks
├── lib/                 # Shared utilities
└── docs/                # Architecture specs and implementation plans
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js 16 · React 19 · TypeScript · Tailwind CSS |
| Backend | Go 1.24 · Chi · PostgreSQL 17 · pgx/v5 · sqlc · goose |
| Caching | Redis 7 |
| Auth | JWT (self-issued, Go backend) |
| Payments | Stripe (hybrid — Stripe.js frontend + webhooks backend) |
| Email | Resend |
| AI | DeepSeek (OpenAI-compatible) |
| Deployment | Vercel (frontend) · Render (backend) |

## Getting Started

### Frontend

```bash
npm install
cp .env.example .env.local   # fill in values
npm run dev
```

Frontend runs at `http://localhost:3000`.

### Backend

```bash
cd backend
cp .env.example .env         # fill in values
make docker-up               # start Postgres + Redis
make migrate-up              # run migrations
make generate                # generate sqlc code
make run                     # start Go server
```

Backend runs at `http://localhost:8080`. See [`backend/README.md`](backend/README.md) for full docs.

## Environment Variables

### Frontend (`.env.local`)

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Backend (`backend/.env`)

See [`backend/.env.example`](backend/.env.example) for all required variables.

## Development

### Running both services locally

```bash
# Terminal 1 — backend
cd backend && make docker-up && make run

# Terminal 2 — frontend
npm run dev
```

### Running tests

```bash
# Frontend
npm run lint

# Backend
cd backend && make test
cd backend && make test-integration   # requires Docker
```

## Docs

- [`docs/superpowers/specs/`](docs/superpowers/specs/) — architecture and feature design specs
- [`docs/superpowers/plans/`](docs/superpowers/plans/) — implementation plans
- [`backend/README.md`](backend/README.md) — Go backend documentation
- [`backend/CONTRIBUTING.md`](backend/CONTRIBUTING.md) — contributing guide
- [`TODOS.md`](TODOS.md) — deferred V2/V3 feature backlog

## License

Proprietary.
