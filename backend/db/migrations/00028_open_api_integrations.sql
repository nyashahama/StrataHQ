-- +goose Up

CREATE TABLE integration_api_clients (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name               TEXT        NOT NULL,
    key_prefix         TEXT        NOT NULL UNIQUE,
    key_hash           TEXT        NOT NULL UNIQUE,
    scopes             TEXT[]      NOT NULL DEFAULT ARRAY['read:schemes', 'read:levies', 'read:financials'],
    created_by_user_id UUID        REFERENCES users(id) ON DELETE SET NULL,
    expires_at         TIMESTAMPTZ,
    revoked_at         TIMESTAMPTZ,
    last_used_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (array_length(scopes, 1) IS NOT NULL)
);

CREATE TRIGGER integration_api_clients_set_updated_at
    BEFORE UPDATE ON integration_api_clients
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_integration_api_clients_org_id
    ON integration_api_clients (org_id);

CREATE INDEX idx_integration_api_clients_key_prefix
    ON integration_api_clients (key_prefix);

CREATE TABLE integration_api_client_schemes (
    client_id UUID NOT NULL REFERENCES integration_api_clients(id) ON DELETE CASCADE,
    scheme_id UUID NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    PRIMARY KEY (client_id, scheme_id)
);

CREATE INDEX idx_integration_api_client_schemes_scheme_id
    ON integration_api_client_schemes (scheme_id);

-- +goose Down

DROP INDEX IF EXISTS idx_integration_api_client_schemes_scheme_id;
DROP TABLE IF EXISTS integration_api_client_schemes;
DROP INDEX IF EXISTS idx_integration_api_clients_key_prefix;
DROP INDEX IF EXISTS idx_integration_api_clients_org_id;
DROP TRIGGER IF EXISTS integration_api_clients_set_updated_at ON integration_api_clients;
DROP TABLE IF EXISTS integration_api_clients;
