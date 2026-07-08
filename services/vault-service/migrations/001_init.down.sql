-- Reverts 001_init.up.sql
-- Note: the "uuid-ossp" extension created by 001_init.up.sql is
-- intentionally NOT dropped here - it is a shared, database-wide
-- extension that other schemas/services may also depend on; dropping
-- it is out of scope for a single service's schema teardown.

DROP TRIGGER IF EXISTS trg_secrets_updated_at ON secrets;
DROP FUNCTION IF EXISTS update_secrets_updated_at();

DROP INDEX IF EXISTS idx_secret_versions_created_at;
DROP INDEX IF EXISTS idx_secret_versions_secret_id;
DROP TABLE IF EXISTS secret_versions;

DROP INDEX IF EXISTS idx_secrets_deleted_at;
DROP INDEX IF EXISTS idx_secrets_tags;
DROP INDEX IF EXISTS idx_secrets_user_type;
DROP INDEX IF EXISTS idx_secrets_type;
DROP INDEX IF EXISTS idx_secrets_org_id;
DROP INDEX IF EXISTS idx_secrets_user_id;
DROP TABLE IF EXISTS secrets;
