-- ============================================================
-- workspace_db.sql — HelixTerminator Workspace Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: workspaces
-- Saved terminal layout configurations.
-- ============================================================
CREATE TABLE workspaces (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL,
  org_id          UUID NOT NULL,
  name            VARCHAR(255) NOT NULL,
  description     TEXT,
  layout          JSONB NOT NULL DEFAULT '{}',
  thumbnail_url   TEXT,
  is_template     BOOLEAN NOT NULL DEFAULT FALSE,
  template_id     UUID,
  pinned          BOOLEAN NOT NULL DEFAULT FALSE,
  auto_connect    BOOLEAN NOT NULL DEFAULT TRUE,
  last_opened_at  TIMESTAMP WITH TIME ZONE,
  open_count      INTEGER NOT NULL DEFAULT 0,
  created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_workspaces_user_id ON workspaces(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspaces_org_id ON workspaces(org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspaces_layout ON workspaces USING GIN (layout) WHERE deleted_at IS NULL;

-- ============================================================
-- TABLE: workspace_snapshots
-- Point-in-time snapshots of workspace layouts.
-- ============================================================
CREATE TABLE workspace_snapshots (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  layout        JSONB NOT NULL,
  snapshot_name VARCHAR(255),
  created_by    UUID NOT NULL,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspace_snapshots_workspace_id ON workspace_snapshots(workspace_id, created_at DESC);

-- ============================================================
-- TABLE: workspace_sessions
-- Maps workspace to the sessions opened within it.
-- ============================================================
CREATE TABLE workspace_sessions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  session_id    UUID NOT NULL,
  pane_id       VARCHAR(50) NOT NULL,
  user_id       UUID NOT NULL,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspace_sessions_workspace_id ON workspace_sessions(workspace_id);
CREATE INDEX idx_workspace_sessions_session_id ON workspace_sessions(session_id);

-- ============================================================
-- TABLE: workspace_templates
-- Reusable workspace layout templates.
-- ============================================================
CREATE TABLE workspace_templates (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_by    UUID NOT NULL,
  org_id        UUID,
  name          VARCHAR(255) NOT NULL,
  description   TEXT,
  category      VARCHAR(100),
  layout        JSONB NOT NULL DEFAULT '{}',
  pane_count    SMALLINT NOT NULL DEFAULT 1,
  preview_url   TEXT,
  public        BOOLEAN NOT NULL DEFAULT FALSE,
  usage_count   INTEGER NOT NULL DEFAULT 0,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspace_templates_org_id ON workspace_templates(org_id);
CREATE INDEX idx_workspace_templates_public ON workspace_templates(public, usage_count DESC)
  WHERE public = TRUE;

-- ============================================================
-- RLS Policies (workspace_db — org-scoped)
-- ============================================================
ALTER TABLE workspaces ENABLE ROW LEVEL SECURITY;
ALTER TABLE workspaces FORCE ROW LEVEL SECURITY;
CREATE POLICY workspaces_org_isolation ON workspaces
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE workspace_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE workspace_snapshots FORCE ROW LEVEL SECURITY;
CREATE POLICY workspace_snapshots_org_isolation ON workspace_snapshots
  USING (workspace_id IN (SELECT id FROM workspaces WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (workspace_id IN (SELECT id FROM workspaces WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE workspace_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE workspace_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY workspace_sessions_org_isolation ON workspace_sessions
  USING (workspace_id IN (SELECT id FROM workspaces WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (workspace_id IN (SELECT id FROM workspaces WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE workspace_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE workspace_templates FORCE ROW LEVEL SECURITY;
CREATE POLICY workspace_templates_org_or_public ON workspace_templates
  USING (
    org_id = current_setting('app.current_org', true)::uuid
    OR (org_id IS NULL AND public = TRUE)
  )
  WITH CHECK (
    org_id = current_setting('app.current_org', true)::uuid
    OR (org_id IS NULL AND public = TRUE)
  );
