-- name: CreateScheme :one
INSERT INTO schemes (org_id, name, address, unit_count)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetScheme :one
SELECT * FROM schemes
WHERE id = $1
LIMIT 1;

-- name: ListSchemesByOrg :many
SELECT * FROM schemes
WHERE org_id = $1
ORDER BY name;

-- name: ListSchemeSummariesByOrg :many
WITH member_counts AS (
  SELECT
    sm.scheme_id,
    COUNT(*)::int AS total_members,
    COUNT(*) FILTER (WHERE sm.role = 'trustee')::int AS trustee_count,
    COUNT(*) FILTER (WHERE sm.role = 'resident')::int AS resident_count
  FROM scheme_memberships sm
  GROUP BY sm.scheme_id
),
maintenance_counts AS (
  SELECT
    mr.scheme_id,
    COUNT(*) FILTER (WHERE mr.status != 'resolved')::bigint AS open_maintenance_count
  FROM maintenance_requests mr
  GROUP BY mr.scheme_id
),
notice_counts AS (
  SELECT
    n.scheme_id,
    COUNT(*)::int AS notice_count
  FROM notices n
  GROUP BY n.scheme_id
),
next_agm AS (
  SELECT
    am.scheme_id,
    (MIN(am.meeting_date) FILTER (
      WHERE am.meeting_date IS NOT NULL
        AND am.meeting_date >= CURRENT_DATE
    ))::date AS next_agm_date
  FROM agm_meetings am
  GROUP BY am.scheme_id
),
latest_period AS (
  SELECT DISTINCT ON (lp.scheme_id)
    lp.scheme_id,
    lp.id AS period_id
  FROM levy_periods lp
  ORDER BY lp.scheme_id, lp.due_date DESC, lp.created_at DESC
),
collection AS (
  SELECT
    p.scheme_id,
    COALESCE(SUM(la.amount_cents), 0)::bigint AS total_due_cents,
    COALESCE(SUM(LEAST(la.paid_cents, la.amount_cents)), 0)::bigint AS total_paid_cents
  FROM latest_period p
  LEFT JOIN levy_accounts la ON la.period_id = p.period_id
  GROUP BY p.scheme_id
)
SELECT
  s.id,
  s.name,
  s.address,
  s.unit_count,
  COALESCE(mc.total_members, 0) AS total_members,
  COALESCE(mc.trustee_count, 0) AS trustee_count,
  COALESCE(mc.resident_count, 0) AS resident_count,
  COALESCE(mtc.open_maintenance_count, 0)::bigint AS open_maintenance_count,
  COALESCE(nc.notice_count, 0) AS notice_count,
  nagm.next_agm_date,
  CASE
    WHEN COALESCE(c.total_due_cents, 0) = 0 THEN 100
    ELSE ROUND((c.total_paid_cents::numeric * 100.0) / c.total_due_cents::numeric)::int
  END AS levy_collection_pct
FROM schemes s
LEFT JOIN member_counts mc ON mc.scheme_id = s.id
LEFT JOIN maintenance_counts mtc ON mtc.scheme_id = s.id
LEFT JOIN notice_counts nc ON nc.scheme_id = s.id
LEFT JOIN next_agm nagm ON nagm.scheme_id = s.id
LEFT JOIN collection c ON c.scheme_id = s.id
WHERE s.org_id = $1
ORDER BY s.name;

-- name: UpdateScheme :one
UPDATE schemes
SET name = $2, address = $3, unit_count = $4
WHERE id = $1
RETURNING *;

-- name: DeleteScheme :exec
DELETE FROM schemes
WHERE id = $1;

-- name: CreateUnit :one
INSERT INTO units (scheme_id, identifier, owner_name, floor, section_value_bps)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUnit :one
SELECT * FROM units
WHERE id = $1
LIMIT 1;

-- name: ListUnitsByScheme :many
SELECT * FROM units
WHERE scheme_id = $1
ORDER BY identifier;

-- name: UpdateUnit :one
UPDATE units
SET identifier = $2,
    owner_name = $3,
    floor = $4,
    section_value_bps = $5
WHERE id = $1
RETURNING *;

-- name: UpsertSchemeMembership :one
INSERT INTO scheme_memberships (user_id, scheme_id, unit_id, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, scheme_id)
DO UPDATE SET unit_id = EXCLUDED.unit_id, role = EXCLUDED.role
RETURNING *;

-- name: GetSchemeMembership :one
SELECT * FROM scheme_memberships
WHERE user_id = $1 AND scheme_id = $2
LIMIT 1;

-- name: ListSchemeMembersByScheme :many
SELECT sm.*,
       u.full_name,
       u.email,
       u.phone,
       un.identifier AS unit_identifier
FROM scheme_memberships sm
JOIN users u ON u.id = sm.user_id
LEFT JOIN units un ON un.id = sm.unit_id
WHERE sm.scheme_id = $1
ORDER BY u.full_name;

-- name: DeleteSchemeMembership :exec
DELETE FROM scheme_memberships
WHERE user_id = $1 AND scheme_id = $2;

-- name: SumSectionValuesByScheme :one
SELECT COALESCE(SUM(section_value_bps), 0)::INTEGER AS total_bps
FROM units
WHERE scheme_id = $1;
