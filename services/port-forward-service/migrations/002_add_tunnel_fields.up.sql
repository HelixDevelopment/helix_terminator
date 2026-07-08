-- Adds the fields required to establish a REAL SSH tunnel (implement
-- real-backends: local/-L, remote/-R, dynamic/-D forwarding) and to track
-- REAL lifecycle status (pending/active/stopped/error) instead of the prior
-- always-"active" placeholder.
ALTER TABLE port_forwards
    ADD COLUMN IF NOT EXISTS forward_type VARCHAR(20) NOT NULL DEFAULT 'local',
    ADD COLUMN IF NOT EXISTS bind_address VARCHAR(255) NOT NULL DEFAULT '127.0.0.1',
    ADD COLUMN IF NOT EXISTS ssh_host VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ssh_port INTEGER NOT NULL DEFAULT 22,
    ADD COLUMN IF NOT EXISTS ssh_username VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS auth_type VARCHAR(20) NOT NULL DEFAULT 'key';

ALTER TABLE port_forwards DROP CONSTRAINT IF EXISTS chk_port_forwards_forward_type;
ALTER TABLE port_forwards ADD CONSTRAINT chk_port_forwards_forward_type CHECK (forward_type IN ('local', 'remote', 'dynamic'));

ALTER TABLE port_forwards DROP CONSTRAINT IF EXISTS chk_port_forwards_auth_type;
ALTER TABLE port_forwards ADD CONSTRAINT chk_port_forwards_auth_type CHECK (auth_type IN ('password', 'key', 'agent'));

ALTER TABLE port_forwards DROP CONSTRAINT IF EXISTS chk_port_forwards_status;
ALTER TABLE port_forwards ADD CONSTRAINT chk_port_forwards_status CHECK (status IN ('pending', 'active', 'inactive', 'stopped', 'error', 'deleted'));

CREATE INDEX IF NOT EXISTS idx_port_forwards_forward_type ON port_forwards(forward_type);
