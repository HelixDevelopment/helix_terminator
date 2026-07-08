-- Reverts 001_init.up.sql

DROP TRIGGER IF EXISTS keychain_items_updated_at ON keychain_items;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_keychain_items_deleted_at;
DROP INDEX IF EXISTS idx_keychain_items_type;
DROP INDEX IF EXISTS idx_keychain_items_org_id;
DROP INDEX IF EXISTS idx_keychain_items_user_id;
DROP TABLE IF EXISTS keychain_items;
