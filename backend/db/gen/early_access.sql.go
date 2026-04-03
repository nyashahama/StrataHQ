// backend/db/gen/early_access.sql.go
package dbgen

import (
	"context"

	"github.com/google/uuid"
)

type CreateEarlyAccessRequestParams struct {
	FullName   string
	Email      string
	SchemeName string
	UnitCount  int32
}

const createEarlyAccessRequest = `
INSERT INTO early_access_requests (full_name, email, scheme_name, unit_count)
VALUES ($1, $2, $3, $4)
RETURNING id, full_name, email, scheme_name, unit_count, status, created_at, reviewed_at
`

func (q *Queries) CreateEarlyAccessRequest(ctx context.Context, p CreateEarlyAccessRequestParams) (EarlyAccessRequest, error) {
	row := q.db.QueryRow(ctx, createEarlyAccessRequest, p.FullName, p.Email, p.SchemeName, p.UnitCount)
	var r EarlyAccessRequest
	err := row.Scan(&r.ID, &r.FullName, &r.Email, &r.SchemeName, &r.UnitCount, &r.Status, &r.CreatedAt, &r.ReviewedAt)
	return r, err
}

const listEarlyAccessRequests = `
SELECT id, full_name, email, scheme_name, unit_count, status, created_at, reviewed_at
FROM early_access_requests
ORDER BY created_at DESC
`

func (q *Queries) ListEarlyAccessRequests(ctx context.Context) ([]EarlyAccessRequest, error) {
	rows, err := q.db.Query(ctx, listEarlyAccessRequests)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []EarlyAccessRequest
	for rows.Next() {
		var r EarlyAccessRequest
		if err := rows.Scan(&r.ID, &r.FullName, &r.Email, &r.SchemeName, &r.UnitCount, &r.Status, &r.CreatedAt, &r.ReviewedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

const getEarlyAccessRequest = `
SELECT id, full_name, email, scheme_name, unit_count, status, created_at, reviewed_at
FROM early_access_requests
WHERE id = $1
`

func (q *Queries) GetEarlyAccessRequest(ctx context.Context, id uuid.UUID) (EarlyAccessRequest, error) {
	row := q.db.QueryRow(ctx, getEarlyAccessRequest, id)
	var r EarlyAccessRequest
	err := row.Scan(&r.ID, &r.FullName, &r.Email, &r.SchemeName, &r.UnitCount, &r.Status, &r.CreatedAt, &r.ReviewedAt)
	return r, err
}

type UpdateEarlyAccessStatusParams struct {
	ID     uuid.UUID
	Status EarlyAccessStatus
}

const updateEarlyAccessStatus = `
UPDATE early_access_requests
SET status = $2, reviewed_at = NOW()
WHERE id = $1
RETURNING id, full_name, email, scheme_name, unit_count, status, created_at, reviewed_at
`

func (q *Queries) UpdateEarlyAccessStatus(ctx context.Context, p UpdateEarlyAccessStatusParams) (EarlyAccessRequest, error) {
	row := q.db.QueryRow(ctx, updateEarlyAccessStatus, p.ID, p.Status)
	var r EarlyAccessRequest
	err := row.Scan(&r.ID, &r.FullName, &r.Email, &r.SchemeName, &r.UnitCount, &r.Status, &r.CreatedAt, &r.ReviewedAt)
	return r, err
}
