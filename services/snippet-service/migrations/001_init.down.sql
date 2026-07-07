-- Reverts 001_init.up.sql

DROP INDEX IF EXISTS idx_snippets_updated_at;
DROP INDEX IF EXISTS idx_snippets_tags;
DROP INDEX IF EXISTS idx_snippets_is_public;
DROP INDEX IF EXISTS idx_snippets_language;
DROP INDEX IF EXISTS idx_snippets_created_by;
DROP INDEX IF EXISTS idx_snippets_org_id;
DROP TABLE IF EXISTS snippets;
