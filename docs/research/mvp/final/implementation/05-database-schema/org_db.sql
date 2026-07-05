-- ============================================================
-- org_db.sql — HelixTerminator Organization/Team Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- ============================================================
-- TABLE: organizations
-- Organizations (tenants).
-- ============================================================
CREATE TABLE organizations (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name                            VARCHAR(255) NOT NULL,
  slug                            CITEXT NOT NULL UNIQUE,
  domain                          CITEXT,
  domain_verified                 BOOLEAN NOT NULL DEFAULT FALSE,
  domain_verified_at              TIMESTAMP WITH TIME ZONE,
  domain_verification_token       VARCHAR(255),
  plan                            VARCHAR(50) NOT NULL DEFAULT 'free'
                                    CHECK (plan IN ('free', 'pro', 'team', 'enterprise')),
  plan_expires_at                 TIMESTAMP WITH TIME ZONE,
  max_members                     INTEGER NOT NULL DEFAULT 5,
  max_vaults                      INTEGER NOT NULL DEFAULT 3,
  max_hosts                       INTEGER NOT NULL DEFAULT 50,
  max_sessions_concurrent         INTEGER NOT NULL DEFAULT 5,
  enforce_mfa                     BOOLEAN NOT NULL DEFAULT FALSE,
  session_recording_required      BOOLEAN NOT NULL DEFAULT FALSE,
  session_recording_retention_days INTEGER NOT NULL DEFAULT 90,
  ip_allowlist                    CIDR[] DEFAULT '{}',
  sso_required                    BOOLEAN NOT NULL DEFAULT FALSE,
  audit_log_retention_days        INTEGER NOT NULL DEFAULT 365,
  owner_id                        UUID NOT NULL,
  billing_email                   CITEXT,
  stripe_customer_id              VARCHAR(255),
  status                          VARCHAR(20) NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active', 'suspended', 'trial', 'cancelled')),
  settings                        JSONB NOT NULL DEFAULT '{}',
  created_at                      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at                      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at                      TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_organizations_slug ON organizations(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_organizations_domain ON organizations(domain) WHERE deleted_at IS NULL AND domain IS NOT NULL;
CREATE INDEX idx_organizations_owner_id ON organizations(owner_id);

-- ============================================================
-- TABLE: org_members
-- Members of organizations.
-- ============================================================
CREATE TABLE org_members (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id       UUID NOT NULL,
  role          VARCHAR(20) NOT NULL DEFAULT 'member'
                  CHECK (role IN ('super_admin', 'org_admin', 'team_admin', 'member', 'auditor', 'api_user')),
  invited_by    UUID,
  joined_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  status        VARCHAR(20) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'suspended', 'pending')),
  last_active_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_org_members_org_user ON org_members(org_id, user_id)
  WHERE status != 'suspended';
CREATE INDEX idx_org_members_user_id ON org_members(user_id);
CREATE INDEX idx_org_members_role ON org_members(org_id, role);

-- ============================================================
-- TABLE: teams
-- Teams within an organization.
-- ============================================================
CREATE TABLE teams (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name          VARCHAR(255) NOT NULL,
  slug          CITEXT NOT NULL,
  description   TEXT,
  settings      JSONB DEFAULT '{}',
  created_by    UUID NOT NULL,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at    TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_teams_org_slug ON teams(org_id, slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_teams_org_id ON teams(org_id) WHERE deleted_at IS NULL;

-- ============================================================
-- TABLE: team_members
-- Members of teams.
-- ============================================================
CREATE TABLE team_members (
  team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  user_id     UUID NOT NULL,
  role        VARCHAR(20) NOT NULL DEFAULT 'member'
                CHECK (role IN ('team_admin', 'member')),
  added_by    UUID NOT NULL,
  added_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (team_id, user_id)
);

CREATE INDEX idx_team_members_user_id ON team_members(user_id);

-- ============================================================
-- TABLE: roles
-- Custom RBAC roles.
-- ============================================================
CREATE TABLE roles (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name          VARCHAR(100) NOT NULL,
  description   TEXT,
  is_system     BOOLEAN NOT NULL DEFAULT FALSE,
  permissions   JSONB NOT NULL DEFAULT '[]',
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_roles_org_name ON roles(org_id, name);
CREATE INDEX idx_roles_permissions ON roles USING GIN (permissions);

-- ============================================================
-- TABLE: role_assignments
-- Assigns roles to users within an org.
-- ============================================================
CREATE TABLE role_assignments (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  user_id       UUID NOT NULL,
  role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  resource_type VARCHAR(50),
  resource_id   UUID,
  granted_by    UUID NOT NULL,
  granted_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  expires_at    TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_role_assignments_org_user ON role_assignments(org_id, user_id);
CREATE INDEX idx_role_assignments_resource ON role_assignments(resource_type, resource_id)
  WHERE resource_type IS NOT NULL;

-- ============================================================
-- TABLE: invitations
-- Pending organization invitations.
-- ============================================================
CREATE TABLE invitations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email           CITEXT NOT NULL,
  role            VARCHAR(20) NOT NULL DEFAULT 'member',
  team_ids        UUID[] DEFAULT '{}',
  invited_by      UUID NOT NULL,
  invitation_token VARCHAR(255) NOT NULL UNIQUE,
  message         TEXT,
  status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'accepted', 'declined', 'expired', 'cancelled')),
  accepted_at     TIMESTAMP WITH TIME ZONE,
  declined_at     TIMESTAMP WITH TIME ZONE,
  expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
  created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invitations_org_email ON invitations(org_id, email)
  WHERE status = 'pending';
CREATE INDEX idx_invitations_token ON invitations(invitation_token);
CREATE INDEX idx_invitations_expires_at ON invitations(expires_at)
  WHERE status = 'pending';

-- ============================================================
-- RLS Policies (org_db)
-- ============================================================
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations FORCE ROW LEVEL SECURITY;
CREATE POLICY organizations_is_current_org ON organizations
  USING (id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (id = current_setting('app.current_org', true)::uuid);

ALTER TABLE org_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_members FORCE ROW LEVEL SECURITY;
CREATE POLICY org_members_org_isolation ON org_members
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE teams ENABLE ROW LEVEL SECURITY;
ALTER TABLE teams FORCE ROW LEVEL SECURITY;
CREATE POLICY teams_org_isolation ON teams
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE team_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE team_members FORCE ROW LEVEL SECURITY;
CREATE POLICY team_members_org_isolation ON team_members
  USING (team_id IN (SELECT id FROM teams WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (team_id IN (SELECT id FROM teams WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles FORCE ROW LEVEL SECURITY;
CREATE POLICY roles_org_isolation ON roles
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE role_assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_assignments FORCE ROW LEVEL SECURITY;
CREATE POLICY role_assignments_org_isolation ON role_assignments
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE invitations FORCE ROW LEVEL SECURITY;
CREATE POLICY invitations_org_isolation ON invitations
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
