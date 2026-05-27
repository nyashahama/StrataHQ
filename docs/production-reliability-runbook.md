# Production Reliability Runbook

This document defines the production readiness gates for StrataHQ, focusing on load testing and performance thresholds.

## Load Testing

### Tool: k6

Load tests are written in JavaScript using [k6](https://k6.io/docs/). Use the
Docker image when k6 is not installed locally, or install k6 on the workstation:

```bash
# Docker, from the repository root
docker run --rm -i \
  --network host \
  -v "$PWD:/work" \
  grafana/k6 run \
    -e BASE_URL=http://localhost:8080 \
    -e TEST_EMAIL=demo@stratahq.com \
    -e TEST_PASSWORD=Demo2024! \
    -e LOAD_VUS=10 \
    -e LOAD_DURATION=2m \
    /work/backend/tests/load/auth-and-dashboard.js

# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver keyserver.ubuntu.com --recv-keys C5AD17C747E3415A3642ECE57ADBB1FFD6F88F84
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6
```

### Test Script: auth-and-dashboard.js

The load test script is located at `backend/tests/load/auth-and-dashboard.js`.
Each invocation runs one load level. Set `LOAD_VUS` and `LOAD_DURATION` for the
target load. The script uses the documented seeded user by default; set
`UNIQUE_TEST_USERS=true` only when the target database has matching plus-address
users seeded for every VU.

It exercises:
- Login endpoint (`POST /api/v1/auth/login`)
- Token refresh endpoint (`POST /api/v1/auth/refresh`)
- Protected GET endpoints:
  - `GET /api/v1/auth/me`
  - `GET /api/v1/schemes`
  - `GET /api/v1/levies`
  - `GET /api/v1/maintenance`
- Token expiry scenario (repeat login + refresh)

### Running Load Tests

#### Prerequisites

1. The backend server must be running.
2. Test credentials must exist in the database:
   ```bash
   # From backend directory
   make seed  # creates demo@stratahq.com with password Demo2024!
   ```

3. Set environment variables:
   ```bash
   export BASE_URL=http://localhost:8080
   export TEST_EMAIL=demo@stratahq.com
   export TEST_PASSWORD=Demo2024!
   ```

#### Execute Load Tests

```bash
# 10 concurrent users
LOAD_VUS=10 LOAD_DURATION=2m k6 run backend/tests/load/auth-and-dashboard.js

# 25 concurrent users
LOAD_VUS=25 LOAD_DURATION=2m k6 run backend/tests/load/auth-and-dashboard.js

# 50 concurrent users
LOAD_VUS=50 LOAD_DURATION=2m k6 run backend/tests/load/auth-and-dashboard.js
```

Docker equivalent:

```bash
for vus in 10 25 50; do
  docker run --rm -i \
    --network host \
    -v "$PWD:/work" \
    grafana/k6 run \
      -e BASE_URL="$BASE_URL" \
      -e TEST_EMAIL="$TEST_EMAIL" \
      -e TEST_PASSWORD="$TEST_PASSWORD" \
      -e LOAD_VUS="$vus" \
      -e LOAD_DURATION=2m \
      /work/backend/tests/load/auth-and-dashboard.js
done
```

### Pass/Fail Thresholds

The following thresholds MUST be met for production readiness:

| Metric | Threshold | Target |
|--------|----------|--------|
| Login first-attempt success | > 99% | Authenticated users can login on first try |
| Refresh requests blocked by rate limit | 0 | Zero 429 responses on `/auth/refresh` |
| Scripted repeat-login failures | 0 | No repeated login failures |
| GET endpoint failures during load | < 1% | No blank states on protected endpoints |
| HTTP request p95 latency | < 1000ms | 95% of requests complete in under 1 second |

### Thresholds Definition in k6

The thresholds are embedded in the test script:

```javascript
thresholds: {
  login_success: ['rate>0.99'],        // >99% login success
  refresh_blocked: ['rate==0'],       // 0 rate limit blocks
  repeat_login_fail: ['rate==0'],       // 0 repeat failures
  get_endpoints_fail: ['rate<0.01'],   // <1% GET failures
  http_req_duration: ['p(95)<1000'],  // p95 < 1s
},
```

### Expected Results by Scenario

| Scenario | VUs | Expected p95 | Pass/Fail |
|----------|-----|--------------|-----------|
| 10 users | 10 | < 500ms | Must PASS |
| 25 users | 25 | < 750ms | Must PASS |
| 50 users | 50 | < 1000ms | Must PASS |

If 50 users fail, tune infrastructure (see Tuning section below). Run 10 and 25 user tests first to establish baseline.

## Tuning (Only After Evidence)

DO NOT tune before running load tests. If metrics indicate pressure, tune in this order:

### 1. Database Connection Pool

File: `backend/internal/platform/database/database.go`

```go
cfg.MaxConns = 10     // Increase first (e.g., 25)
cfg.MinConns = 2      // Increase (e.g., 5)
```

### 2. Per-Route Rate Limits

File: `backend/internal/server/router.go`

```go
// Auth login rate limit (5 requests/minute → increase if needed)
r.With(middleware.PerEndpointRateLimit(rdb, "auth-login", 10, 1*time.Minute)).Post("/login", h.Auth.Login)

// Auth refresh rate limit (30 requests/minute → increase if needed)
r.With(middleware.PerEndpointRateLimit(rdb, "auth-refresh", 60, 1*time.Minute)).Post("/refresh", h.Auth.Refresh)
```

### 3. Dashboard Aggregation (Optional)

If `/levies/:schemeId` or `/schemes` endpoints are slow under load:

- Consider adding Redis caching for frequently accessed dashboard data
- Aggregate queries at the service layer rather than per-request

## Verification Commands

Run these before deploying to staging:

```bash
# Frontend lint
npm run lint

# Frontend tests
npm test

# Backend unit tests
cd backend && make test

# Backend integration tests (requires Docker)
cd backend && make test-integration

# Backend observability/security focused tests
cd backend && go test ./internal/middleware ./internal/platform/response ./internal/server ./internal/billing ./internal/whatsapp ./internal/security
```

## Launch Gate

Run this full matrix for release approval:

```bash
npm run lint
npm test
npm run build
cd backend && go test ./internal/... -v -race
cd backend && go test ./tests/integration/... -v -race -tags=integration
LOAD_VUS=10 LOAD_DURATION=2m k6 run backend/tests/load/auth-and-dashboard.js
LOAD_VUS=25 LOAD_DURATION=2m k6 run backend/tests/load/auth-and-dashboard.js
LOAD_VUS=50 LOAD_DURATION=2m k6 run backend/tests/load/auth-and-dashboard.js
```

For production or staging sign-off, prefer the manual GitHub Actions workflow
`Production Launch Gate`. It records the health checks and 10/25/50-user k6
runs in Actions logs without exposing credentials on a workstation.

Required workflow inputs:

- `backend_base_url` - deployed backend URL, without a trailing slash
- `frontend_health_url` - optional deployed frontend `/api/health` URL
- `load_duration` - k6 duration for each load level, default `2m`

Required repository or environment secrets:

- `LAUNCH_GATE_TEST_EMAIL`
- `LAUNCH_GATE_TEST_PASSWORD`

Optional secret:

- `VERCEL_AUTOMATION_BYPASS_SECRET` - used only for protected Vercel frontend
  health checks

The launch gate runs from a single GitHub Actions runner IP. If the deployed
backend keeps the conservative defaults (`AUTH_LOGIN_RATE_LIMIT=5` and
`AUTH_REFRESH_RATE_LIMIT=30` per minute per IP), the 10/25/50-user auth load
matrix can hit auth throttles before measuring application capacity. For
staging or production launch-gate windows, set these backend environment
variables high enough for the matrix, then restore stricter values if desired:

```bash
AUTH_LOGIN_RATE_LIMIT=240
AUTH_REFRESH_RATE_LIMIT=480
```

After the aggregated portfolio summary query is deployed, include a dedicated pass through `/agent` portfolio overview during staging verification to confirm summary counts and collection percentage remain correct under load.

## Monitoring

### Request Correlation

Every backend response should include `X-Request-ID`.

Verification:

```bash
curl -i "$BASE_URL/healthz" | grep -i "x-request-id"
```

For API errors, JSON responses from request-aware handlers include:

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "missing authorization header",
    "requestId": "..."
  }
}
```

Support workflow:

1. Ask the user for the request ID shown in the UI or browser network response.
2. Search application logs for `request_id`.
3. Correlate route, status, actor, and provider failures.

### Security and Abuse Alerts

Alert on:

- Elevated `401` or `403` rate on protected API routes.
- Elevated `429` rate on `/api/v1/auth/login`, `/api/v1/auth/refresh`, `/api/v1/ai/copilot`, `/api/v1/billing/webhooks/stripe`, and `/api/v1/whatsapp/webhooks`.
- Stripe webhook signature failures.
- Twilio webhook signature failures.
- Audit event recorder failures.
- Sudden increase in `5xx` errors for billing, documents, levy reconciliation, AGM voting, or auth.

Initial action:

1. Check `/healthz`, `/readyz`, and `/metrics`.
2. Filter logs by `request_id` or route pattern.
3. Compare `http_requests_total` status labels before and after the alert.
4. Confirm provider status pages for Stripe, Twilio, Resend, and the AI provider.

### Resource Audit Log

Business audit records live in `resource_audit_events`; request telemetry remains in `audit_events`.

Verify audit writes after staging deploy:

```bash
psql "$DATABASE_URL" -c "select action, resource_type, occurred_at from resource_audit_events order by occurred_at desc limit 10;"
```

Minimum actions expected during smoke testing:

- `document.uploaded`
- `document.deleted`
- `levy_period.created`
- `levy.reconciled`
- `collection_event.reminder_sent`
- `agm_meeting.scheduled`
- `agm.vote_cast`
- `agm.proxy_assigned`

### Prometheus Metrics

The backend exposes Prometheus metrics at `/metrics`. Key metrics for load testing:

- `http_request_duration_seconds_bucket` — request latency histograms
- `http_requests_total` — request counters by endpoint and status
- `rate_limit_blocked_total` — rate limit blocks by endpoint

### Health Checks

- `/healthz` — liveness probe (always returns 200)
- `/readyz` — readiness probe (checks DB + Redis connectivity)

## Rollback Plan

If production load tests fail after tuning:

1. Revert database pool settings to defaults
2. Revert rate limits to conservative values
3. Scale horizontally (add more backend instances) before tuning vertically
4. Enable Redis caching for expensive queries

## Background Worker Operations

The background worker runs `go run ./cmd/worker/` in development and the `bin/worker` binary in deployed environments. It processes durable PostgreSQL jobs from `background_jobs`.

### Required Environment

The worker requires the same core provider and database variables as the API:

- `DATABASE_URL`
- `REDIS_URL`
- `JWT_SECRET`
- `RESEND_API_KEY`
- `AI_BASE_URL`
- `AI_API_KEY`
- `AI_MODEL`
- `APP_BASE_URL`
- `EMAIL_FROM`
- `TWILIO_ACCOUNT_SID`
- `TWILIO_AUTH_TOKEN`
- `TWILIO_WHATSAPP_NUMBER`

Worker tuning variables:

- `WORKER_POLL_INTERVAL`: default `2s`
- `WORKER_LEASE_TTL`: default `5m`
- `WORKER_BATCH_SIZE`: default `10`
- `WORKER_MAX_ATTEMPTS`: default `5`

### Collection Reminder Delivery

The API records collection reminders with delivery status `queued` and inserts one job per enabled channel:

- `collection_reminder_email`
- `collection_reminder_whatsapp`

The worker sends the provider request and then updates `collection_events.email_status` or `collection_events.whatsapp_status` to `sent` or `failed`.

### Retry Behavior

Transient provider failures are retried with exponential backoff starting at 30 seconds and capped at 5 minutes. A job moves to `failed` after its configured max attempts.

### Operator Queries

Queued or running jobs:

```sql
SELECT kind, status, count(*)
FROM background_jobs
WHERE status IN ('queued', 'running')
GROUP BY kind, status
ORDER BY kind, status;
```

Failed jobs:

```sql
SELECT id, kind, attempts, max_attempts, last_error, failed_at
FROM background_jobs
WHERE status = 'failed'
ORDER BY failed_at DESC
LIMIT 50;
```

Stale running jobs:

```sql
SELECT id, kind, locked_by, locked_at, now() - locked_at AS locked_for
FROM background_jobs
WHERE status = 'running'
  AND locked_at < now() - interval '5 minutes'
ORDER BY locked_at ASC;
```

### Alerts

Alert on:

- Any `failed` jobs in the last 15 minutes.
- More than 100 queued jobs for a single kind.
- Any `running` job locked longer than `WORKER_LEASE_TTL`.
- More than 5 consecutive worker iteration errors in logs.

### Verification Commands

```bash
cd backend && make generate
cd backend && go test ./internal/jobs ./internal/levy ./internal/config -count=1
cd backend && go build ./cmd/server ./cmd/worker
cd backend && make test
```

## Contact

For issues during load testing, contact the engineering team.
