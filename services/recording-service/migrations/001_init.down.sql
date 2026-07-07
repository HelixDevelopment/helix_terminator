-- Reverts 001_init.up.sql

DROP INDEX IF EXISTS idx_recordings_created_at;
DROP INDEX IF EXISTS idx_recordings_status;
DROP INDEX IF EXISTS idx_recordings_org_id;
DROP INDEX IF EXISTS idx_recordings_user_id;
DROP INDEX IF EXISTS idx_recordings_host_id;
DROP INDEX IF EXISTS idx_recordings_session_id;
DROP TABLE IF EXISTS recordings;
