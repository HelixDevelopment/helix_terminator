-- ============================================================
-- keychain_db.sql — HelixTerminator Keychain Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- TABLE: ssh_keys
-- SSH key metadata (private key stored encrypted).
-- ============================================================
CREATE TABLE ssh_keys (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id              UUID NOT NULL,
  org_id                UUID NOT NULL,
  user_id               UUID NOT NULL,
  name                  VARCHAR(255) NOT NULL,
  type                  VARCHAR(20) NOT NULL DEFAULT 'ssh_key'
                          CHECK (type IN ('ssh_key', 'certificate', 'pgp', 'gpg')),
  algorithm             VARCHAR(20) NOT NULL
                          CHECK (algorithm IN ('ed25519', 'ecdsa', 'rsa', 'dsa', 'ecdsa-sk', 'ed25519-sk')),
  bits                  INTEGER,
  comment               VARCHAR(512),
  fingerprint           VARCHAR(512) NOT NULL,
  public_key_openssh    TEXT NOT NULL,
  encrypted_private_key BYTEA,
  private_key_iv        BYTEA,
  has_passphrase        BOOLEAN NOT NULL DEFAULT FALSE,
  passphrase_protected  BOOLEAN NOT NULL DEFAULT FALSE,
  is_agent_forwarding   BOOLEAN NOT NULL DEFAULT FALSE,
  source                VARCHAR(20) NOT NULL DEFAULT 'generated'
                          CHECK (source IN ('generated', 'imported', 'agent')),
  expires_at            TIMESTAMP WITH TIME ZONE,
  created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at            TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_ssh_keys_vault_id ON ssh_keys(vault_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ssh_keys_user_id ON ssh_keys(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ssh_keys_fingerprint ON ssh_keys(fingerprint);
CREATE INDEX idx_ssh_keys_name_trgm ON ssh_keys USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL;

-- ============================================================
-- TABLE: key_deployments
-- Record of public key deployments to hosts.
-- ============================================================
CREATE TABLE key_deployments (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key_id                UUID NOT NULL REFERENCES ssh_keys(id) ON DELETE CASCADE,
  host_id               UUID NOT NULL,
  host_name             VARCHAR(255) NOT NULL,
  target_user           VARCHAR(255) NOT NULL,
  auth_key_options      JSONB DEFAULT '{}',
  deployed_by           UUID NOT NULL,
  deployed_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  revoked               BOOLEAN NOT NULL DEFAULT FALSE,
  revoked_at            TIMESTAMP WITH TIME ZONE,
  revoked_by            UUID,
  revoke_reason         VARCHAR(255),
  status                VARCHAR(20) NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'revoked', 'expired', 'error'))
);

CREATE INDEX idx_key_deployments_key_id ON key_deployments(key_id);
CREATE INDEX idx_key_deployments_host_id ON key_deployments(host_id);

-- ============================================================
-- TABLE: key_usage_log
-- Immutable log of key usage events.
-- ============================================================
CREATE TABLE key_usage_log (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key_id      UUID NOT NULL,
  host_id     UUID,
  user_id     UUID NOT NULL,
  session_id  UUID,
  event_type  VARCHAR(50) NOT NULL
                CHECK (event_type IN ('auth_success', 'auth_failure', 'sign', 'deploy', 'revoke')),
  ip_address  INET,
  occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (occurred_at);

CREATE INDEX idx_key_usage_key_id ON key_usage_log(key_id);
CREATE INDEX idx_key_usage_occurred_at ON key_usage_log USING BRIN (occurred_at);

CREATE TABLE key_usage_log_2026_q2 PARTITION OF key_usage_log
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE key_usage_log_2026_q3 PARTITION OF key_usage_log
  FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');
CREATE TABLE key_usage_log_2026_q4 PARTITION OF key_usage_log
  FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');

-- ============================================================
-- TABLE: certificate_store
-- SSH certificates issued or stored for use.
-- ============================================================
CREATE TABLE certificate_store (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key_id                UUID REFERENCES ssh_keys(id) ON DELETE SET NULL,
  vault_id              UUID NOT NULL,
  user_id               UUID NOT NULL,
  certificate_type      VARCHAR(20) NOT NULL CHECK (certificate_type IN ('user', 'host')),
  certificate_openssh   TEXT NOT NULL,
  serial                BIGINT NOT NULL,
  fingerprint           VARCHAR(512) NOT NULL,
  principals            TEXT[] NOT NULL DEFAULT '{}',
  extensions            JSONB DEFAULT '{}',
  critical_options      JSONB DEFAULT '{}',
  valid_after           TIMESTAMP WITH TIME ZONE NOT NULL,
  valid_before          TIMESTAMP WITH TIME ZONE NOT NULL,
  signed_by_ca_id       UUID,
  revoked               BOOLEAN NOT NULL DEFAULT FALSE,
  revoked_at            TIMESTAMP WITH TIME ZONE,
  revoke_reason         VARCHAR(255),
  created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_cert_store_serial ON certificate_store(serial);
CREATE INDEX idx_cert_store_key_id ON certificate_store(key_id);
CREATE INDEX idx_cert_store_user_id ON certificate_store(user_id);
CREATE INDEX idx_cert_store_valid_before ON certificate_store(valid_before)
  WHERE revoked = FALSE;

-- ============================================================
-- RLS Policies (keychain_db — org-scoped)
-- ============================================================
ALTER TABLE ssh_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE ssh_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY ssh_keys_org_isolation ON ssh_keys
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE key_deployments ENABLE ROW LEVEL SECURITY;
ALTER TABLE key_deployments FORCE ROW LEVEL SECURITY;
CREATE POLICY key_deployments_org_isolation ON key_deployments
  USING (key_id IN (SELECT id FROM ssh_keys WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (key_id IN (SELECT id FROM ssh_keys WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE key_usage_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE key_usage_log FORCE ROW LEVEL SECURITY;
CREATE POLICY key_usage_log_org_isolation ON key_usage_log
  USING (key_id IN (SELECT id FROM ssh_keys WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (key_id IN (SELECT id FROM ssh_keys WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE certificate_store ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_store FORCE ROW LEVEL SECURITY;
CREATE POLICY certificate_store_org_isolation ON certificate_store
  USING (key_id IN (SELECT id FROM ssh_keys WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (key_id IN (SELECT id FROM ssh_keys WHERE org_id = current_setting('app.current_org', true)::uuid));
