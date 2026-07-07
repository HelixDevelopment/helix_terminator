-- ============================================================
-- analytics_db.sql — HelixTerminator Analytics Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: metrics
-- Time-series metrics aggregation.
-- ============================================================
CREATE TABLE metrics (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  metric_name   VARCHAR(255) NOT NULL,
  metric_type   VARCHAR(50) NOT NULL
                  CHECK (metric_type IN ('counter', 'gauge', 'histogram', 'summary')),
  value         DOUBLE PRECISION NOT NULL,
  labels        JSONB DEFAULT '{}',
  bucket        TIMESTAMP WITH TIME ZONE NOT NULL,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (bucket);

CREATE INDEX idx_metrics_org_name ON metrics(org_id, metric_name, bucket DESC);
CREATE INDEX idx_metrics_bucket ON metrics USING BRIN (bucket);

CREATE TABLE metrics_2026_01 PARTITION OF metrics
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE metrics_2026_02 PARTITION OF metrics
  FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE metrics_2026_03 PARTITION OF metrics
  FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

-- ============================================================
-- TABLE: slo_targets
-- SLO target definitions.
-- ============================================================
CREATE TABLE slo_targets (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  service_name  VARCHAR(100) NOT NULL,
  metric_name   VARCHAR(255) NOT NULL,
  target_value  DOUBLE PRECISION NOT NULL,
  window_days   INTEGER NOT NULL DEFAULT 30,
  alert_threshold DOUBLE PRECISION,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_slo_targets_org_service_metric ON slo_targets(org_id, service_name, metric_name);

-- ============================================================
-- TABLE: dashboard_widgets
-- Dashboard widget configurations.
-- ============================================================
CREATE TABLE dashboard_widgets (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  user_id       UUID NOT NULL,
  name          VARCHAR(255) NOT NULL,
  widget_type   VARCHAR(50) NOT NULL,
  config        JSONB NOT NULL DEFAULT '{}',
  position      JSONB NOT NULL DEFAULT '{}',
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dashboard_widgets_org_user ON dashboard_widgets(org_id, user_id);

-- ============================================================
-- RLS Policies
-- ============================================================
ALTER TABLE metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE metrics FORCE ROW LEVEL SECURITY;
CREATE POLICY metrics_org_isolation ON metrics
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE slo_targets ENABLE ROW LEVEL SECURITY;
ALTER TABLE slo_targets FORCE ROW LEVEL SECURITY;
CREATE POLICY slo_targets_org_isolation ON slo_targets
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE dashboard_widgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard_widgets FORCE ROW LEVEL SECURITY;
CREATE POLICY dashboard_widgets_org_isolation ON dashboard_widgets
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
