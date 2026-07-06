CREATE TABLE IF NOT EXISTS port_forwards (
    id UUID PRIMARY KEY,
    host_id UUID NOT NULL,
    local_port INTEGER NOT NULL CHECK (local_port > 0 AND local_port <= 65535),
    remote_port INTEGER NOT NULL CHECK (remote_port > 0 AND remote_port <= 65535),
    remote_host VARCHAR(255) NOT NULL,
    protocol VARCHAR(10) NOT NULL DEFAULT 'tcp',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_port_forwards_host_id ON port_forwards(host_id);
CREATE INDEX IF NOT EXISTS idx_port_forwards_status ON port_forwards(status);
CREATE INDEX IF NOT EXISTS idx_port_forwards_deleted_at ON port_forwards(deleted_at) WHERE deleted_at IS NULL;
