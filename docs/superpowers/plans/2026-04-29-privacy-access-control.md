# Phase 2 — Privacy and Access Control Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Filter sensitive unit/owner data from resident API responses and add role-based document visibility controls.

**Architecture:** Add role-checks in existing service methods (`Get`, `ListUnits`, `ListDocuments`) rather than new endpoints. Add a `visibility` column to `scheme_documents` with a migration. Keep trustees seeing all scheme data; only residents get filtered.

**Tech Stack:** Go, pgx, goose migrations, sqlc queries, chi router, existing integration test framework with `-tags=integration`

---

### Task 1: Filter units for residents in scheme detail `Get`

**Files:**
- Modify: `backend/internal/scheme/service.go:223-241`

- [ ] **Step 1: Read current `Get` method to confirm boundaries**

Read `backend/internal/scheme/service.go` lines 212-256 to confirm the exact code to modify.

- [ ] **Step 2: Add resident filtering for Units**

After `units, err := s.db.Q.ListUnitsByScheme(ctx, scheme.ID)` on line 223, and before `detail := &SchemeDetail{` on line 233, insert the filtering block:

```go
	if !auth.IsAdminRole(role) && !auth.IsTrusteeRole(role) && unitID != nil {
		filtered := make([]dbgen.Unit, 0, 1)
		for _, u := range units {
			if u.ID == *unitID {
				filtered = append(filtered, u)
				break
			}
		}
		units = filtered
	}
```

This uses `!auth.IsAdminRole(role) && !auth.IsTrusteeRole(role)` — effectively `auth.IsResidentRole(role)`. Only residents get filtered; admins and trustees see all units.

- [ ] **Step 3: Verify `IsTrusteeRole` exists**

Check `backend/internal/auth/roles.go` — if `IsTrusteeRole` doesn't exist, we can use `role == "resident"` or check if `IsResidentRole` exists and invert the check.

Run: `grep -n "IsTrusteeRole\|IsResidentRole\|IsAdminRole" backend/internal/auth/roles.go`

- [ ] **Step 4: Build and vet**

```bash
cd backend && go build ./internal/scheme/... && go vet ./internal/scheme/...
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/scheme/service.go
git commit -m "fix: filter scheme units to resident's own unit in Get"
```

---

### Task 2: Filter units for residents in `ListUnits` endpoint

**Files:**
- Modify: `backend/internal/scheme/service.go:365-380`

- [ ] **Step 1: Read current `ListUnits` method**

Read lines 365-400 to understand the current implementation.

- [ ] **Step 2: Add the same filtering logic**

After the `ListUnitsByScheme` call, insert the same filtering block from Task 1:

```go
	units, err := s.db.Q.ListUnitsByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}

	if !auth.IsAdminRole(role) && !auth.IsTrusteeRole(role) && unitID != nil {
		filtered := make([]dbgen.Unit, 0, 1)
		for _, u := range units {
			if u.ID == *unitID {
				filtered = append(filtered, u)
				break
			}
		}
		units = filtered
	}
```

- [ ] **Step 3: Build and vet**

```bash
cd backend && go build ./internal/scheme/... && go vet ./internal/scheme/...
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/scheme/service.go
git commit -m "fix: filter units to resident's own unit in ListUnits"
```

---

### Task 3: Add document visibility migration

**Files:**
- Create: `backend/db/migrations/00024_document_visibility.sql`
- Modify: `backend/db/queries/documents.sql` (add role-filtered list query)
- Regenerate: `backend/db/gen/*` (via sqlc generate)

- [ ] **Step 1: Create migration file**

Create `backend/db/migrations/00024_document_visibility.sql`:

```sql
-- +goose Up

CREATE TYPE document_visibility AS ENUM ('all', 'trustee', 'admin');

ALTER TABLE scheme_documents
    ADD COLUMN visibility document_visibility NOT NULL DEFAULT 'all';

-- +goose Down

ALTER TABLE scheme_documents
    DROP COLUMN IF EXISTS visibility;

DROP TYPE IF EXISTS document_visibility;
```

