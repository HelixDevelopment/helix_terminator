-- ============================================================
-- ai_db.sql — HelixTerminator AI Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: ai_suggestions
-- Command completion suggestions log.
-- ============================================================
CREATE TABLE ai_suggestions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  user_id       UUID NOT NULL,
  session_id    UUID,
  model_name    VARCHAR(100) NOT NULL,
  context_hash  VARCHAR(64) NOT NULL,
  partial_command VARCHAR(512) NOT NULL,
  suggestions   JSONB NOT NULL DEFAULT '[]',
  selected_index INTEGER,
  latency_ms    INTEGER NOT NULL,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_suggestions_user_id ON ai_suggestions(user_id, created_at DESC);
CREATE INDEX idx_ai_suggestions_org_id ON ai_suggestions(org_id, created_at DESC);

-- ============================================================
-- TABLE: ai_models
-- Available AI models configuration.
-- ============================================================
CREATE TABLE ai_models (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID,
  name          VARCHAR(100) NOT NULL,
  model_type    VARCHAR(50) NOT NULL
                  CHECK (model_type IN ('completion', 'explanation', 'anomaly', 'runbook', 'incident')),
  version       VARCHAR(50) NOT NULL,
  endpoint_url  TEXT,
  config        JSONB DEFAULT '{}',
  enabled       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_models_org_type ON ai_models(org_id, model_type);

-- ============================================================
-- TABLE: ai_feedback
-- User feedback on AI suggestions.
-- ============================================================
CREATE TABLE ai_feedback (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  user_id       UUID NOT NULL,
  suggestion_id UUID NOT NULL,
  rating        INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment       TEXT,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_feedback_suggestion_id ON ai_feedback(suggestion_id);
CREATE INDEX idx_ai_feedback_user_id ON ai_feedback(user_id, created_at DESC);

-- ============================================================
-- RLS Policies
-- ============================================================
ALTER TABLE ai_suggestions ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_suggestions FORCE ROW LEVEL SECURITY;
CREATE POLICY ai_suggestions_org_isolation ON ai_suggestions
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE ai_models ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_models FORCE ROW LEVEL SECURITY;
CREATE POLICY ai_models_org_isolation ON ai_models
  USING (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE ai_feedback ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_feedback FORCE ROW LEVEL SECURITY;
CREATE POLICY ai_feedback_org_isolation ON ai_feedback
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
