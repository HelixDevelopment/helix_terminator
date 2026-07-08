-- Reverts 001_init.up.sql

DROP TRIGGER IF EXISTS ai_requests_updated_at ON ai_requests;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_ai_requests_status;
DROP INDEX IF EXISTS idx_ai_requests_org_id;
DROP INDEX IF EXISTS idx_ai_requests_user_id;
DROP TABLE IF EXISTS ai_requests;
