-- Reverts 001_init.up.sql

DROP TRIGGER IF EXISTS workspaces_updated_at ON workspaces;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_workspace_hosts_host_id;
DROP INDEX IF EXISTS idx_workspace_hosts_workspace_id;
DROP TABLE IF EXISTS workspace_hosts;

DROP INDEX IF EXISTS idx_workspaces_tags;
DROP INDEX IF EXISTS idx_workspaces_deleted_at;
DROP INDEX IF EXISTS idx_workspaces_user_id;
DROP INDEX IF EXISTS idx_workspaces_org_id;
DROP TABLE IF EXISTS workspaces;

-- Note: the "uuid-ossp" extension is deliberately NOT dropped here. It is a
-- database-wide (not schema-scoped) object that other services sharing the
-- same "helixterminator" PostgreSQL database also depend on (idempotently
-- re-declared via their own CREATE EXTENSION IF NOT EXISTS); dropping it on
-- a workspace_service-only rollback would break every other service still
-- relying on uuid_generate_v4().
