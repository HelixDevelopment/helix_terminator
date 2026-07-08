-- 001_init.sql
-- Create secrets and secret_versions tables with indexes, triggers, foreign keys.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS secrets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    org_id UUID,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('ssh_key', 'api_token', 'password', 'certificate', 'env_var')),
    encrypted_value TEXT NOT NULL,
    iv TEXT NOT NULL,
    salt TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_secrets_user_id ON secrets(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_secrets_org_id ON secrets(org_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_secrets_type ON secrets(type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_secrets_user_type ON secrets(user_id, type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_secrets_tags ON secrets USING GIN(tags) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_secrets_deleted_at ON secrets(deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS secret_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    secret_id UUID NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
    encrypted_value TEXT NOT NULL,
    iv TEXT NOT NULL,
    salt TEXT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_secret_versions_secret_id ON secret_versions(secret_id);
CREATE INDEX IF NOT EXISTS idx_secret_versions_created_at ON secret_versions(created_at DESC);

-- Trigger to auto-update updated_at on secrets
CREATE OR REPLACE FUNCTION update_secrets_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_secrets_updated_at ON secrets;
CREATE TRIGGER trg_secrets_updated_at
    BEFORE UPDATE ON secrets
    FOR EACH ROW
    EXECUTE FUNCTION update_secrets_updated_at();
