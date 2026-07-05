-- 001_init.sql
-- Create workspaces and workspace_hosts tables with indexes and triggers.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL,
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    color VARCHAR(7),
    icon VARCHAR(50),
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_workspaces_org_id ON workspaces(org_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_user_id ON workspaces(user_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_deleted_at ON workspaces(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_workspaces_tags ON workspaces USING GIN(tags);

CREATE TABLE IF NOT EXISTS workspace_hosts (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    host_id UUID NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by UUID NOT NULL,
    PRIMARY KEY (workspace_id, host_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_hosts_workspace_id ON workspace_hosts(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_hosts_host_id ON workspace_hosts(host_id);

-- Trigger function to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on workspaces
DROP TRIGGER IF EXISTS workspaces_updated_at ON workspaces;
CREATE TRIGGER workspaces_updated_at
    BEFORE UPDATE ON workspaces
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
