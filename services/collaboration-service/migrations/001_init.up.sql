CREATE TABLE IF NOT EXISTS collaboration_sessions (
    id UUID PRIMARY KEY,
    host_id UUID NOT NULL,
    created_by UUID,
    org_id UUID,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    participants UUID[] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_collaboration_sessions_host_id ON collaboration_sessions(host_id);
CREATE INDEX IF NOT EXISTS idx_collaboration_sessions_created_by ON collaboration_sessions(created_by);
CREATE INDEX IF NOT EXISTS idx_collaboration_sessions_org_id ON collaboration_sessions(org_id);
CREATE INDEX IF NOT EXISTS idx_collaboration_sessions_status ON collaboration_sessions(status);
CREATE INDEX IF NOT EXISTS idx_collaboration_sessions_ended_at ON collaboration_sessions(ended_at) WHERE ended_at IS NULL;
