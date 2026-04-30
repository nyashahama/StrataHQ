-- name: CreateContractor :one
INSERT INTO contractors (
    org_id, name, trade, phone, email, suburb, city, province,
    public_profile, vetted, active, notes, created_by_user_id
)
VALUES (
    sqlc.arg(org_id), sqlc.arg(name), sqlc.arg(trade),
    sqlc.arg(phone), sqlc.arg(email), sqlc.arg(suburb),
    sqlc.arg(city), sqlc.arg(province), sqlc.arg(public_profile),
    sqlc.arg(vetted), sqlc.arg(active), sqlc.arg(notes),
    sqlc.arg(created_by_user_id)
)
RETURNING *;

-- name: UpdateContractor :one
UPDATE contractors
SET name = sqlc.arg(name),
    trade = sqlc.arg(trade),
    phone = sqlc.arg(phone),
    email = sqlc.arg(email),
    suburb = sqlc.arg(suburb),
    city = sqlc.arg(city),
    province = sqlc.arg(province),
    public_profile = sqlc.arg(public_profile),
    vetted = sqlc.arg(vetted),
    active = sqlc.arg(active),
    notes = sqlc.arg(notes)
WHERE id = sqlc.arg(id)
  AND org_id = sqlc.arg(org_id)
RETURNING *;

-- name: GetContractor :one
SELECT *
FROM contractors
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: ListContractorSchemeIDs :many
SELECT scheme_id
FROM scheme_contractors
WHERE contractor_id = sqlc.arg(contractor_id)
ORDER BY scheme_id;

-- name: UpsertSchemeContractor :exec
INSERT INTO scheme_contractors (scheme_id, contractor_id, preferred)
VALUES (sqlc.arg(scheme_id), sqlc.arg(contractor_id), sqlc.arg(preferred))
ON CONFLICT (scheme_id, contractor_id)
DO UPDATE SET preferred = EXCLUDED.preferred;

-- name: DeleteSchemeContractor :execrows
DELETE FROM scheme_contractors
WHERE scheme_id = sqlc.arg(scheme_id)
  AND contractor_id = sqlc.arg(contractor_id);

-- name: DeleteContractorSchemeLinks :exec
DELETE FROM scheme_contractors
WHERE contractor_id = sqlc.arg(contractor_id);

-- name: CountSchemesByOrgForContractorLinks :one
SELECT COUNT(*)::int
FROM schemes
WHERE org_id = sqlc.arg(org_id)
  AND id = ANY(sqlc.arg(scheme_ids)::uuid[]);

-- name: ListContractorsByOrg :many
SELECT
    c.*,
    COALESCE(AVG(cr.rating), 0)::float8 AS average_rating,
    COUNT(cr.id)::int AS review_count,
    COUNT(mr.id) FILTER (WHERE mr.status = 'resolved')::int AS completed_job_count,
    COALESCE(bool_or(sc.preferred), false)::boolean AS preferred
FROM contractors c
LEFT JOIN contractor_reviews cr ON cr.contractor_id = c.id
LEFT JOIN maintenance_requests mr ON mr.contractor_id = c.id
LEFT JOIN scheme_contractors sc ON sc.contractor_id = c.id
WHERE c.org_id = sqlc.arg(org_id)
  AND (sqlc.narg(scheme_id)::uuid IS NULL OR sc.scheme_id = sqlc.narg(scheme_id)::uuid)
  AND (sqlc.narg(trade)::maintenance_category IS NULL OR c.trade = sqlc.narg(trade)::maintenance_category)
  AND (sqlc.narg(suburb)::text IS NULL OR lower(c.suburb) = lower(sqlc.narg(suburb)::text))
  AND (sqlc.narg(query)::text IS NULL OR lower(c.name) LIKE '%' || lower(sqlc.narg(query)::text) || '%')
  AND (sqlc.narg(vetted)::boolean IS NULL OR c.vetted = sqlc.narg(vetted)::boolean)
  AND (sqlc.narg(active)::boolean IS NULL OR c.active = sqlc.narg(active)::boolean)
GROUP BY c.id
ORDER BY c.active DESC, preferred DESC, average_rating DESC, c.name ASC;

-- name: SearchContractorMarketplace :many
SELECT
    c.*,
    COALESCE(sc.preferred, false)::boolean AS preferred,
    COALESCE(AVG(cr.rating), 0)::float8 AS average_rating,
    COUNT(cr.id)::int AS review_count,
    COUNT(mr.id) FILTER (WHERE mr.status = 'resolved')::int AS completed_job_count
FROM contractors c
LEFT JOIN scheme_contractors sc
  ON sc.contractor_id = c.id
 AND sc.scheme_id = sqlc.arg(scheme_id)
LEFT JOIN contractor_reviews cr ON cr.contractor_id = c.id
LEFT JOIN maintenance_requests mr ON mr.contractor_id = c.id
WHERE c.active = true
  AND (
    sc.scheme_id IS NOT NULL
    OR (c.public_profile = true AND c.vetted = true)
  )
  AND (sqlc.narg(trade)::maintenance_category IS NULL OR c.trade = sqlc.narg(trade)::maintenance_category)
  AND (sqlc.narg(suburb)::text IS NULL OR lower(c.suburb) = lower(sqlc.narg(suburb)::text))
GROUP BY c.id, sc.preferred
ORDER BY preferred DESC, average_rating DESC, completed_job_count DESC, c.name ASC;

-- name: ContractorAssignableToScheme :one
SELECT c.*
FROM contractors c
LEFT JOIN scheme_contractors sc
  ON sc.contractor_id = c.id
 AND sc.scheme_id = sqlc.arg(scheme_id)
JOIN schemes s ON s.id = sqlc.arg(scheme_id)
WHERE c.id = sqlc.arg(contractor_id)
  AND c.active = true
  AND (
    sc.scheme_id IS NOT NULL
    OR (c.public_profile = true AND c.vetted = true)
  )
  AND (
    c.org_id = s.org_id
    OR (c.public_profile = true AND c.vetted = true)
  )
LIMIT 1;

-- name: CreateContractorReview :one
INSERT INTO contractor_reviews (
    contractor_id, scheme_id, maintenance_request_id, rating, comment, created_by_user_id
)
VALUES (
    sqlc.arg(contractor_id), sqlc.arg(scheme_id), sqlc.arg(maintenance_request_id),
    sqlc.arg(rating), sqlc.arg(comment), sqlc.arg(created_by_user_id)
)
RETURNING *;

-- name: ListContractorReviews :many
SELECT *
FROM contractor_reviews
WHERE contractor_id = sqlc.arg(contractor_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_rows);
