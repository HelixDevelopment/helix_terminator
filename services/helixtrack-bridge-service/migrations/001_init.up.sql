CREATE TABLE IF NOT EXISTS helixtrack_bridges (
    id UUID PRIMARY KEY,
    integration_id VARCHAR(255) NOT NULL,
    org_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    config JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_helixtrack_bridges_org_id ON helixtrack_bridges(org_id);
CREATE INDEX IF NOT EXISTS idx_helixtrack_bridges_status ON helixtrack_bridges(status);
CREATE INDEX IF NOT EXISTS idx_helixtrack_bridges_created_at ON helixtrack_bridges(created_at DESC);
