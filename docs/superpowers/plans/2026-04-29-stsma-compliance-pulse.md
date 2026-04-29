# STSMA Compliance Pulse — Implementation Plan

> **For agentic workers:** Use subagent-driven-development or executing-plans to implement.

**Goal:** Add portfolio compliance view, item management endpoints, deadline tracking, trend snapshots, and seed catalog.

**Architecture:** New migration for trend table, extended compliance service with mutations, portfolio aggregation via org-level query, and frontend updates.

---

## Task 1: Add `compliance_assessments` table and item management queries

**Files:**
- Create: `backend/db/migrations/00026_compliance_assessments.sql`
- Modify: `backend/db/queries/compliance.sql`
- Regenerate: `backend/db/gen/`

- [ ] **Step 1: Create migration**

```sql
-- +goose Up
CREATE TABLE compliance_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id UUID NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    score INT NOT NULL CHECK (score >= 0 AND score <= 100),
    total_items INT NOT NULL,
    compliant_count INT NOT NULL,
    at_risk_count INT NOT NULL,
    non_compliant_count INT NOT NULL,
    assessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_compliance_assessments_scheme_assessed
    ON compliance_assessments (scheme_id, assessed_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_compliance_assessments_scheme_assessed;
DROP TABLE IF EXISTS compliance_assessments;
```

- [ ] **Step 2: Add queries**

In `backend/db/queries/compliance.sql`, add:

```sql
-- name: CreateComplianceAssessment :one
INSERT INTO compliance_assessments (scheme_id, score, total_items, compliant_count, at_risk_count, non_compliant_count)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListComplianceAssessmentsByScheme :many
SELECT * FROM compliance_assessments
WHERE scheme_id = $1
ORDER BY assessed_at DESC
LIMIT $2;

-- name: UpdateComplianceItem :one
UPDATE compliance_items
SET status = $2, detail = $3, action = $4, due_date = $5, assessed_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteComplianceItem :exec
DELETE FROM compliance_items WHERE id = $1;

-- name: CountUpcomingDeadlinesByScheme :one
SELECT COUNT(*) FROM compliance_items
WHERE scheme_id = $1
  AND due_date IS NOT NULL
  AND due_date >= CURRENT_DATE
  AND due_date <= CURRENT_DATE + INTERVAL '30 days'
  AND status != 'compliant';
```

- [ ] **Step 3: Regenerate sqlc**

```bash
cd backend/db && sqlc generate
```

---

## Task 2: Add item management to compliance service

**Files:**
- Modify: `backend/internal/compliance/service.go`

- [ ] **Step 1: Add input types**

```go
type CreateItemInput struct {
    Category    string
    Title       string
    Requirement string
    Detail      string
    Action      string
    DueDate     *time.Time
}

type UpdateItemInput struct {
    Status   *string
    Detail   *string
    Action   *string
    DueDate  *time.Time
}
```

- [ ] **Step 2: Add service methods**

