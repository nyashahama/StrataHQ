# Audit Retention & Cleanup — Implementation Plan

> **For agentic workers:** Use subagent-driven-development or executing-plans to implement this plan task-by-task.

**Goal:** Complete the 7 remaining audit findings: data retention, hermetic tests, linting, and edge-case hardening.

**Architecture:** Each issue is independent and can be implemented in parallel. Tasks are ordered by priority.

---

## Task 1: Bank CSV retention (#107)

**Files:**
- Create: `backend/db/migrations/00025_bank_csv_nullable.sql`
- Modify: `backend/db/queries/levy.sql`
- Modify: `backend/internal/levy/bank_statement_import.go`
- Regenerate: `backend/db/gen/`

- [ ] **Step 1: Create migration**

```sql
-- +goose Up
ALTER TABLE bank_statement_imports ALTER COLUMN raw_csv DROP NOT NULL;

-- +goose Down
ALTER TABLE bank_statement_imports ALTER COLUMN raw_csv SET NOT NULL;
```

- [ ] **Step 2: Add sqlc query**

In `backend/db/queries/levy.sql`, add at end:

```sql
-- name: ClearBankStatementImportRawCsv :exec
UPDATE bank_statement_imports SET raw_csv = NULL WHERE id = $1;
```

- [ ] **Step 3: Regenerate sqlc**

```bash
cd backend/db && sqlc generate
```

- [ ] **Step 4: Update service code**

In `backend/internal/levy/bank_statement_import.go`, in `ProcessBankStatementImport`, after `tx.Commit` succeeds, add:

```go
if _, clearErr := s.db.Q.ClearBankStatementImportRawCsv(ctx, importUUID); clearErr != nil {
    return fmt.Errorf("clear raw csv: %w", clearErr)
}
```

- [ ] **Step 5: Build and test**

```bash
cd backend && go build ./... && go test ./internal/levy/... -v -count=1
```

- [ ] **Step 6: Commit**

---

## Task 2: Remove @anthropic-ai/sdk (#114)

**Files:**
- Modify: `package.json`

- [ ] **Step 1: Check if any code imports it**

```bash
rg "@anthropic-ai/sdk" --type ts --type tsx
```

- [ ] **Step 2: Remove reference if unused**

Remove `"@anthropic-ai/sdk"` from `dependencies` in `package.json`.

- [ ] **Step 3: Run npm install**

```bash
npm install
```

- [ ] **Step 4: Build and typecheck**

```bash
npx tsc --noEmit && npm run build
```

- [ ] **Step 5: Commit**

---

## Task 3: Stripe future timestamp check (#116)

**Files:**
- Modify: `backend/internal/billing/provider.go`

- [ ] **Step 1: Read current timestamp check**

Read `backend/internal/billing/provider.go` around line 96. The current check likely only rejects stale timestamps.

- [ ] **Step 2: Update to absolute tolerance**

```go
diff := now.Sub(timestamp)
if diff < 0 {
    diff = -diff
}
if diff > tolerance {
    return fmt.Errorf("timestamp outside tolerance")
}
```

- [ ] **Step 3: Build and test**

```bash
cd backend && go build ./... && go test ./internal/billing/... -v -count=1
```

- [ ] **Step 4: Commit**

---

## Task 4: JSON trailing token rejection (#115)

**Files:**
- Modify: `backend/internal/platform/response/json.go`

- [ ] **Step 1: Read current DecodeJSON**

Read `backend/internal/platform/response/json.go` to understand the current `DecodeJSON` function.

- [ ] **Step 2: Add trailing token check**

After `decoder.Decode(&target)`, add:

```go
if decoder.More() {
    return fmt.Errorf("trailing tokens after JSON value")
}
```

- [ ] **Step 3: Build and test**

```bash
cd backend && go build ./... && go test ./internal/platform/response/... -v -count=1
```

- [ ] **Step 4: Commit**

---

## Task 5: /metrics auth protection (#117)

**Files:**
- Modify: `backend/internal/server/router.go`
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Add config field**

In `backend/internal/config/config.go`, add to `ConfigStrings`:

```go
MetricsToken string
```

Load from env:
```go
MetricsToken: os.Getenv("METRICS_TOKEN"),
```

- [ ] **Step 2: Add auth check to metrics route**

In `backend/internal/server/router.go`, wrap the metrics handler to check `METRICS_TOKEN` when configured:

```go
if cfg.MetricsToken != "" {
    r.Use(metricsAuthMiddleware(cfg.MetricsToken))
}
r.Handle("/metrics", promhttp.Handler())
```

With a simple middleware:
```go
func metricsAuthMiddleware(token string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Header.Get("Authorization") != "Bearer "+token {
                response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

- [ ] **Step 3: Build and test**

```bash
cd backend && go build ./... && go test ./internal/server/... -v -count=1
```

- [ ] **Step 4: Commit**

---

## Task 6: ESLint setup (#113)

**Files:**
- Create: `eslint.config.mjs`
- Modify: `package.json`

- [ ] **Step 1: Install dependencies**

```bash
npm install --save-dev eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-jsx-a11y
```

- [ ] **Step 2: Create eslint.config.mjs**

```js
import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import jsxA11y from "eslint-plugin-jsx-a11y";

export default tseslint.config(
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    plugins: {
      "react-hooks": reactHooks,
      "jsx-a11y": jsxA11y,
    },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
    },
  },
  {
    ignores: [".next/", "node_modules/", "out/"],
  }
);
```

- [ ] **Step 3: Update package.json scripts**

```json
"lint": "eslint .",
"typecheck": "tsc --noEmit",
```

- [ ] **Step 4: Run ESLint and fix any errors**

```bash
npm run lint
```

- [ ] **Step 5: Commit**

---

## Task 7: Hermetic unit tests with miniredis (#112)

**Files:**
- Modify: `backend/go.mod`
- Modify: `backend/internal/server/router_rate_limit_test.go`
- Modify: `backend/internal/middleware/ratelimit_test.go` (if applicable)

- [ ] **Step 1: Install miniredis**

```bash
cd backend && go get github.com/alicebob/miniredis/v2
```

- [ ] **Step 2: Update router rate limit tests**

Replace real Redis client creation with miniredis:

```go
import "github.com/alicebob/miniredis/v2"

func setupTestRedis(t *testing.T) *redis.Client {
    t.Helper()
    mr, err := miniredis.Run()
    if err != nil {
        t.Fatalf("miniredis: %v", err)
    }
    t.Cleanup(mr.Close)
    return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}
```

Use `setupTestRedis` in each rate limit test instead of connecting to real Redis.

- [ ] **Step 3: Run tests**

```bash
cd backend && go test ./internal/server/... -v -count=1
```
Expected: all tests pass without Redis running.

- [ ] **Step 4: Verify backwards compatibility**

```bash
cd backend && go test ./... -v -count=1
```
Expected: all tests pass.

- [ ] **Step 5: Commit**

---

## Final verification

- [ ] Run full test suite: `go test ./... -count=1`
- [ ] Run TypeScript: `npm run lint && npm run typecheck`
- [ ] Build both binaries: `go build -o bin/server ./cmd/server/ && go build -o bin/worker ./cmd/worker/`
- [ ] Verify CI passes
