-- ============================================================
-- helixtrack_db.sql — HelixTerminator HelixTrack Integration Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: helixtrack_links
-- Links between HelixTerminator sessions and HelixTrack issues.
-- ============================================================
CREATE TABLE helixtrack_links (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  user_id           UUID NOT NULL,
  session_id        UUID NOT NULL,
  helixtrack_issue_id VARCHAR(255) NOT NULL,
  helixtrack_project_id VARCHAR(255),
  link_type         VARCHAR(20) NOT NULL DEFAULT 'related'
                      CHECK (link_type IN ('related', 'caused_by', 'fixes', 'blocks')),
  metadata          JSONB DEFAULT '{}',
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_helixtrack_links_org_id ON helixtrack_links(org_id, created_at DESC);
CREATE INDEX idx_helixtrack_links_session_id ON helixtrack_links(session_id);
CREATE INDEX idx_helixtrack_links_issue_id ON helixtrack_links(helixtrack_issue_id);

-- ============================================================
-- TABLE: helixtrack_sync_states
-- Sync state tracking for HelixTrack integration.
-- ============================================================
CREATE TABLE helixtrack_sync_states (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  sync_type     VARCHAR(50) NOT NULL
                  CHECK (sync_type IN ('issues', 'sprints', 'deployments', 'users')),
  last_sync_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  last_sync_status VARCHAR(20) NOT NULL DEFAULT 'success'
                  CHECK (last_sync_status IN ('success', 'partial', 'failed')),
  cursor        TEXT,
  error_message TEXT,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_helixtrack_sync_states_org_type ON helixtrack_sync_states(org_id, sync_type);

-- ============================================================
-- RLS Policies
-- ============================================================
ALTER TABLE helixtrack_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE helixtrack_links FORCE ROW LEVEL SECURITY;
CREATE POLICY helixtrack_links_org_isolation ON helixtrack_links
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE helixtrack_sync_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE helixtrack_sync_states FORCE ROW LEVEL SECURITY;
CREATE POLICY helixtrack_sync_states_org_isolation ON helixtrack_sync_states
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
