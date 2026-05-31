-- backend/db/migrations/00033_invitation_pending_unique.sql
-- +goose Up

CREATE UNIQUE INDEX invitations_org_scheme_unit_email_pending_idx
ON invitations (
    org_id,
    scheme_id,
    COALESCE(unit_id, '00000000-0000-0000-0000-000000000000'::uuid),
    lower(email)
)
WHERE status = 'pending';

-- +goose Down

DROP INDEX IF EXISTS invitations_org_scheme_unit_email_pending_idx;
