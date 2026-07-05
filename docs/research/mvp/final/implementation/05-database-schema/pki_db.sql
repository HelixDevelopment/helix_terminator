-- ============================================================
-- pki_db.sql — HelixTerminator PKI Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: ca_keys
-- Certificate Authority key pairs.
-- ============================================================
CREATE TABLE ca_keys (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  ca_type           VARCHAR(20) NOT NULL CHECK (ca_type IN ('user', 'host')),
  algorithm         VARCHAR(20) NOT NULL CHECK (algorithm IN ('ed25519', 'rsa', 'ecdsa')),
  bits              INTEGER,
  public_key_openssh TEXT NOT NULL,
  encrypted_private_key BYTEA NOT NULL,
  fingerprint       VARCHAR(512) NOT NULL,
  serial_counter    BIGINT NOT NULL DEFAULT 1,
  active            BOOLEAN NOT NULL DEFAULT TRUE,
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  rotated_at        TIMESTAMP WITH TIME ZONE,
  retired_at        TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_ca_keys_org_type_active ON ca_keys(org_id, ca_type) WHERE active = TRUE;

-- ============================================================
-- TABLE: certificates
-- Issued SSH certificates.
-- ============================================================
CREATE TABLE certificates (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  ca_key_id         UUID NOT NULL REFERENCES ca_keys(id) ON DELETE CASCADE,
  entity_type       VARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'host')),
  entity_id         UUID NOT NULL,
  certificate_type  VARCHAR(20) NOT NULL CHECK (certificate_type IN ('user', 'host')),
  certificate_openssh TEXT NOT NULL,
  serial            BIGINT NOT NULL,
  fingerprint       VARCHAR(512) NOT NULL,
  principals        TEXT[] NOT NULL DEFAULT '{}',
  extensions        JSONB DEFAULT '{}',
  critical_options  JSONB DEFAULT '{}',
  valid_after       TIMESTAMP WITH TIME ZONE NOT NULL,
  valid_before      TIMESTAMP WITH TIME ZONE NOT NULL,
  source_address    VARCHAR(255),
  force_command     TEXT,
  revoked           BOOLEAN NOT NULL DEFAULT FALSE,
  revoked_at        TIMESTAMP WITH TIME ZONE,
  revoked_by        UUID,
  revoke_reason     VARCHAR(255),
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_certificates_entity ON certificates(entity_type, entity_id);
CREATE INDEX idx_certificates_valid_before ON certificates(valid_before)
  WHERE revoked = FALSE;
CREATE INDEX idx_certificates_fingerprint ON certificates(fingerprint);

-- ============================================================
-- TABLE: certificate_revocations
-- Revocation records.
-- ============================================================
CREATE TABLE certificate_revocations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID NOT NULL,
  certificate_id  UUID NOT NULL REFERENCES certificates(id) ON DELETE CASCADE,
  serial          BIGINT NOT NULL,
  revoked_by      UUID NOT NULL,
  revoked_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  reason          VARCHAR(255),
  crl_published   BOOLEAN NOT NULL DEFAULT FALSE,
  crl_published_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_cert_revocations_org_id ON certificate_revocations(org_id);
CREATE INDEX idx_cert_revocations_serial ON certificate_revocations(serial);

-- ============================================================
-- TABLE: crl_entries
-- Certificate Revocation List entries.
-- ============================================================
CREATE TABLE crl_entries (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ca_key_id   UUID NOT NULL REFERENCES ca_keys(id) ON DELETE CASCADE,
  serial      BIGINT NOT NULL,
  revoked_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  reason      VARCHAR(255)
);

CREATE INDEX idx_crl_entries_ca_key_id ON crl_entries(ca_key_id);

-- ============================================================
-- RLS Policies (pki_db)
-- ============================================================
ALTER TABLE ca_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE ca_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY ca_keys_org_isolation ON ca_keys
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE certificates ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificates FORCE ROW LEVEL SECURITY;
CREATE POLICY certificates_org_isolation ON certificates
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE certificate_revocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_revocations FORCE ROW LEVEL SECURITY;
CREATE POLICY certificate_revocations_org_isolation ON certificate_revocations
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE crl_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE crl_entries FORCE ROW LEVEL SECURITY;
CREATE POLICY crl_entries_org_isolation ON crl_entries
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
