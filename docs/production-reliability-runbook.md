# Production Reliability Runbook

This document defines the production readiness gates for StrataHQ, focusing on load testing and performance thresholds.

## Load Testing

### Tool: k6

Load tests are written in JavaScript using [k6](https://k6.io/docs/). Install k6 locally:

```bash
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
k6 run --vus 10 --duration 2m backend/tests/load/auth-and-dashboard.js

# 25 concurrent users
k6 run --vus 25 --duration 2m backend/tests/load/auth-and-dashboard.js

# 50 concurrent users
k6 run --vus 50 --duration 2m backend/tests/load/auth-and-dashboard.js
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
```

## Monitoring

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

## Contact

For issues during load testing, contact the engineering team.