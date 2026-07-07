-- Reverts 001_init.up.sql

DROP INDEX IF EXISTS idx_ssh_channels_session_id;
DROP TABLE IF EXISTS ssh_channels;

DROP INDEX IF EXISTS idx_ssh_sessions_status;
DROP INDEX IF EXISTS idx_ssh_sessions_host_id;
DROP INDEX IF EXISTS idx_ssh_sessions_user_id;
DROP TABLE IF EXISTS ssh_sessions;
