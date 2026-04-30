-- +goose Up

CREATE TABLE contractors (
    id                 UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             UUID                 NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name               TEXT                 NOT NULL,
    trade              maintenance_category NOT NULL,
    phone              TEXT,
    email              TEXT,
    suburb             TEXT                 NOT NULL,
    city               TEXT                 NOT NULL DEFAULT 'Cape Town',
    province           TEXT                 NOT NULL DEFAULT 'Western Cape',
    public_profile     BOOLEAN              NOT NULL DEFAULT false,
    vetted             BOOLEAN              NOT NULL DEFAULT false,
    active             BOOLEAN              NOT NULL DEFAULT true,
    notes              TEXT,
    created_by_user_id UUID                 REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ          NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ          NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_contractors_unique_org_name_trade_suburb
    ON contractors (org_id, lower(name), trade, lower(suburb));

CREATE TRIGGER contractors_set_updated_at
    BEFORE UPDATE ON contractors
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_contractors_org_trade_suburb
    ON contractors (org_id, trade, suburb);

CREATE INDEX idx_contractors_public_marketplace
    ON contractors (trade, suburb)
    WHERE active = true AND vetted = true AND public_profile = true;

ALTER TABLE contractors ENABLE ROW LEVEL SECURITY;

CREATE TABLE scheme_contractors (
    scheme_id     UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    contractor_id UUID        NOT NULL REFERENCES contractors(id) ON DELETE CASCADE,
    preferred     BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (scheme_id, contractor_id)
);

CREATE INDEX idx_scheme_contractors_contractor_id
    ON scheme_contractors (contractor_id);

ALTER TABLE scheme_contractors ENABLE ROW LEVEL SECURITY;

CREATE TABLE contractor_reviews (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    contractor_id          UUID        NOT NULL REFERENCES contractors(id) ON DELETE CASCADE,
    scheme_id              UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    maintenance_request_id UUID        NOT NULL REFERENCES maintenance_requests(id) ON DELETE CASCADE,
    rating                 INTEGER     NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment                TEXT,
    created_by_user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (contractor_id, maintenance_request_id)
);

CREATE INDEX idx_contractor_reviews_contractor_created
    ON contractor_reviews (contractor_id, created_at DESC);

CREATE INDEX idx_contractor_reviews_scheme_created
    ON contractor_reviews (scheme_id, created_at DESC);

ALTER TABLE contractor_reviews ENABLE ROW LEVEL SECURITY;

ALTER TABLE maintenance_requests
    ADD COLUMN contractor_id UUID REFERENCES contractors(id) ON DELETE SET NULL;

CREATE INDEX idx_maintenance_requests_contractor_id
    ON maintenance_requests (contractor_id)
    WHERE contractor_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_maintenance_requests_contractor_id;
ALTER TABLE maintenance_requests DROP COLUMN IF EXISTS contractor_id;
DROP INDEX IF EXISTS idx_contractor_reviews_scheme_created;
DROP INDEX IF EXISTS idx_contractor_reviews_contractor_created;
DROP TABLE IF EXISTS contractor_reviews;
DROP INDEX IF EXISTS idx_scheme_contractors_contractor_id;
DROP TABLE IF EXISTS scheme_contractors;
DROP INDEX IF EXISTS idx_contractors_public_marketplace;
DROP INDEX IF EXISTS idx_contractors_org_trade_suburb;
DROP INDEX IF EXISTS idx_contractors_unique_org_name_trade_suburb;
DROP TRIGGER IF EXISTS contractors_set_updated_at ON contractors;
DROP TABLE IF EXISTS contractors;
