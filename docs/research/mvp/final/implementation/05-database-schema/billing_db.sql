-- ============================================================
-- billing_db.sql — HelixTerminator Billing Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: subscriptions
-- Organization subscription records.
-- ============================================================
CREATE TABLE subscriptions (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  plan              VARCHAR(50) NOT NULL
                      CHECK (plan IN ('free', 'pro', 'team', 'enterprise', 'self_hosted')),
  status            VARCHAR(20) NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'trialing', 'past_due', 'cancelled', 'paused')),
  stripe_subscription_id VARCHAR(255),
  stripe_customer_id  VARCHAR(255),
  current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
  current_period_end  TIMESTAMP WITH TIME ZONE NOT NULL,
  cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
  trial_start       TIMESTAMP WITH TIME ZONE,
  trial_end         TIMESTAMP WITH TIME ZONE,
  seat_count        INTEGER NOT NULL DEFAULT 1,
  seat_price_cents  INTEGER NOT NULL DEFAULT 0,
  currency          VARCHAR(3) NOT NULL DEFAULT 'USD',
  metadata          JSONB DEFAULT '{}',
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_subscriptions_org_id ON subscriptions(org_id) WHERE status IN ('active', 'trialing', 'past_due');

-- ============================================================
-- TABLE: invoices
-- Billing invoices.
-- ============================================================
CREATE TABLE invoices (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  subscription_id   UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
  stripe_invoice_id VARCHAR(255),
  amount_due_cents  INTEGER NOT NULL DEFAULT 0,
  amount_paid_cents INTEGER NOT NULL DEFAULT 0,
  currency          VARCHAR(3) NOT NULL DEFAULT 'USD',
  status            VARCHAR(20) NOT NULL DEFAULT 'draft'
                      CHECK (status IN ('draft', 'open', 'paid', 'uncollectible', 'void')),
  due_date          TIMESTAMP WITH TIME ZONE,
  paid_at           TIMESTAMP WITH TIME ZONE,
  pdf_url           TEXT,
  line_items        JSONB DEFAULT '[]',
  metadata          JSONB DEFAULT '{}',
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_org_id ON invoices(org_id, created_at DESC);
CREATE INDEX idx_invoices_status ON invoices(status) WHERE status IN ('open', 'draft');

-- ============================================================
-- TABLE: usage_records
-- Metered usage records.
-- ============================================================
CREATE TABLE usage_records (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  subscription_id   UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
  metric_name       VARCHAR(100) NOT NULL,
  quantity          BIGINT NOT NULL DEFAULT 0,
  unit              VARCHAR(50) NOT NULL,
  recorded_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  stripe_usage_record_id VARCHAR(255)
);

CREATE INDEX idx_usage_records_org_metric ON usage_records(org_id, metric_name, recorded_at DESC);

-- ============================================================
-- RLS Policies
-- ============================================================
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions FORCE ROW LEVEL SECURITY;
CREATE POLICY subscriptions_org_isolation ON subscriptions
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoices FORCE ROW LEVEL SECURITY;
CREATE POLICY invoices_org_isolation ON invoices
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE usage_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_records FORCE ROW LEVEL SECURITY;
CREATE POLICY usage_records_org_isolation ON usage_records
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
