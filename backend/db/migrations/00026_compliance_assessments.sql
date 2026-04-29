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
