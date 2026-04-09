# Codebase Stabilization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the confirmed auth/session/runtime bugs, harden the frontend-backend contract, and raise test coverage around the highest-risk flows.

**Architecture:** The first phase fixes contract and hook-order defects without changing product scope. The second phase removes the root cause behind the session inconsistencies by centralizing session shaping and separating public fetches from authenticated fetches. The last phase adds coverage around currently under-tested domains so regressions are caught automatically.

**Tech Stack:** Next.js 16, React 19, TypeScript, Vitest, Go, Chi, sqlc

---

### Task 1: Repair Session Refresh Contract

**Files:**
- Modify: `app/api/session/refresh/route.ts`
- Modify: `lib/api-contract.ts`
- Test: `lib/api-contract.test.ts`

- [ ] **Step 1: Add a failing contract test for wrapped API data**

Add a test case in `lib/api-contract.test.ts` that proves `{ "data": { "id": "u1" } }` unwraps to `{ id: "u1" }`.

Run: `npm test -- lib/api-contract.test.ts`
Expected: PASS for existing tests, then FAIL once you add a session-refresh-specific assertion that currently uses raw `res.json()`.

- [ ] **Step 2: Update session refresh to use the shared contract helper**

Change `app/api/session/refresh/route.ts` so it imports and uses `readApiData` instead of `await res.json()`, and build the session object from the unwrapped `auth/me` payload.

Key acceptance criteria:
- `session.id`, `session.email`, `session.role`, and `session.scheme_memberships` are populated from backend data.
- The route never writes an object full of `undefined` fields into `sh_session`.

- [ ] **Step 3: Add a route-focused test or helper-level assertion**

If route-handler tests are not already set up, add a helper-level assertion that exercises the same unwrap path used by the route. Prefer the smallest test that locks in the contract.

Run: `npm test`
Expected: PASS

- [ ] **Step 4: Verify the user-facing flows that depend on session refresh**

Check these flows manually after implementation:
- Update profile from `/app/[schemeId]/profile`
- Update organisation settings from `/agent/settings`
- Reload after save and confirm the user remains authenticated with the correct role and memberships

Run: `npm run lint`
Expected: PASS

### Task 2: Split Public Fetches From Authenticated Fetches

**Files:**
- Modify: `lib/api.ts`
- Modify: `app/auth/invite/[token]/page.tsx`
- Test: `lib/api-contract.test.ts`

- [ ] **Step 1: Add a failing test for public-route 401 handling**

Add a test that demonstrates the current `apiFetch()` behavior is wrong for public verification endpoints because any `401` triggers token refresh and then a redirect path.

Run: `npm test`
Expected: FAIL on the new public-route assertion

- [ ] **Step 2: Introduce an explicit public fetch path**

Implement one of these two approaches:
- Add a `publicApiFetch()` helper that never calls `refreshTokens()`
- Or add an `auth: false` option to `apiFetch()`

Use the new public path in `app/auth/invite/[token]/page.tsx` for invitation verification.

- [ ] **Step 3: Preserve current retry behavior for authenticated pages**

Keep the existing retry-on-401 behavior for dashboard pages and settings forms, but only when the caller opts into authenticated fetches.

Run: `npm test`
Expected: PASS

- [ ] **Step 4: Manually verify the invalid-invite experience**

Check:
- Invalid invite token shows the inline “Invalid invite” state
- Expired invite token does not redirect to `/auth/login`
- Valid invite token still loads and can be accepted

### Task 3: Fix Copilot Hook Ordering

**Files:**
- Modify: `components/Copilot.tsx`
- Test: `components/Copilot.test.tsx` or equivalent frontend test file

- [ ] **Step 1: Add a failing render test for auth-state transitions**

Write a test that renders `Copilot` with no user first and then with an admin or trustee user, or vice versa, and assert the component does not throw hook-order errors.

Run: `npm test`
Expected: FAIL before the hook-order fix

- [ ] **Step 2: Move all hooks above the conditional return**

Refactor `components/Copilot.tsx` so `useEffect` calls are unconditional and the role gate only affects rendered output, not hook execution order.

Implementation guardrails:
- Remove the `react-hooks/rules-of-hooks` suppressions
- Keep resident users from seeing the widget
- Keep existing chat behavior unchanged

