-- ============================================================
-- config_db.sql — HelixTerminator Configuration Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: feature_flags
-- Centralized feature flags.
-- ============================================================
CREATE TABLE feature_flags (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID,
  key           VARCHAR(255) NOT NULL,
  name          VARCHAR(255) NOT NULL,
  description   TEXT,
  value_type    VARCHAR(20) NOT NULL
                  CHECK (value_type IN ('boolean', 'string', 'integer', 'float', 'json')),
  value         JSONB NOT NULL,
  default_value JSONB NOT NULL,
  enabled       BOOLEAN NOT NULL DEFAULT TRUE,
  rollout_percentage INTEGER NOT NULL DEFAULT 100
                  CHECK (rollout_percentage BETWEEN 0 AND 100),
  target_groups JSONB DEFAULT '[]',
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at    TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_feature_flags_org_key ON feature_flags(org_id, key) WHERE deleted_at IS NULL;
CREATE INDEX idx_feature_flags_org_id ON feature_flags(org_id) WHERE deleted_at IS NULL;

-- ============================================================
-- TABLE: operational_parameters
-- Runtime operational parameters.
-- ============================================================
CREATE TABLE operational_parameters (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID,
  key           VARCHAR(255) NOT NULL,
  value         JSONB NOT NULL,
  value_type    VARCHAR(20) NOT NULL,
  description   TEXT,
  updated_by    UUID NOT NULL,
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_operational_params_org_key ON operational_parameters(org_id, key);

-- ============================================================
-- TABLE: config_audit_log
-- Audit log for configuration changes.
-- ============================================================
CREATE TABLE config_audit_log (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID,
  key           VARCHAR(255) NOT NULL,
  old_value     JSONB,
  new_value     JSONB NOT NULL,
  changed_by    UUID NOT NULL,
  change_reason TEXT,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_config_audit_log_org_key ON config_audit_log(org_id, key, created_at DESC);
CREATE INDEX idx_config_audit_log_created_at ON config_audit_log USING BRIN (created_at);

-- ============================================================
-- RLS Policies
-- ============================================================
ALTER TABLE feature_flags ENABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flags FORCE ROW LEVEL SECURITY;
CREATE POLICY feature_flags_org_isolation ON feature_flags
  USING (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE operational_parameters ENABLE ROW LEVEL SECURITY;
ALTER TABLE operational_parameters FORCE ROW LEVEL SECURITY;
CREATE POLICY operational_parameters_org_isolation ON operational_parameters
  USING (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE config_audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY config_audit_log_org_isolation ON config_audit_log
  USING (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id IS NULL OR org_id = current_setting('app.current_org', true)::uuid);
