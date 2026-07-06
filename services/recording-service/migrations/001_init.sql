CREATE TABLE IF NOT EXISTS recordings (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL,
    host_id UUID NOT NULL,
    user_id UUID,
    org_id UUID,
    file_path VARCHAR(1024) NOT NULL,
    format VARCHAR(50) NOT NULL DEFAULT 'asciinema',
    status VARCHAR(50) NOT NULL DEFAULT 'recording',
    duration_sec INTEGER NOT NULL DEFAULT 0,
    file_size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recordings_session_id ON recordings(session_id);
CREATE INDEX IF NOT EXISTS idx_recordings_host_id ON recordings(host_id);
CREATE INDEX IF NOT EXISTS idx_recordings_user_id ON recordings(user_id);
CREATE INDEX IF NOT EXISTS idx_recordings_org_id ON recordings(org_id);
CREATE INDEX IF NOT EXISTS idx_recordings_status ON recordings(status);
CREATE INDEX IF NOT EXISTS idx_recordings_created_at ON recordings(created_at DESC);
