-- ============================================================
-- collab_db.sql — HelixTerminator Collaboration Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: collaboration_sessions
-- Collaborative terminal session metadata.
-- ============================================================
CREATE TABLE collaboration_sessions (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ssh_session_id    UUID NOT NULL UNIQUE,
  org_id            UUID NOT NULL,
  owner_id          UUID NOT NULL,
  title             VARCHAR(255),
  max_participants  SMALLINT NOT NULL DEFAULT 10,
  allow_input       BOOLEAN NOT NULL DEFAULT FALSE,
  status            VARCHAR(20) NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'ended')),
  started_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  ended_at          TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_collab_sessions_ssh_session ON collaboration_sessions(ssh_session_id);
CREATE INDEX idx_collab_sessions_org_id ON collaboration_sessions(org_id, started_at DESC);

-- ============================================================
-- TABLE: collaboration_participants
-- Participants in a collaboration session.
-- ============================================================
CREATE TABLE collaboration_participants (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  collab_id       UUID NOT NULL REFERENCES collaboration_sessions(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL,
  display_name    VARCHAR(100) NOT NULL,
  role            VARCHAR(20) NOT NULL DEFAULT 'viewer'
                    CHECK (role IN ('owner', 'contributor', 'viewer')),
  cursor_color    VARCHAR(20),
  connected_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  disconnected_at TIMESTAMP WITH TIME ZONE,
  is_active       BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_collab_participants_collab_id ON collaboration_participants(collab_id)
  WHERE is_active = TRUE;
CREATE INDEX idx_collab_participants_user_id ON collaboration_participants(user_id);

-- ============================================================
-- TABLE: collaboration_events
-- Event log for collaboration sessions.
-- ============================================================
CREATE TABLE collaboration_events (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  collab_id     UUID NOT NULL,
  user_id       UUID NOT NULL,
  event_type    VARCHAR(50) NOT NULL
                  CHECK (event_type IN ('chat', 'cursor', 'join', 'leave', 'control_request', 'control_granted')),
  payload       JSONB DEFAULT '{}',
  occurred_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (occurred_at);

CREATE INDEX idx_collab_events_collab_id ON collaboration_events(collab_id, occurred_at);
CREATE INDEX idx_collab_events_occurred_at ON collaboration_events USING BRIN (occurred_at);

CREATE TABLE collaboration_events_2026_q2 PARTITION OF collaboration_events
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE collaboration_events_2026_q3 PARTITION OF collaboration_events
  FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');

-- ============================================================
-- RLS Policies (collab_db — org-scoped)
-- ============================================================
ALTER TABLE collaboration_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE collaboration_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY collaboration_sessions_org_isolation ON collaboration_sessions
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE collaboration_participants ENABLE ROW LEVEL SECURITY;
ALTER TABLE collaboration_participants FORCE ROW LEVEL SECURITY;
CREATE POLICY collaboration_participants_org_isolation ON collaboration_participants
  USING (collab_id IN (SELECT id FROM collaboration_sessions WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (collab_id IN (SELECT id FROM collaboration_sessions WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE collaboration_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE collaboration_events FORCE ROW LEVEL SECURITY;
CREATE POLICY collaboration_events_org_isolation ON collaboration_events
  USING (collab_id IN (SELECT id FROM collaboration_sessions WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (collab_id IN (SELECT id FROM collaboration_sessions WHERE org_id = current_setting('app.current_org', true)::uuid));
