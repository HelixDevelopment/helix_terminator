-- Reverts 001_init.up.sql

DROP INDEX IF EXISTS idx_sftp_sessions_created_at;
DROP INDEX IF EXISTS idx_sftp_sessions_status;
DROP INDEX IF EXISTS idx_sftp_sessions_user_id;
DROP INDEX IF EXISTS idx_sftp_sessions_host_id;
DROP TABLE IF EXISTS sftp_sessions;