- [ ] **Step 2: Add filtered list query to documents.sql**

Read `backend/db/queries/documents.sql` first to understand existing queries. Then add:

```sql
-- name: ListDocumentsBySchemeAndVisibility :many
SELECT * FROM scheme_documents
WHERE scheme_id = $1 AND visibility = ANY($2::document_visibility[])
ORDER BY created_at DESC;
```

- [ ] **Step 3: Regenerate sqlc**

```bash
cd backend/db && sqlc generate
```

- [ ] **Step 4: Build and vet**

```bash
cd backend && go build ./... && go vet ./internal/documents/...
```

- [ ] **Step 5: Commit**

```bash
git add backend/db/migrations/00024_document_visibility.sql backend/db/queries/documents.sql backend/db/gen/
git commit -m "feat: add document visibility column and role-filtered query"
```

---

### Task 4: Apply document visibility filtering in service layer

**Files:**
- Modify: `backend/internal/documents/service.go:77-117` (List method)

- [ ] **Step 1: Read current `List` method**

Read `backend/internal/documents/service.go` lines 77-117.

- [ ] **Step 2: Add role-based visibility check in `resolveAccess`**

After `resolveAccess` returns and before the document query, determine the allowed visibilities:

```go
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	visibilities := determineVisibilities(access.role)
	docs, err := s.db.Q.ListDocumentsBySchemeAndVisibility(ctx, dbgen.ListDocumentsBySchemeAndVisibilityParams{
		SchemeID:     scheme.ID,
		Column2:      visibilities,
	})
```

Add the helper function at the bottom of the file:

```go
func determineVisibilities(role string) []dbgen.DocumentVisibility {
	switch {
	case auth.IsAdminRole(role):
		return []dbgen.DocumentVisibility{
			dbgen.DocumentVisibilityAll,
			dbgen.DocumentVisibilityTrustee,
			dbgen.DocumentVisibilityAdmin,
		}
	case role == string(auth.RoleTrustee):
		return []dbgen.DocumentVisibility{
			dbgen.DocumentVisibilityAll,
			dbgen.DocumentVisibilityTrustee,
		}
	default:
		return []dbgen.DocumentVisibility{
			dbgen.DocumentVisibilityAll,
		}
	}
}
```

Wait — need to check how `ListDocumentsBySchemeAndVisibility` params are generated by sqlc. The sqlc parameter for `ANY($2::document_visibility[])` will be a typed array. Read the generated code to confirm the param type.

- [ ] **Step 3: Update `Create` to default visibility to `admin`**

In the `Create` method (line 119), add `Visibility: dbgen.DocumentVisibilityAdmin` to the create params. This ensures new documents are admin-only by default. The frontend should provide a visibility field in the create form.

- [ ] **Step 4: Build and vet**

```bash
cd backend && go build ./... && go vet ./internal/documents/...
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/documents/service.go
git commit -m "fix: filter document listing by role-based visibility"
```

---

### Task 5: Resident integration tests — scheme detail and units

**Files:**
- Modify: `backend/tests/integration/scheme_test.go`

- [ ] **Step 1: Read test helpers in `testhelpers_test.go`**

Read lines 106-130 to understand `withAuthContext`, `withNonAdminContext`, `withOrgRoleContext`.

- [ ] **Step 2: Check existing `setupResident` helper**

Search for helper functions that create non-admin identities:

```bash
grep -n "setupResident\|createResident\|withResident" backend/tests/integration/*_test.go
```

If none exists, we'll use `auth.GenerateAccessToken` directly to create a resident token.

- [ ] **Step 3: Add test at end of `scheme_test.go`**

