-- ============================================================
-- audit_db.sql — HelixTerminator Audit Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: audit_pii_keys
-- Per-subject Data Encryption Keys for PII envelope encryption.
-- Must be created BEFORE audit_events (FK dependency).
-- ============================================================
CREATE TABLE audit_pii_keys (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  subject_type      VARCHAR(20) NOT NULL DEFAULT 'user'
                      CHECK (subject_type IN ('user', 'service_account', 'anonymous')),
  subject_user_id   UUID,
  wrapped_dek       BYTEA,
  wrap_key_ref      VARCHAR(255) NOT NULL,
  algorithm         VARCHAR(20) NOT NULL DEFAULT 'aes-256-gcm',
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  destroyed_at      TIMESTAMP WITH TIME ZONE,
  destroy_reason    VARCHAR(50) CHECK (destroy_reason IN (
                       'gdpr_erasure_art17', 'key_rotation', 'org_offboarded'
                     ))
);

CREATE UNIQUE INDEX idx_audit_pii_keys_subject ON audit_pii_keys(org_id, subject_user_id)
  WHERE destroyed_at IS NULL AND subject_user_id IS NOT NULL;
CREATE INDEX idx_audit_pii_keys_org_id ON audit_pii_keys(org_id);

-- ============================================================
-- TABLE: audit_events
-- Immutable, cryptographically-chained audit log.
-- Partitioned by month for efficient time-range queries.
-- ============================================================
CREATE TABLE audit_events (
  id              UUID NOT NULL DEFAULT gen_random_uuid(),
  seq             BIGSERIAL,
  org_id          UUID NOT NULL,
  event_type      VARCHAR(100) NOT NULL,
  user_id         UUID,
  resource_type   VARCHAR(50),
  resource_id     UUID,
  outcome         VARCHAR(20) NOT NULL DEFAULT 'success'
                    CHECK (outcome IN ('success', 'failure', 'partial')),
  session_id      UUID,
  source_service  VARCHAR(50) NOT NULL,
  metadata        JSONB NOT NULL DEFAULT '{}',
  ip_address      BYTEA,
  user_agent      BYTEA,
  resource_name   BYTEA,
  pii_key_id      UUID NOT NULL REFERENCES audit_pii_keys(id),
  hash            VARCHAR(128) NOT NULL,
  prev_hash       VARCHAR(128),
  occurred_at     TIMESTAMP WITH TIME ZONE NOT NULL,
  recorded_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id, occurred_at)
) PARTITION BY RANGE (occurred_at);

CREATE INDEX idx_audit_events_org_id ON audit_events(org_id, occurred_at DESC);
CREATE INDEX idx_audit_events_user_id ON audit_events(user_id, occurred_at DESC)
  WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_events_event_type ON audit_events(event_type, occurred_at DESC);
CREATE INDEX idx_audit_events_resource ON audit_events(resource_type, resource_id, occurred_at DESC)
  WHERE resource_type IS NOT NULL;
CREATE INDEX idx_audit_events_occurred_at ON audit_events USING BRIN (occurred_at);
CREATE INDEX idx_audit_events_metadata ON audit_events USING GIN (metadata);
CREATE INDEX idx_audit_events_pii_key_id ON audit_events(pii_key_id);

CREATE TABLE audit_events_2026_01 PARTITION OF audit_events
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_events_2026_02 PARTITION OF audit_events
  FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE audit_events_2026_03 PARTITION OF audit_events
  FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE audit_events_2026_04 PARTITION OF audit_events
  FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE audit_events_2026_05 PARTITION OF audit_events
  FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE audit_events_2026_06 PARTITION OF audit_events
  FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE audit_events_2026_07 PARTITION OF audit_events
  FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE audit_events_2026_08 PARTITION OF audit_events
  FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE audit_events_2026_09 PARTITION OF audit_events
  FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE audit_events_2026_10 PARTITION OF audit_events
  FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE audit_events_2026_11 PARTITION OF audit_events
  FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE audit_events_2026_12 PARTITION OF audit_events
  FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

-- ============================================================
-- TABLE: audit_event_hash_chain
-- Tracks the chain head for tamper detection.
-- ============================================================
CREATE TABLE audit_event_hash_chain (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL UNIQUE,
  last_event_id UUID NOT NULL,
  last_hash     VARCHAR(128) NOT NULL,
  chain_length  BIGINT NOT NULL DEFAULT 0,
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================
-- TABLE: audit_exports
-- Records of audit log export jobs.
-- ============================================================
CREATE TABLE audit_exports (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  requested_by      UUID NOT NULL,
  format            VARCHAR(20) NOT NULL CHECK (format IN ('json', 'csv', 'syslog')),
  filter_json       JSONB NOT NULL DEFAULT '{}',
  status            VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
  event_count       BIGINT DEFAULT 0,
  file_size_bytes   BIGINT DEFAULT 0,
  storage_path      TEXT,
  download_token    VARCHAR(255),
  download_expires_at TIMESTAMP WITH TIME ZONE,
  error_message     TEXT,
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  completed_at      TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_audit_exports_org_id ON audit_exports(org_id, created_at DESC);

-- ============================================================
-- RLS Policies (audit_db)
-- ============================================================
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;
CREATE POLICY audit_events_org_isolation ON audit_events
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE audit_pii_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_pii_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY audit_pii_keys_org_isolation ON audit_pii_keys
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE audit_event_hash_chain ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_event_hash_chain FORCE ROW LEVEL SECURITY;
CREATE POLICY audit_event_hash_chain_org_isolation ON audit_event_hash_chain
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE audit_exports ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_exports FORCE ROW LEVEL SECURITY;
CREATE POLICY audit_exports_org_isolation ON audit_exports
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