```go
func (s *Service) CreateItem(ctx context.Context, identity auth.Identity, schemeID string, input CreateItemInput) (*ItemInfo, error) {
    access, err := s.resolveAccess(ctx, identity, schemeID)
    if err != nil { return nil, err }
    if auth.IsResidentRole(access.role) { return nil, ErrForbidden }

    if !validCategory(input.Category) || input.Title == "" {
        return nil, ErrInvalidInput
    }

    var dueDate pgtype.Date
    if input.DueDate != nil {
        dueDate = pgtype.Date{Time: *input.DueDate, Valid: true}
    }

    item, err := s.db.Q.CreateComplianceItem(ctx, dbgen.CreateComplianceItemParams{
        SchemeID:    access.scheme.ID,
        Category:    dbgen.ComplianceCategory(input.Category),
        Title:       input.Title,
        Requirement: input.Requirement,
        Detail:      input.Detail,
        Action:      input.Action,
        DueDate:     dueDate,
    })
    if err != nil { return nil, err }
    return &ItemInfo{...map item...}, nil
}

func (s *Service) UpdateItem(ctx context.Context, identity auth.Identity, schemeID, itemID string, input UpdateItemInput) (*ItemInfo, error) {
    access, err := s.resolveAccess(ctx, identity, schemeID)
    if err != nil { return nil, err }
    if auth.IsResidentRole(access.role) { return nil, ErrForbidden }

    id, err := uuid.Parse(itemID)
    if err != nil { return nil, ErrInvalidInput }

    existing, err := s.db.Q.GetComplianceItem(ctx, id)
    if err != nil { ... return nil, ErrNotFound }
    if existing.SchemeID != access.scheme.ID { return nil, ErrForbidden }

    status := existing.Status
    if input.Status != nil { status = dbgen.ComplianceStatus(*input.Status) }
    detail := existing.Detail
    if input.Detail != nil { detail = *input.Detail }
    action := existing.Action
    if input.Action != nil { action = *input.Action }
    dueDate := existing.DueDate
    if input.DueDate != nil { dueDate = pgtype.Date{Time: *input.DueDate, Valid: true} }

    updated, err := s.db.Q.UpdateComplianceItem(ctx, dbgen.UpdateComplianceItemParams{
        ID:      id,
        Status:  status,
        Detail:  detail,
        Action:  action,
        DueDate: dueDate,
    })
    ...
}

func (s *Service) DeleteItem(ctx context.Context, identity auth.Identity, schemeID, itemID string) error {
    // similar resolve + delete
}

func (s *Service) Assess(ctx context.Context, identity auth.Identity, schemeID string) (*DashboardResponse, error) {
    dashboard, err := s.Dashboard(ctx, identity, schemeID)
    if err != nil { return nil, err }

    access, _ := s.resolveAccess(ctx, identity, schemeID)
    _, _ = s.db.Q.CreateComplianceAssessment(ctx, dbgen.CreateComplianceAssessmentParams{
        SchemeID:         access.scheme.ID,
        Score:            int32(dashboard.Score),
        TotalItems:       int32(dashboard.Total),
        CompliantCount:   int32(dashboard.CompliantCount),
        AtRiskCount:      int32(dashboard.AtRiskCount),
        NonCompliantCount: int32(dashboard.NonCompliantCount),
    })
    return dashboard, nil
}
```

- [ ] **Step 3: Add upcoming deadlines to DashboardResponse**

Extend `DashboardResponse`:
```go
type DashboardResponse struct {
    // ... existing ...
    UpcomingDeadlines int `json:"upcoming_deadlines"`
}
```

In `Dashboard`, call `CountUpcomingDeadlinesByScheme`.

- [ ] **Step 4: Add `GetComplianceItem` query** (needed for update)

```sql
-- name: GetComplianceItem :one
SELECT * FROM compliance_items WHERE id = $1 LIMIT 1;
```

---

## Task 3: Add portfolio compliance endpoint

**Files:**
- Modify: `backend/internal/compliance/service.go`
- Modify: `backend/internal/compliance/handler.go`
- Modify: `backend/internal/compliance/routes.go`

- [ ] **Step 1: Add `PortfolioDashboard` service method**

```go
type PortfolioSchemeInfo struct {
    SchemeID           string    `json:"scheme_id"`
    SchemeName         string    `json:"scheme_name"`
    Score              int       `json:"score"`
    Total              int       `json:"total"`
    CompliantCount     int       `json:"compliant_count"`
    AtRiskCount        int       `json:"at_risk_count"`
    NonCompliantCount  int       `json:"non_compliant_count"`
    UpcomingDeadlines  int       `json:"upcoming_deadlines"`
    LastAssessedAt     time.Time `json:"last_assessed_at"`
}

type PortfolioDashboardResponse struct {
    Schemes          []PortfolioSchemeInfo `json:"schemes"`
    OverallScore     int                   `json:"overall_score"`
    TotalSchemes     int                   `json:"total_schemes"`
    HealthySchemes   int                   `json:"healthy_schemes"`
    AtRiskSchemes    int                   `json:"at_risk_schemes"`
    CriticalSchemes  int                   `json:"critical_schemes"`
}
```

Logic: iterate over all schemes in the org, call `Dashboard` for each, aggregate scores.

- [ ] **Step 2: Add handler and route**

In `handler.go`:
```go
func (h *Handler) PortfolioDashboard(w http.ResponseWriter, r *http.Request) {
    identity, ok := auth.IdentityFromRequest(r)
    if !ok { ... }
    dashboard, err := h.service.PortfolioDashboard(r.Context(), identity)
    ...
}
```

