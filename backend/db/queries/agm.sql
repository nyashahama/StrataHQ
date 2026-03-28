-- name: CreateAgmMeeting :one
INSERT INTO agm_meetings (scheme_id, meeting_date, quorum_required)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAgmMeeting :one
SELECT * FROM agm_meetings
WHERE id = $1
LIMIT 1;

-- name: ListAgmMeetingsByScheme :many
SELECT * FROM agm_meetings
WHERE scheme_id = $1
ORDER BY meeting_date DESC;

-- name: UpdateAgmMeetingStatus :one
UPDATE agm_meetings
SET status = $2
WHERE id = $1
RETURNING *;

-- name: UpdateAgmQuorumPresent :one
UPDATE agm_meetings
SET quorum_present = $2
WHERE id = $1
RETURNING *;

-- name: CreateAgmResolution :one
INSERT INTO agm_resolutions (meeting_id, title, description, total_eligible)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAgmResolution :one
SELECT * FROM agm_resolutions
WHERE id = $1
LIMIT 1;

-- name: ListAgmResolutionsByMeeting :many
SELECT * FROM agm_resolutions
WHERE meeting_id = $1
ORDER BY created_at;

-- name: UpdateAgmResolutionVotes :one
UPDATE agm_resolutions
SET votes_for     = $2,
    votes_against = $3
WHERE id = $1
RETURNING *;

-- name: UpdateAgmResolutionStatus :one
UPDATE agm_resolutions
SET status = $2
WHERE id = $1
RETURNING *;

-- name: CreateProxyAssignment :one
INSERT INTO proxy_assignments (meeting_id, grantor_user_id, grantee_user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetProxyAssignment :one
SELECT * FROM proxy_assignments
WHERE meeting_id = $1 AND grantor_user_id = $2
LIMIT 1;

-- name: ListProxyAssignmentsByMeeting :many
SELECT * FROM proxy_assignments
WHERE meeting_id = $1;

-- name: CreateAgmVote :one
INSERT INTO agm_votes (resolution_id, voter_user_id, vote)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAgmVote :one
SELECT * FROM agm_votes
WHERE resolution_id = $1 AND voter_user_id = $2
LIMIT 1;

-- name: ListAgmVotesByResolution :many
SELECT * FROM agm_votes
WHERE resolution_id = $1;
