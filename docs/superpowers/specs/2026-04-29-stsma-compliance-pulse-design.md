# STSMA Compliance Pulse — Design

## Goal
Elevate the existing per-scheme compliance dashboard into a proactive "compliance pulse" that surfaces deadlines, trends, and portfolio-level gaps. Add mutation endpoints so trustees/admins can manage compliance items through the app.

## Current State
- `GET /api/v1/compliance/{schemeId}` — read-only dashboard with items, score, and status counts
- 4 categories: financial, governance, administrative, insurance
- Items exist only in DB — no create/update/delete API
- No deadlines tracking or trend history
- Per-scheme only, no portfolio aggregation

## Desired State
- **Portfolio compliance overview** — admin sees compliance scores across all schemes
- **Upcoming deadlines** — items with due dates within 30 days flagged
- **Compliance trend** — last 4 assessments tracked per scheme
- **Item management** — create, update, delete compliance items via API
- **Assessment trigger** — "run assessment" endpoint that updates `assessed_at`
- **Category gap analysis** — which categories have the most non-compliant items

## Design

### 1. Portfolio Compliance Endpoint

**New endpoint:** `GET /api/v1/compliance/portfolio`

Returns aggregated compliance data for all schemes in the admin's organization:
```json
{
  "schemes": [
    {
      "scheme_id": "...",
      "scheme_name": "...",
      "score": 78,
      "total": 12,
      "compliant_count": 8,
      "at_risk_count": 3,
      "non_compliant_count": 1,
      "upcoming_deadlines": 2,
      "last_assessed_at": "2026-04-20T..."
    }
  ],
  "overall_score": 82,
  "total_schemes": 5,
  "healthy_schemes": 3,
  "at_risk_schemes": 2,
  "critical_schemes": 0
}
```

### 2. Compliance Trend Tracking

**New table:** `compliance_assessments`

```sql
CREATE TABLE compliance_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id UUID NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    score INT NOT NULL,
    total_items INT NOT NULL,
    compliant_count INT NOT NULL,
    at_risk_count INT NOT NULL,
    non_compliant_count INT NOT NULL,
    assessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Scores are snapshotted when an assessment is triggered. Dashboard returns the last N assessments.

### 3. Item Management Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/compliance/{schemeId}/items` | Create compliance item |
| `PUT` | `/compliance/{schemeId}/items/{id}` | Update item (status, detail, action) |
| `DELETE` | `/compliance/{schemeId}/items/{id}` | Delete item |
| `POST` | `/compliance/{schemeId}/assess` | Trigger assessment — snapshot current score |

### 4. Upcoming Deadlines

Add a `due_soon_count` field to both per-scheme and portfolio responses. Items with `due_date` within 30 days of now AND `status != 'compliant'` are counted.

### 5. Default Compliance Catalog

Add a seed function that creates the standard STSMA compliance items when a new scheme is created. Items based on STSMA requirements:

| Category | Item | Default Status |
|---|---|---|
| Financial | Reserve fund minimum contribution | open |
| Financial | Annual audit statements submitted | open |
| Financial | Levy budget approved for current year | open |
| Governance | Trustees elected and registered | open |
| Governance | AGM held within 4 months of FYE | open |
| Governance | Meeting minutes maintained | open |
| Administrative | Scheme rules registered with CSOS | open |
| Administrative | CSOS annual return filed | open |
| Administrative | Records kept at registered address | open |
| Insurance | Building insurance in force | open |
| Insurance | Replacement valuation current (<3yrs) | open |

Status defaults to "non-compliant" — schemes must prove compliance by updating each item.

### Non-goals
- Automatic status assessment (requires AI/document review)
- Public compliance report generation
- Integration with CSOS API
- Reminder notifications for upcoming deadlines (Phase 2)
