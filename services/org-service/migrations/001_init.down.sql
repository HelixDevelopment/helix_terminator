-- Reverts 001_init.up.sql

DROP TRIGGER IF EXISTS trg_update_team_member_count ON memberships;
DROP TRIGGER IF EXISTS trg_update_org_member_count ON memberships;
DROP TRIGGER IF EXISTS trg_memberships_updated_at ON memberships;
DROP TRIGGER IF EXISTS trg_teams_updated_at ON teams;
DROP TRIGGER IF EXISTS trg_organizations_updated_at ON organizations;

DROP FUNCTION IF EXISTS update_team_member_count();
DROP FUNCTION IF EXISTS update_org_member_count();
DROP FUNCTION IF EXISTS update_memberships_updated_at();
DROP FUNCTION IF EXISTS update_teams_updated_at();
DROP FUNCTION IF EXISTS update_organizations_updated_at();

DROP INDEX IF EXISTS idx_memberships_role;
DROP INDEX IF EXISTS idx_memberships_team_id;
DROP INDEX IF EXISTS idx_memberships_user_id;
DROP INDEX IF EXISTS idx_memberships_org_id;
DROP TABLE IF EXISTS memberships;

DROP INDEX IF EXISTS idx_teams_org_id;
DROP TABLE IF EXISTS teams;

DROP INDEX IF EXISTS idx_organizations_deleted_at;
DROP INDEX IF EXISTS idx_organizations_owner_id;
DROP INDEX IF EXISTS idx_organizations_slug;
DROP TABLE IF EXISTS organizations;
