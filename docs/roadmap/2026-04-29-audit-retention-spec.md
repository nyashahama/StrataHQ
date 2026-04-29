# Audit Retention & Cleanup — Design

## Goal
Complete the remaining 7 audit findings: data retention for sensitive financial records, hermetic test infrastructure, code quality tooling, and edge-case hardening.

## Background
19 of 26 audit findings have been addressed across Phases 0–5. The remaining findings are lower severity but collectively improve operational security, developer experience, and defense-in-depth.

## Remaining Issues

| # | Issue | Severity | Category |
|---|---|---|---|
| #107 | Raw bank CSV stored indefinitely in DB | Medium | Data retention |
| #112 | Unit tests depend on local Redis | Medium | Test infrastructure |
| #113 | npm run lint is only TypeScript | Low | Code quality |
| #114 | @anthropic-ai/sdk appears unused | Low | Dependencies |
| #115 | JSON decoder may not reject trailing tokens | Low | Input validation |
| #116 | Stripe webhook ignores future timestamps | Low | Input validation |
| #117 | /metrics endpoint is public | Low | Operations |

## Design

### 1. Bank CSV retention (#107)

**Approach:** Make `raw_csv` NULLable and clear it after successful processing. Add a background cleanup job for imports older than 90 days.

**Migration:** `00025_bank_csv_nullable.sql`
```sql
ALTER TABLE bank_statement_imports ALTER COLUMN raw_csv DROP NOT NULL;
```

**Service change:** After `ProcessBankStatementImport` commits successfully, set `raw_csv = NULL`:
```go
_, _ = s.db.Q.ClearBankStatementImportRawCsv(ctx, importUUID)
```

**Query:** Add `ClearBankStatementImportRawCsv` to `queries/levy.sql`.

### 2. Hermetic unit tests (#112)

**Approach:** Use `miniredis` (in-memory Redis mock) for tests in `internal/server/` and `internal/middleware/` that currently require a real Redis connection. Tests that truly need integration (full job queue lifecycle) stay behind `-tags=integration`.

**Files:**
- Add `miniredis/v2` dependency
- Update `internal/server/router_rate_limit_test.go` to use miniredis
- Update `internal/middleware/` rate limit tests to accept a Redis client interface

### 3. ESLint (#113)

**Approach:** Add ESLint with TypeScript, React hooks, accessibility, and import rules. Replace `npm run lint` with `eslint .`. Keep `tsc --noEmit` as a separate `typecheck` script.

**Config:** Base on `@eslint/js` + `typescript-eslint` + `eslint-plugin-react-hooks` + `eslint-plugin-jsx-a11y`.

### 4. Remove unused @anthropic-ai/sdk (#114)

**Approach:** Remove from `package.json`. If any code references it, remove those references.

### 5. JSON trailing token rejection (#115)

**Approach:** After `json.Decoder.Decode()` in `response.DecodeJSON`, check if the decoder has remaining non-whitespace bytes:
```go
if dec.More() {
    return ErrInvalidInput
}
```

### 6. Stripe future timestamp check (#116)

**Approach:** In `backend/internal/billing/provider.go`, change the timestamp check from `timestamp - tolerance` to absolute tolerance:
```go
if abs(now - timestamp) > tolerance {
    return error
}
```

### 7. /metrics protection (#117)

**Approach:** Document that deployment infrastructure (reverse proxy, firewall) should restrict `/metrics`. Add an optional `METRICS_TOKEN` env var — when set, metrics endpoint requires `Authorization: Bearer <token>`.

## Non-goals
- Moving bank CSVs to object storage (infrastructure change, separate project)
- Replacing all Redis usage with interfaces across the codebase
- Adding full snapshot/restore testing for background jobs
