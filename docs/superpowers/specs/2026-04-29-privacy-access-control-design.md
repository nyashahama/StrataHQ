# Phase 2 — Privacy and Access Control Design

## Goal
Protect resident privacy by filtering sensitive owner/unit data in API responses and restricting document visibility by role. Ensure residents see only their own information, not the full scheme register or all documents.

## Background
The security audit identified that residents (non-admin scheme members) receive too much data through several endpoints:

| Endpoint | Leak | Severity |
|---|---|---|
| `GET /schemes/{id}` | `Units[]` includes all units with owner names, floors, section values | High (#95) |
| `GET /schemes/{id}/units` | Same as above — all units unfiltered | High |
| `GET /schemes/{id}/documents` | All documents listed with full metadata (and base64 content in `storage_key`) | Critical (#108) |
| `GET /schemes/{id}` | `RecentNotices[]` unfiltered by visibility | Medium |

### What already works
- `ListMembers` correctly filters: residents see only trustees
- `Dashboard` (levy) correctly strips levy roll/trend for residents
- Admin/trustee access is unchanged

## Design

### 1. Filter scheme units for non-admin roles

**Target files:** `backend/internal/scheme/service.go` — `Get` (line 212) and `ListUnits` (line 365)

**Approach:** When the caller is not admin (`!auth.IsAdminRole(role)`), return only the member's own unit instead of all units. Trustees see all units (they need the full register for scheme management).

```go
// In Get, after line 226 (ListUnitsByScheme):
if !auth.IsAdminRole(role) && unitID != nil {
    // Non-admin: filter to only the member's own unit
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

Trustees are non-admin role but should see all units. Use `auth.IsResidentRole(role)` for the filter instead.

### 2. Document visibility by role

**Target files:** `backend/internal/documents/service.go` — `List` (line 77)

**Approach:** Add a `visibility` column to `scheme_documents` with values `all`, `trustee`, `admin`. Default is `admin` for new documents (conservative). Filter listing by the caller's role:
- Admin: sees everything
- Trustee: sees `all` + `trustee` visibility
- Resident: sees only `all` visibility

Migration adds the column with existing documents defaulting to `all` (backward-compatible).

**Files changed:**
- `backend/db/migrations/00023_document_visibility.sql` — new migration
- `backend/db/queries/documents.sql` — update `ListDocumentsByScheme` to accept visibility filter
- `backend/internal/documents/service.go` — pass visibility filter based on role

### 3. Notice filtering for residents

**Target files:** `backend/internal/scheme/service.go` — `Get` (line 228)

**Approach:** Add a `visible_to_residents` column (default `false`) to notices. Filter `RecentNotices` for residents. Admins/trustees see all notices.

### 4. Raw CSV retention

**Target files:** 
- `backend/db/migrations/00025_csv_retention.sql` — new migration (added to 00023 instead to keep single migration)
- `backend/internal/levy/bank_statement_import.go` — null out `raw_csv` after successful processing

**Approach:** After a bank statement import is fully processed (status = `applied`), set `raw_csv` to NULL. The import had its data extracted into rows and the raw file is no longer needed. Add a background cleanup job for imports older than 90 days.

### 5. Test coverage

**Target files:** `backend/tests/integration/scheme_test.go`, `backend/tests/integration/documents_test.go`

Add integration tests:
1. Resident gets scheme detail — sees only own unit, no other units
2. Resident lists units — sees only own unit
3. Trustee gets scheme detail — sees all units
4. Resident lists documents — sees only `all` visibility docs
5. Trustee lists documents — sees `all` + `trustee` visibility docs
6. Admin lists documents — sees all

### Non-goals (deferred)
- Moving document payloads to S3/object storage (Phase 5 infrastructure)
- WhatsApp goroutine → job queue migration (Phase 5)
- Document content hashing/versioning
- Notice visibility column (minor; can be addressed in a follow-up)
