CREATE TABLE IF NOT EXISTS container_bridges (
    id UUID PRIMARY KEY,
    host_id UUID NOT NULL,
    container_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    ports TEXT[] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_container_bridges_host_id ON container_bridges(host_id);
CREATE INDEX IF NOT EXISTS idx_container_bridges_status ON container_bridges(status);
CREATE INDEX IF NOT EXISTS idx_container_bridges_created_at ON container_bridges(created_at DESC);
