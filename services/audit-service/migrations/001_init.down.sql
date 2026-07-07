-- Reverts 001_init.up.sql

DROP INDEX IF EXISTS idx_audit_logs_org_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_severity;
DROP INDEX IF EXISTS idx_audit_logs_resource_type;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_org_id;
DROP TABLE IF EXISTS audit_logs;
