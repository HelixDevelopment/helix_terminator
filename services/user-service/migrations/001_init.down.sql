-- Reverts 001_init.up.sql

DROP TRIGGER IF EXISTS user_profiles_updated_at ON user_profiles;
DROP TRIGGER IF EXISTS users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_user_profiles_gitlab_id;
DROP INDEX IF EXISTS idx_user_profiles_github_id;
DROP TABLE IF EXISTS user_profiles;

DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_org_id;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
