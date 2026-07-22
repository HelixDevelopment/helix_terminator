-- 001_init.sql
-- Create workspaces and workspace_hosts tables with indexes and triggers.
--
-- Uses gen_random_uuid() (built into PostgreSQL 13+ pg_catalog, always
-- resolvable via every service's search_path with no extension) rather
-- than uuid_generate_v4() (from the "uuid-ossp" extension). Found via
-- real-Postgres integration testing (T2): CREATE EXTENSION objects are
-- database-wide, not schema-scoped - in helix_terminator's shared-
-- database, schema-per-service topology (see migrations/migrate.go's
-- doc comment), whichever service's migration runs FIRST installs
-- "uuid-ossp" into ITS OWN schema; every other service's subsequent
-- "CREATE EXTENSION IF NOT EXISTS" is then a silent no-op (the
-- extension already exists by name, database-wide), and if that
-- service's search_path does not include the schema the extension
-- actually landed in, its own uuid_generate_v4() calls fail with
-- "function uuid_generate_v4() does not exist" - reproduced here by
-- running org-service's migration then workspace-service's migration
-- against one shared database.
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
