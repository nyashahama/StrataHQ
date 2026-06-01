-- name: EnqueueBackgroundJob :one
INSERT INTO background_jobs (
    kind,
    payload,
    idempotency_key,
    max_attempts,
    run_after
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (kind, idempotency_key) WHERE status IN ('queued', 'running') DO UPDATE
SET updated_at = background_jobs.updated_at
RETURNING *;

-- name: ClaimDueBackgroundJobs :many
WITH due_jobs AS (
    SELECT id
    FROM background_jobs
    WHERE status = 'queued'
      AND run_after <= now()
    ORDER BY run_after ASC, created_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE background_jobs bj
SET status = 'running',
    locked_at = now(),
    locked_by = $2,
    updated_at = now()
FROM due_jobs
WHERE bj.id = due_jobs.id
RETURNING bj.*;

-- name: MarkBackgroundJobSucceeded :one
UPDATE background_jobs
SET status = 'succeeded',
    locked_at = NULL,
    locked_by = NULL,
    last_error = NULL,
    succeeded_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RetryBackgroundJob :one
UPDATE background_jobs
SET status = 'queued',
    attempts = attempts + 1,
    run_after = $2,
    locked_at = NULL,
    locked_by = NULL,
    last_error = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: MarkBackgroundJobFailed :one
UPDATE background_jobs
SET status = 'failed',
    attempts = attempts + 1,
    locked_at = NULL,
    locked_by = NULL,
    last_error = $2,
    failed_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RecoverStaleBackgroundJobs :many
UPDATE background_jobs
SET status = CASE
        WHEN attempts + 1 >= max_attempts THEN 'failed'
        ELSE 'queued'
    END,
    attempts = attempts + 1,
    locked_at = NULL,
    locked_by = NULL,
    last_error = $2,
    failed_at = CASE
        WHEN attempts + 1 >= max_attempts THEN now()
        ELSE NULL
    END,
    updated_at = now()
WHERE status = 'running'
  AND locked_at < $1
RETURNING *;

-- name: GetBackgroundJob :one
SELECT *
FROM background_jobs
WHERE id = $1;