- [ ] **Step 3: Verify interaction behavior**

Manually verify:
- Widget appears for admin/trustee users
- Widget remains hidden for residents
- Open/close, autofocus, and scroll-to-bottom still work

Run: `npm run lint`
Expected: PASS

### Task 4: Harden Session Integrity

**Files:**
- Modify: `app/api/session/route.ts`
- Modify: `middleware.ts`
- Modify: `lib/auth.tsx`
- Modify: `lib/auth-actions.ts`
- Test: `lib/backend-proxy.test.ts` or new auth/session-focused test files

- [ ] **Step 1: Stop treating `sh_session` as a trusted source of truth**

Refactor so the frontend session is either:
- Rebuilt from authoritative backend data, or
- Signed/verified before use

At minimum, `app/api/session/route.ts` must reject malformed or incomplete session payloads instead of blindly returning them.

- [ ] **Step 2: Tighten middleware semantics**

Replace the current “cookie exists” protection in `middleware.ts` with a narrower rule:
- Protected pages require a plausible authenticated state
- Public routes remain public
- API routes are not blanket-whitelisted unless explicitly intended

- [ ] **Step 3: Add explicit auth bootstrap behavior**

Update `lib/auth.tsx` so app startup handles stale/expired auth more predictably. The target behavior is:
- No false-positive logged-in UI based on stale `sh_session`
- Clean transition to login when access/refresh is no longer valid

- [ ] **Step 4: Verify full auth lifecycle**

Check:
- Login
- Token expiry followed by authenticated API retry
- Logout
- Reload after profile update
- Reload after onboarding setup

Run: `npm run lint`
Expected: PASS

### Task 5: Expand Test Coverage Around Untested Backend Domains

**Files:**
- Modify: `backend/internal/billing/*.go`
- Modify: `backend/internal/communications/*.go`
- Modify: `backend/internal/compliance/*.go`
- Modify: `backend/internal/financials/*.go`
- Modify: `backend/internal/levy/*.go`
- Modify: `backend/internal/maintenance/*.go`
- Modify: `backend/internal/scheme/*.go`
- Test: new `*_test.go` files next to each service or handler

- [ ] **Step 1: Add table-driven authorization tests for each service**

Cover at least:
- admin access to in-org resources
- member access to in-scheme resources
- forbidden access to foreign-org or foreign-scheme resources
- invalid UUID input returning the correct domain error

- [ ] **Step 2: Add regression tests for the confirmed auth/session bugs where backend helpers are involved**

Prioritize:
- invitation verify/accept flows
- profile and organisation update flows
- billing subscription access

- [ ] **Step 3: Run backend unit tests**

Run: `cd backend && go test ./...`
Expected: PASS

- [ ] **Step 4: Run integration tests for critical flows when local services are available**

Run: `cd backend && make test-integration`
Expected: PASS

### Task 6: Clean Up Architectural Duplication

**Files:**
- Modify: `backend/internal/server/router.go`
- Modify: `backend/internal/auth/routes.go`
- Modify: shared frontend fetch/session helpers as needed

- [ ] **Step 1: Remove or consolidate duplicated route wiring**

The backend currently has both explicit protected auth route wiring in `router.go` and a `ProtectedRoutes()` helper in `backend/internal/auth/routes.go`. Consolidate the pattern so auth routes are defined in one place.

- [ ] **Step 2: Consolidate session shaping**

Define one helper that maps backend `MeResponse` to the frontend `SessionUser` shape. Use that helper from:
- login/register post-auth flows
- onboarding updates
- `/api/session/refresh`

- [ ] **Step 3: Run final verification**

Run:
- `npm run lint`
- `npm test`
- `cd backend && go test ./...`

Expected:
- all commands PASS
- no remaining hook-rule suppressions in `components/Copilot.tsx`
- no direct `await res.json()` calls against wrapped backend responses in frontend server routes

---

**Self-review notes**

Spec coverage:
- Confirmed bugs are covered in Tasks 1-4.
- Structural improvements and test gaps are covered in Tasks 5-6.

Placeholder scan:
- No `TODO`, `TBD`, or “handle appropriately” placeholders remain.

Type consistency:
- Session refresh, auth bootstrap, and contract unwrapping all use the same `SessionUser` / backend response mapping language.
