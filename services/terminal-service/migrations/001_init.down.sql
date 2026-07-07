-- Reverts 001_init.up.sql

DROP INDEX IF EXISTS idx_terminal_recordings_session_id;
DROP TABLE IF EXISTS terminal_recordings;

DROP INDEX IF EXISTS idx_terminal_outputs_sequence;
DROP INDEX IF EXISTS idx_terminal_outputs_session_id;
DROP TABLE IF EXISTS terminal_outputs;

DROP INDEX IF EXISTS idx_terminal_sessions_created_at;
DROP INDEX IF EXISTS idx_terminal_sessions_status;
DROP INDEX IF EXISTS idx_terminal_sessions_host_id;
DROP INDEX IF EXISTS idx_terminal_sessions_user_id;
DROP TABLE IF EXISTS terminal_sessions;
