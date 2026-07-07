-- Reverts 001_init.up.sql

DROP TRIGGER IF EXISTS configs_updated_at ON configs;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_configs_deleted_at;
DROP INDEX IF EXISTS idx_configs_key;
DROP INDEX IF EXISTS idx_configs_scope_id;
DROP INDEX IF EXISTS idx_configs_scope;
DROP INDEX IF EXISTS idx_configs_scope_scope_id_key;
DROP TABLE IF EXISTS configs;