```go
func TestScheme_ResidentDetailSeesOnlyOwnUnit(t *testing.T) {
	accessToken, orgID := setupAgent(t)
	claims, err := auth.ValidateAccessToken(accessToken, testJWTSigningKey)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	schemeID := setupScheme(t, accessToken)

	ctx := context.Background()
	unitA, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "1A", "Alice Adams"))
	if err != nil {
		t.Fatalf("create unit A: %v", err)
	}
	unitB, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "2B", "Bob Brown"))
	if err != nil {
		t.Fatalf("create unit B: %v", err)
	}

	// Create a resident user and add them to unit A
	residentEmail := uniqueEmail(t)
	residentPassword := "test-resident-pass-1"
	residentUser, err := createResidentMember(ctx, t, schemeID, unitA.ID, residentEmail, residentPassword)
	if err != nil {
		t.Fatalf("create resident member: %v", err)
	}

	residentToken, err := auth.GenerateAccessToken(
		residentUser.ID.String(), orgID, "resident",
		"http://localhost:3000", "stratahq-api",
		testJWTSigningKey, 15*time.Minute,
	)
	if err != nil {
		t.Fatalf("generate resident token: %v", err)
	}

	h := newSchemeHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/schemes/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"id": schemeID})
	req = withAuthContext(req, residentToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Get(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident detail: status=%d body=%s", w.Code, w.Body)
	}
	detail := decodeSuccess[scheme.SchemeDetail](t, w)

	if len(detail.Units) != 1 {
		t.Fatalf("resident sees %d units, want 1", len(detail.Units))
	}
	if detail.Units[0].Identifier != "1A" {
		t.Fatalf("resident sees unit %q, want 1A", detail.Units[0].Identifier)
	}
	if detail.Units[0].OwnerName != "Alice Adams" {
		t.Fatalf("resident sees owner %q, want Alice Adams", detail.Units[0].OwnerName)
	}

	// Trustee should see both units
	trusteeEmail := uniqueEmail(t)
	trusteeUser, err := createTrusteeMember(ctx, t, schemeID, trusteeEmail)
	if err != nil {
		t.Fatalf("create trustee: %v", err)
	}
	trusteeToken, err := auth.GenerateAccessToken(
		trusteeUser.ID.String(), orgID, "trustee",
		"http://localhost:3000", "stratahq-api",
		testJWTSigningKey, 15*time.Minute,
	)
	if err != nil {
		t.Fatalf("generate trustee token: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/schemes/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"id": schemeID})
	req = withAuthContext(req, trusteeToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Get(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("trustee detail: status=%d body=%s", w.Code, w.Body)
	}
	trusteeDetail := decodeSuccess[scheme.SchemeDetail](t, w)
	if len(trusteeDetail.Units) != 2 {
		t.Fatalf("trustee sees %d units, want 2", len(trusteeDetail.Units))
	}
}
```

But this requires `createResidentMember` and `createTrusteeMember` helpers. Let me check what's already available in the test helpers.

Actually, looking at members_test.go, there's already a `createMemberRecord` helper. Use `testQ.UpsertSchemeMembership` directly.

Simpler approach — create resident via the registration + invitation flow, or directly insert into DB:

```go
func createMemberWithRole(ctx context.Context, t *testing.T, schemeID uuid.UUID, unitID *uuid.UUID, role string) (dbgen.User, error) {
	t.Helper()
	email := uniqueEmail(t)
	user, err := testQ.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        email,
		PasswordHash: "test-hash",
		FullName:     role + " Member",
	})
	if err != nil {
		return dbgen.User{}, err
	}
	var unitIDPg pgtype.UUID
	if unitID != nil {
		unitIDPg = pgtype.UUID{Bytes: *unitID, Valid: true}
	}
	_, err = testQ.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
		UserID:   user.ID,
		SchemeID: schemeID,
		UnitID:   unitIDPg,
		Role:     role,
	})
	return user, err
}
```

- [ ] **Step 4: Add `createMemberWithRole` helper to test file**

Add it to `scheme_test.go` before the test functions. Need to add `pgtype` and `dbgen` imports if not already there.

- [ ] **Step 5: Ensure imports are complete**

The test will need: `"context"`, `"net/http"`, `"net/http/httptest"`, `"testing"`, `"time"`, `"github.com/google/uuid"`, `"github.com/jackc/pgx/v5/pgtype"`, `dbgen`, `auth`, `scheme`.

