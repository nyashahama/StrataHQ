-- +goose Up

CREATE TYPE agm_status AS ENUM ('upcoming', 'in_progress', 'closed');

CREATE TYPE resolution_status AS ENUM ('open', 'passed', 'failed');

CREATE TYPE vote_choice AS ENUM ('for', 'against', 'abstain');

CREATE TABLE agm_meetings (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id        UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    meeting_date     DATE        NOT NULL,
    quorum_required  INTEGER     NOT NULL CHECK (quorum_required > 0),
    quorum_present   INTEGER     NOT NULL DEFAULT 0 CHECK (quorum_present >= 0),
    status           agm_status  NOT NULL DEFAULT 'upcoming',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER agm_meetings_set_updated_at
    BEFORE UPDATE ON agm_meetings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_agm_meetings_scheme_id ON agm_meetings(scheme_id);

CREATE TABLE agm_resolutions (
    id              UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id      UUID              NOT NULL REFERENCES agm_meetings(id) ON DELETE CASCADE,
    title           TEXT              NOT NULL,
    description     TEXT              NOT NULL,
    votes_for       INTEGER           NOT NULL DEFAULT 0 CHECK (votes_for >= 0),
    votes_against   INTEGER           NOT NULL DEFAULT 0 CHECK (votes_against >= 0),
    total_eligible  INTEGER           NOT NULL CHECK (total_eligible > 0),
    status          resolution_status NOT NULL DEFAULT 'open',
    created_at      TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

CREATE TRIGGER agm_resolutions_set_updated_at
    BEFORE UPDATE ON agm_resolutions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_agm_resolutions_meeting_id ON agm_resolutions(meeting_id);

-- One proxy per grantor per meeting. Grantor delegates their vote to grantee.
CREATE TABLE proxy_assignments (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id       UUID        NOT NULL REFERENCES agm_meetings(id) ON DELETE CASCADE,
    grantor_user_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    grantee_user_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (meeting_id, grantor_user_id)
);

-- One vote per voter per resolution.
CREATE TABLE agm_votes (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    resolution_id   UUID        NOT NULL REFERENCES agm_resolutions(id) ON DELETE CASCADE,
    voter_user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vote            vote_choice NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (resolution_id, voter_user_id)
);

-- +goose Down

DROP TABLE   IF EXISTS agm_votes;
DROP TABLE   IF EXISTS proxy_assignments;
DROP TRIGGER IF EXISTS agm_resolutions_set_updated_at ON agm_resolutions;
DROP INDEX   IF EXISTS idx_agm_resolutions_meeting_id;
DROP TABLE   IF EXISTS agm_resolutions;
DROP TRIGGER IF EXISTS agm_meetings_set_updated_at ON agm_meetings;
DROP INDEX   IF EXISTS idx_agm_meetings_scheme_id;
DROP TABLE   IF EXISTS agm_meetings;
DROP TYPE    IF EXISTS vote_choice;
DROP TYPE    IF EXISTS resolution_status;
DROP TYPE    IF EXISTS agm_status;
