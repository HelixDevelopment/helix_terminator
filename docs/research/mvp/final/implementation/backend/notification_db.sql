-- ============================================================
-- notification_db.sql — HelixTerminator Notification Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: notifications
-- Multi-channel notification delivery records.
-- ============================================================
CREATE TABLE notifications (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  user_id           UUID NOT NULL,
  type              VARCHAR(50) NOT NULL
                      CHECK (type IN ('email', 'push', 'in_app', 'slack', 'webhook', 'sms')),
  channel           VARCHAR(50) NOT NULL,
  subject           VARCHAR(255) NOT NULL,
  body              TEXT NOT NULL,
  template_id       UUID,
  priority          VARCHAR(20) NOT NULL DEFAULT 'normal'
                      CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
  status            VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'bounced')),
  error_message     TEXT,
  sent_at           TIMESTAMP WITH TIME ZONE,
  delivered_at      TIMESTAMP WITH TIME ZONE,
  read_at           TIMESTAMP WITH TIME ZONE,
  metadata          JSONB DEFAULT '{}',
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_org_id ON notifications(org_id, created_at DESC);
CREATE INDEX idx_notifications_status ON notifications(status) WHERE status IN ('pending', 'failed');

-- ============================================================
-- TABLE: notification_templates
-- Reusable notification templates.
-- ============================================================
CREATE TABLE notification_templates (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID,
  name          VARCHAR(255) NOT NULL,
  description   TEXT,
  type          VARCHAR(50) NOT NULL,
  channel       VARCHAR(50) NOT NULL,
  subject_template VARCHAR(255),
  body_template TEXT NOT NULL,
  variables     JSONB DEFAULT '{}',
  enabled       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notification_templates_org_id ON notification_templates(org_id);

-- ============================================================
-- TABLE: notification_preferences
-- Per-user notification preferences.
-- ============================================================
CREATE TABLE notification_preferences (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL,
  org_id        UUID NOT NULL,
  channel       VARCHAR(50) NOT NULL,
  enabled       BOOLEAN NOT NULL DEFAULT TRUE,
  quiet_hours_start TIME,
  quiet_hours_end   TIME,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_notification_prefs_user_channel ON notification_preferences(user_id, channel);

-- ============================================================
-- RLS Policies
-- ============================================================
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications FORCE ROW LEVEL SECURITY;
CREATE POLICY notifications_org_isolation ON notifications
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE notification_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_templates FORCE ROW LEVEL SECURITY;
CREATE POLICY notification_templates_org_isolation ON notification_templates
  USING (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences FORCE ROW LEVEL SECURITY;
CREATE POLICY notification_preferences_org_isolation ON notification_preferences
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