In `routes.go`:
```go
r.Get("/portfolio", h.PortfolioDashboard)
```

---

## Task 4: Add compliance item routes

**Files:**
- Modify: `backend/internal/compliance/handler.go`
- Modify: `backend/internal/compliance/routes.go`

- [ ] **Step 1: Add handler methods**

```go
func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) { ... }
func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) { ... }
func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) { ... }
func (h *Handler) Assess(w http.ResponseWriter, r *http.Request) { ... }
```

- [ ] **Step 2: Register routes**

```go
r.Post("/{schemeId}/items", h.CreateItem)
r.Put("/{schemeId}/items/{itemId}", h.UpdateItem)
r.Delete("/{schemeId}/items/{itemId}", h.DeleteItem)
r.Post("/{schemeId}/assess", h.Assess)
```

---

## Task 5: Add default compliance catalog seed

**Files:**
- Create: `backend/internal/compliance/catalog.go`

- [ ] **Step 1: Create function to seed default items**

```go
var DefaultComplianceItems = []struct {
    Category    string
    Title       string
    Requirement string
}{
    {"financial", "Reserve fund minimum contribution", "STSMA s3(1)(b) — ..."},
    {"financial", "Annual audit statements submitted", "STSMA s4(1) — ..."},
    {"financial", "Levy budget approved for current year", "STSMA s4(2) — ..."},
    {"governance", "Trustees elected and registered", "STSMA s7(1) — ..."},
    {"governance", "AGM held within 4 months of FYE", "STSMA s7(2) — ..."},
    {"governance", "Meeting minutes maintained", "STSMA s7(3) — ..."},
    {"administrative", "Scheme rules registered with CSOS", "STSMA s11 — ..."},
    {"administrative", "CSOS annual return filed", "CSOS Act s5 — ..."},
    {"administrative", "Records kept at registered address", "STSMA s5 — ..."},
    {"insurance", "Building insurance in force", "STSMA s12(1) — ..."},
    {"insurance", "Replacement valuation current (<3yrs)", "STSMA s12(2) — ..."},
}

func (s *Service) SeedDefaultItems(ctx context.Context, schemeID uuid.UUID) (int, error) {
    count := 0
    for _, item := range DefaultComplianceItems {
        _, err := s.db.Q.CreateComplianceItem(ctx, dbgen.CreateComplianceItemParams{
            SchemeID:    schemeID,
            Category:    dbgen.ComplianceCategory(item.Category),
            Title:       item.Title,
            Requirement: item.Requirement,
            Status:      dbgen.ComplianceStatusNonCompliant,
            Detail:      "Not yet assessed",
            Action:      "",
        })
        if err != nil { return count, err }
        count++
    }
    return count, nil
}
```

- [ ] **Step 2: Call seed when scheme is created**

In `backend/internal/scheme/service.go`, after creating a scheme, call:
```go
_, _ = s.compliance.SeedDefaultItems(ctx, schemeID)
```
This requires injecting the compliance service into the scheme service (or calling after creation in the handler).

---

## Task 6: Update frontend types and API client

**Files:**
- Modify: `lib/compliance.ts`
- Modify: `lib/compliance-api.ts`

- [ ] **Step 1: Add new types**

Add `PortfolioCompliance`, `CreateComplianceItemInput`, `UpdateComplianceItemInput` types.

- [ ] **Step 2: Add API functions**

```ts
export async function getPortfolioCompliance(): Promise<PortfolioCompliance> { ... }
export async function createComplianceItem(schemeId: string, input: CreateComplianceItemInput): Promise<ComplianceItem> { ... }
export async function updateComplianceItem(schemeId: string, itemId: string, input: UpdateComplianceItemInput): Promise<ComplianceItem> { ... }
export async function deleteComplianceItem(schemeId: string, itemId: string): Promise<void> { ... }
export async function assessCompliance(schemeId: string): Promise<ComplianceDashboard> { ... }
```

---

## Task 7: Build and verify

- [ ] Build: `go build ./...`
- [ ] Tests: `go test ./... -count=1`
- [ ] Frontend: `npx tsc --noEmit && npm run build`
