CREATE TABLE IF NOT EXISTS sftp_sessions (
    id UUID PRIMARY KEY,
    host_id UUID NOT NULL,
    user_id UUID,
    remote_path VARCHAR(1024) NOT NULL,
    local_path VARCHAR(1024) NOT NULL,
    direction VARCHAR(20) NOT NULL DEFAULT 'download',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    bytes_transferred BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sftp_sessions_host_id ON sftp_sessions(host_id);
CREATE INDEX IF NOT EXISTS idx_sftp_sessions_user_id ON sftp_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sftp_sessions_status ON sftp_sessions(status);
CREATE INDEX IF NOT EXISTS idx_sftp_sessions_created_at ON sftp_sessions(created_at DESC);
