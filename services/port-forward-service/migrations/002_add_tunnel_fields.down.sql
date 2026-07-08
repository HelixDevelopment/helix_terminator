-- Reverses 002_add_tunnel_fields.up.sql ONLY: drops the CHECK
-- constraints + index it added, then drops the 6 real-SSH-tunnel
-- columns it added, leaving the 001_init port_forwards table (and its
-- original status DEFAULT 'active' with no CHECK constraint) intact.
DROP INDEX IF EXISTS idx_port_forwards_forward_type;

ALTER TABLE port_forwards DROP CONSTRAINT IF EXISTS chk_port_forwards_status;
ALTER TABLE port_forwards DROP CONSTRAINT IF EXISTS chk_port_forwards_auth_type;
ALTER TABLE port_forwards DROP CONSTRAINT IF EXISTS chk_port_forwards_forward_type;

ALTER TABLE port_forwards
    DROP COLUMN IF EXISTS auth_type,
    DROP COLUMN IF EXISTS ssh_username,
    DROP COLUMN IF EXISTS ssh_port,
    DROP COLUMN IF EXISTS ssh_host,
    DROP COLUMN IF EXISTS bind_address,
    DROP COLUMN IF EXISTS forward_type;
