-- 001_init.sql
-- Host service initial migration

CREATE TYPE auth_type AS ENUM ('password', 'key', 'agent', 'vault_key');
CREATE TYPE connection_status AS ENUM ('unknown', 'online', 'offline', 'error');

CREATE TABLE IF NOT EXISTS hosts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    org_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 22,
    username VARCHAR(255) NOT NULL,
    auth_type auth_type NOT NULL DEFAULT 'password',
    vault_secret_id UUID,
    connection_params JSONB,
    tags TEXT[],
    last_connected_at TIMESTAMPTZ,
    connection_status connection_status NOT NULL DEFAULT 'unknown',
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_hosts_user_id ON hosts(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_org_id ON hosts(org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_status ON hosts(connection_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_tags ON hosts USING GIN(tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_deleted_at ON hosts(deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS host_connection_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_host_connection_logs_host_id ON host_connection_logs(host_id);
CREATE INDEX idx_host_connection_logs_created_at ON host_connection_logs(created_at DESC);

-- Trigger to update updated_at on hosts
CREATE OR REPLACE FUNCTION update_hosts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_hosts_updated_at
    BEFORE UPDATE ON hosts
    FOR EACH ROW
    EXECUTE FUNCTION update_hosts_updated_at();
