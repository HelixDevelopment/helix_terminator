-- ============================================================
-- vault_db.sql — HelixTerminator Vault Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- TABLE: vaults
-- Vault containers (E2E encrypted).
-- ============================================================
CREATE TABLE vaults (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID NOT NULL,
  owner_id        UUID NOT NULL,
  name            VARCHAR(255) NOT NULL,
  description     TEXT,
  color           VARCHAR(20),
  icon            VARCHAR(50),
  encrypted       BOOLEAN NOT NULL DEFAULT TRUE,
  sync_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
  item_count      INTEGER NOT NULL DEFAULT 0,
  storage_bytes   BIGINT NOT NULL DEFAULT 0,
  version         BIGINT NOT NULL DEFAULT 1,
  created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_vaults_org_id ON vaults(org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vaults_owner_id ON vaults(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vaults_name_trgm ON vaults USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL;

-- ============================================================
-- TABLE: vault_members
-- Users with access to each vault.
-- ============================================================
CREATE TABLE vault_members (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id             UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  user_id              UUID NOT NULL,
  permission           VARCHAR(20) NOT NULL CHECK (permission IN ('read', 'write', 'admin')),
  is_owner             BOOLEAN NOT NULL DEFAULT FALSE,
  invited_by           UUID,
  encrypted_vault_key  BYTEA NOT NULL,
  kdf_params           JSONB NOT NULL DEFAULT '{}',
  joined_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_vault_members_vault_user ON vault_members(vault_id, user_id);
CREATE INDEX idx_vault_members_user_id ON vault_members(user_id);

-- ============================================================
-- TABLE: vault_items
-- Individual encrypted items within a vault.
-- ============================================================
CREATE TABLE vault_items (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id        UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  item_type       VARCHAR(50) NOT NULL
                    CHECK (item_type IN (
                      'host', 'ssh_key', 'password', 'note',
                      'certificate', 'totp_secret', 'api_credential', 'file'
                    )),
  encrypted_data  BYTEA NOT NULL,
  checksum        VARCHAR(128) NOT NULL,
  iv              BYTEA NOT NULL,
  version         INTEGER NOT NULL DEFAULT 1,
  is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
  deleted_at      TIMESTAMP WITH TIME ZONE,
  created_by      UUID NOT NULL,
  updated_by      UUID,
  created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vault_items_vault_id ON vault_items(vault_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_vault_items_updated_at ON vault_items USING BRIN (updated_at);

-- ============================================================
-- TABLE: vault_item_versions
-- Version history for vault items.
-- ============================================================
CREATE TABLE vault_item_versions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item_id         UUID NOT NULL REFERENCES vault_items(id) ON DELETE CASCADE,
  vault_id        UUID NOT NULL,
  version         INTEGER NOT NULL,
  encrypted_data  BYTEA NOT NULL,
  checksum        VARCHAR(128) NOT NULL,
  iv              BYTEA NOT NULL,
  changed_by      UUID NOT NULL,
  created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vault_item_versions_item_id ON vault_item_versions(item_id, version DESC);
CREATE INDEX idx_vault_item_versions_vault_id ON vault_item_versions(vault_id);

-- ============================================================
-- TABLE: vault_sync_states
-- Per-client sync cursors for delta synchronization.
-- ============================================================
CREATE TABLE vault_sync_states (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id        UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL,
  client_id       VARCHAR(255) NOT NULL,
  cursor          TEXT NOT NULL DEFAULT '',
  server_version  BIGINT NOT NULL DEFAULT 0,
  last_synced_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  client_platform VARCHAR(100),
  app_version     VARCHAR(50)
);

CREATE UNIQUE INDEX idx_vault_sync_states_vault_client ON vault_sync_states(vault_id, client_id);
CREATE INDEX idx_vault_sync_states_user_id ON vault_sync_states(user_id);

-- ============================================================
-- TABLE: vault_audit_events
-- Vault-level audit log.
-- ============================================================
CREATE TABLE vault_audit_events (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id      UUID NOT NULL,
  item_id       UUID,
  user_id       UUID NOT NULL,
  event_type    VARCHAR(100) NOT NULL,
  ip_address    INET,
  metadata      JSONB DEFAULT '{}',
  occurred_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (occurred_at);

CREATE INDEX idx_vault_audit_vault_id ON vault_audit_events(vault_id);
CREATE INDEX idx_vault_audit_user_id ON vault_audit_events(user_id);
CREATE INDEX idx_vault_audit_occurred_at ON vault_audit_events USING BRIN (occurred_at);

CREATE TABLE vault_audit_events_2026_q2 PARTITION OF vault_audit_events
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE vault_audit_events_2026_q3 PARTITION OF vault_audit_events
  FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');
CREATE TABLE vault_audit_events_2026_q4 PARTITION OF vault_audit_events
  FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');

-- ============================================================
-- RLS Policies (vault_db — org-scoped)
-- ============================================================
ALTER TABLE vaults ENABLE ROW LEVEL SECURITY;
ALTER TABLE vaults FORCE ROW LEVEL SECURITY;
CREATE POLICY vaults_org_isolation ON vaults
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE vault_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE vault_members FORCE ROW LEVEL SECURITY;
CREATE POLICY vault_members_org_isolation ON vault_members
  USING (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE vault_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE vault_items FORCE ROW LEVEL SECURITY;
CREATE POLICY vault_items_org_isolation ON vault_items
  USING (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE vault_item_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE vault_item_versions FORCE ROW LEVEL SECURITY;
CREATE POLICY vault_item_versions_org_isolation ON vault_item_versions
  USING (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE vault_sync_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE vault_sync_states FORCE ROW LEVEL SECURITY;
CREATE POLICY vault_sync_states_org_isolation ON vault_sync_states
  USING (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE vault_audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE vault_audit_events FORCE ROW LEVEL SECURITY;
CREATE POLICY vault_audit_events_org_isolation ON vault_audit_events
  USING (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (vault_id IN (SELECT id FROM vaults WHERE org_id = current_setting('app.current_org', true)::uuid));