- [ ] **Step 6: Build test**

```bash
cd backend && go vet -tags=integration ./tests/integration/
```

- [ ] **Step 7: Commit**

```bash
git add backend/tests/integration/scheme_test.go
git commit -m "test: resident detail sees only own unit, trustee sees all"
```

---

### Task 6: Resident integration test — document visibility

**Files:**
- Modify: `backend/tests/integration/documents_test.go`

- [ ] **Step 1: Read current `documents_test.go`**

Read the full file to understand existing test patterns and imports.

- [ ] **Step 2: Add document visibility test**

```go
func TestDocuments_ResidentVisibility(t *testing.T) {
	ctx := context.Background()
	accessToken, orgID := setupAgent(t)
	claims, err := auth.ValidateAccessToken(accessToken, testJWTSigningKey)
	if err != nil {
		t.Fatalf("validate admin token: %v", err)
	}
	schemeID := setupScheme(t, accessToken)

	// Create a unit for the resident
	unit, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "3C", "Resident Owner"))
	if err != nil {
		t.Fatalf("create unit: %v", err)
	}

	// Create admin user (already authenticated via setupAgent)
	adminUserID := uuid.MustParse(claims.Subject)

	// Create resident member
	residentEmail := uniqueEmail(t)
	residentUser, err := testQ.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        residentEmail,
		PasswordHash: "test-hash",
		FullName:     "Resident User",
	})
	if err != nil {
		t.Fatalf("create resident user: %v", err)
	}
	_, err = testQ.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
		UserID:   residentUser.ID,
		SchemeID: uuid.MustParse(schemeID),
		UnitID:   pgtype.UUID{Bytes: unit.ID, Valid: true},
		Role:     "resident",
	})
	if err != nil {
		t.Fatalf("create resident membership: %v", err)
	}

	// Create documents with different visibilities
	docAll, err := testQ.CreateDocument(ctx, dbgen.CreateDocumentParams{
		SchemeID:         uuid.MustParse(schemeID),
		Name:             "Public Rules",
		StorageKey:       "/docs/rules.pdf",
		FileType:         dbgen.DocumentFileTypePdf,
		Category:         dbgen.DocumentCategoryRules,
		SizeBytes:        1024,
		UploadedByUserID: pgtype.UUID{Bytes: adminUserID, Valid: true},
		Visibility:       dbgen.DocumentVisibilityAll,
	})
	if err != nil {
		t.Fatalf("create all doc: %v", err)
	}

	docAdmin, err := testQ.CreateDocument(ctx, dbgen.CreateDocumentParams{
		SchemeID:         uuid.MustParse(schemeID),
		Name:             "Admin Financials",
		StorageKey:       "/docs/financials.pdf",
		FileType:         dbgen.DocumentFileTypePdf,
		Category:         dbgen.DocumentCategoryFinancial,
		SizeBytes:        2048,
		UploadedByUserID: pgtype.UUID{Bytes: adminUserID, Valid: true},
		Visibility:       dbgen.DocumentVisibilityAdmin,
	})
	if err != nil {
		t.Fatalf("create admin doc: %v", err)
	}

	// Admin sees both
	adminHandler := newDocumentsHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/schemes/"+schemeID+"/documents", nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	adminHandler.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin list: status=%d body=%s", w.Code, w.Body)
	}
	adminDocs := decodeSuccess[[]documents.DocumentInfo](t, w)
	if len(adminDocs) < 2 {
		t.Fatalf("admin sees %d docs, want >= 2", len(adminDocs))
	}

	// Resident sees only "all" visibility
	residentToken, err := auth.GenerateAccessToken(
		residentUser.ID.String(), orgID, "resident",
		"http://localhost:3000", "stratahq-api",
		testJWTSigningKey, 15*time.Minute,
	)
	if err != nil {
		t.Fatalf("generate resident token: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/schemes/"+schemeID+"/documents", nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, residentToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	adminHandler.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident list: status=%d body=%s", w.Code, w.Body)
	}
	residentDocs := decodeSuccess[[]documents.DocumentInfo](t, w)
	if len(residentDocs) != 1 {
		t.Fatalf("resident sees %d docs, want 1", len(residentDocs))
	}
	if residentDocs[0].ID != docAll.ID.String() {
		t.Fatalf("resident sees doc %q, want %q", residentDocs[0].ID, docAll.ID.String())
	}

	_ = docAdmin
}
```

