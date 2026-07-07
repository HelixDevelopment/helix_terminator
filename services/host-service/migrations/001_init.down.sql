-- Reverts 001_init.up.sql

DROP TRIGGER IF EXISTS trigger_hosts_updated_at ON hosts;
DROP FUNCTION IF EXISTS update_hosts_updated_at();

DROP INDEX IF EXISTS idx_host_connection_logs_created_at;
DROP INDEX IF EXISTS idx_host_connection_logs_host_id;
DROP TABLE IF EXISTS host_connection_logs;

DROP INDEX IF EXISTS idx_hosts_deleted_at;
DROP INDEX IF EXISTS idx_hosts_tags;
DROP INDEX IF EXISTS idx_hosts_status;
DROP INDEX IF EXISTS idx_hosts_org_id;
DROP INDEX IF EXISTS idx_hosts_user_id;
DROP TABLE IF EXISTS hosts;

DROP TYPE IF EXISTS connection_status;
DROP TYPE IF EXISTS auth_type;
