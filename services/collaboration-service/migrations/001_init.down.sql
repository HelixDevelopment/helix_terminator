-- Reverts 001_init.up.sql

DROP INDEX IF EXISTS idx_collaboration_sessions_ended_at;
DROP INDEX IF EXISTS idx_collaboration_sessions_status;
DROP INDEX IF EXISTS idx_collaboration_sessions_org_id;
DROP INDEX IF EXISTS idx_collaboration_sessions_created_by;
DROP INDEX IF EXISTS idx_collaboration_sessions_host_id;
DROP TABLE IF EXISTS collaboration_sessions;