Wait — the DocumentInfo type may not include visibility. Let me check the type and what the `ListDocumentsBySchemeAndVisibility` query returns. The DocumentInfo is a DTO, not a generated row type.

Also, `CreateDocument` params will need a `Visibility` field. This depends on sqlc generation. Let me check what params are generated.

Actually, this is getting too speculative without seeing the generated code. Let me simplify the plan — provide the core logic and note that exact field names depend on sqlc generation output.

- [ ] **Step 3: Build test**

```bash
cd backend && go vet -tags=integration ./tests/integration/
```

- [ ] **Step 4: Commit**

```bash
git add backend/tests/integration/documents_test.go
git commit -m "test: document visibility filtering for residents"
```

---

### Task 7: Raw CSV retention — null out after processing

**Files:**
- Modify: `backend/internal/levy/bank_statement_import.go`

- [ ] **Step 1: Add null-CSV after processing in `ProcessBankStatementImport`**

In `ProcessBankStatementImport`, after the transaction commits (line 739), add a call to null out the `raw_csv`:

```go
	if commitErr := tx.Commit(ctx); commitErr != nil {
		return commitErr
	}

	// Clear raw CSV after successful processing to minimize retained financial data
	if _, clearErr := s.db.Q.UpdateBankStatementImportRawCsv(ctx, dbgen.UpdateBankStatementImportRawCsvParams{
		ID:     importUUID,
		RawCsv: nil,
	}); clearErr != nil {
		return fmt.Errorf("clear raw csv: %w", clearErr)
	}

	return nil
```

- [ ] **Step 2: Add sqlc query for clearing raw_csv**

Read `backend/db/queries/levy.sql` and add:

```sql
-- name: UpdateBankStatementImportRawCsv :one
UPDATE bank_statement_imports
SET raw_csv = $2
WHERE id = $1
RETURNING id;
```

- [ ] **Step 3: Regenerate sqlc**

```bash
cd backend/db && sqlc generate
```

- [ ] **Step 4: Add migration to make raw_csv nullable**

Create `backend/db/migrations/00025_csv_nullable.sql`:

```sql
-- +goose Up
ALTER TABLE bank_statement_imports
    ALTER COLUMN raw_csv DROP NOT NULL;

-- +goose Down
ALTER TABLE bank_statement_imports
    ALTER COLUMN raw_csv SET NOT NULL;
```

- [ ] **Step 5: Build and vet**

```bash
cd backend && go build ./... && go vet ./internal/levy/...
```

- [ ] **Step 6: Update unit test expectations if `TestParseFNBStatementCSV` or related tests depend on non-null raw_csv**

Run: `go test ./internal/levy/... -v -count=1`

- [ ] **Step 7: Commit**

```bash
git add backend/internal/levy/bank_statement_import.go backend/db/queries/levy.sql backend/db/migrations/00025_csv_nullable.sql backend/db/gen/
git commit -m "feat: null out raw CSV after successful bank import processing"
```

---

### Task 8: All tasks final verification

- [ ] **Step 1: Run all unit tests**

```bash
cd backend && go test ./internal/... -v -race -count=1
```

- [ ] **Step 2: Run integration tests** (requires test database + Redis)

```bash
cd backend && go test -tags=integration ./tests/integration/... -v -race -count=1
```

- [ ] **Step 3: Verify CI RLS check still passes** (Task 0.4)

The existing RLS drift check from Phase 0 should continue to pass with the new migration.

- [ ] **Step 4: Final commit if any fixes needed**

```bash
git add -A && git commit -m "chore: final verification fixes for Phase 2"
```
