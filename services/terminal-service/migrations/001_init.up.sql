-- 001_init.sql
-- Create tables, indexes, and constraints for terminal-service

CREATE TABLE IF NOT EXISTS terminal_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    host_id UUID NOT NULL,
    ssh_session_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    cols INTEGER NOT NULL DEFAULT 80,
    rows INTEGER NOT NULL DEFAULT 24,
    shell_type VARCHAR(50) NOT NULL DEFAULT '/bin/bash',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_terminal_sessions_user_id ON terminal_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_host_id ON terminal_sessions(host_id);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_status ON terminal_sessions(status);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_created_at ON terminal_sessions(created_at DESC);

CREATE TABLE IF NOT EXISTS terminal_outputs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES terminal_sessions(id) ON DELETE CASCADE,
    output_type VARCHAR(20) NOT NULL,
    data BYTEA NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sequence_num INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_terminal_outputs_session_id ON terminal_outputs(session_id);
CREATE INDEX IF NOT EXISTS idx_terminal_outputs_sequence ON terminal_outputs(session_id, sequence_num);

CREATE TABLE IF NOT EXISTS terminal_recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES terminal_sessions(id) ON DELETE CASCADE,
    format VARCHAR(20) NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_terminal_recordings_session_id ON terminal_recordings(session_id);
