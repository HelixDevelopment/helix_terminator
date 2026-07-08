-- Reverts 001_init.up.sql

DROP TRIGGER IF EXISTS certificates_updated_at ON certificates;
DROP TRIGGER IF EXISTS certificate_authorities_updated_at ON certificate_authorities;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_certificates_not_after;
DROP INDEX IF EXISTS idx_certificates_status;
DROP INDEX IF EXISTS idx_certificates_org_id;
DROP INDEX IF EXISTS idx_certificates_ca_id;
DROP TABLE IF EXISTS certificates;

DROP INDEX IF EXISTS idx_certificate_authorities_deleted_at;
DROP INDEX IF EXISTS idx_certificate_authorities_org_id;
DROP TABLE IF EXISTS certificate_authorities;
