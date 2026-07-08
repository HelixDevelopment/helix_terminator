-- Reverts 001_init.up.sql

DROP INDEX IF EXISTS idx_container_bridges_created_at;
DROP INDEX IF EXISTS idx_container_bridges_status;
DROP INDEX IF EXISTS idx_container_bridges_host_id;
DROP TABLE IF EXISTS container_bridges;
