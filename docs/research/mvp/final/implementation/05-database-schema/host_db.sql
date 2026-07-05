-- ============================================================
-- host_db.sql — HelixTerminator Host Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- TABLE: hosts
-- SSH host definitions.
-- ============================================================
CREATE TABLE hosts (
  id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id                    UUID NOT NULL,
  group_id                    UUID,
  org_id                      UUID NOT NULL,
  created_by                  UUID NOT NULL,
  name                        VARCHAR(255) NOT NULL,
  hostname                    VARCHAR(512) NOT NULL,
  port                        INTEGER NOT NULL DEFAULT 22
                                CHECK (port BETWEEN 1 AND 65535),
  username                    VARCHAR(255),
  auth_method                 VARCHAR(20) NOT NULL DEFAULT 'key'
                                CHECK (auth_method IN (
                                  'key', 'password', 'certificate',
                                  'interactive', 'agent', 'pgp'
                                )),
  key_id                      UUID,
  encrypted_password          BYTEA,
  certificate_id              UUID,
  os                          VARCHAR(50),
  os_version                  VARCHAR(100),
  arch                        VARCHAR(20),
  description                 TEXT,
  color                       VARCHAR(20),
  icon                        VARCHAR(50),
  tags                        TEXT[] NOT NULL DEFAULT '{}',
  jump_host_id                UUID,
  proxy_command               TEXT,
  connection_timeout_seconds  INTEGER NOT NULL DEFAULT 30,
  keepalive_interval_seconds  INTEGER NOT NULL DEFAULT 60,
  keepalive_count_max         INTEGER NOT NULL DEFAULT 3,
  server_alive_interval       INTEGER NOT NULL DEFAULT 0,
  compression                 BOOLEAN NOT NULL DEFAULT FALSE,
  cipher_suite                TEXT,
  macs                        TEXT,
  kex_algorithms              TEXT,
  host_key_algorithms         TEXT,
  environment_variables       JSONB NOT NULL DEFAULT '{}',
  startup_snippet_id          UUID,
  status                      VARCHAR(20) NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'inactive', 'unreachable', 'archived')),
  last_connected_at           TIMESTAMP WITH TIME ZONE,
  last_connection_status      VARCHAR(20),
  fingerprint_verified        BOOLEAN NOT NULL DEFAULT FALSE,
  custom_fields               JSONB NOT NULL DEFAULT '{}',
  sort_order                  INTEGER NOT NULL DEFAULT 0,
  created_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at                  TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_hosts_vault_id ON hosts(vault_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_group_id ON hosts(group_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_org_id ON hosts(org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_status ON hosts(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_name_trgm ON hosts USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_hostname_trgm ON hosts USING GIN (hostname gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_tags ON hosts USING GIN (tags);
CREATE INDEX idx_hosts_last_connected_at ON hosts(last_connected_at DESC NULLS LAST) WHERE deleted_at IS NULL;
CREATE INDEX idx_hosts_jump_host_id ON hosts(jump_host_id) WHERE jump_host_id IS NOT NULL;

-- ============================================================
-- TABLE: host_groups
-- Hierarchical host grouping.
-- ============================================================
CREATE TABLE host_groups (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id                        UUID NOT NULL,
  org_id                          UUID NOT NULL,
  parent_id                       UUID REFERENCES host_groups(id) ON DELETE SET NULL,
  name                            VARCHAR(255) NOT NULL,
  description                     TEXT,
  color                           VARCHAR(20),
  icon                            VARCHAR(50),
  default_key_id                  UUID,
  default_username                VARCHAR(255),
  default_port                    INTEGER CHECK (default_port BETWEEN 1 AND 65535),
  default_jump_host_id            UUID,
  default_connection_timeout      INTEGER,
  default_keepalive_interval      INTEGER,
  inherit_from_parent             BOOLEAN NOT NULL DEFAULT TRUE,
  inherit_key                     BOOLEAN NOT NULL DEFAULT TRUE,
  inherit_username                BOOLEAN NOT NULL DEFAULT TRUE,
  inherit_port                    BOOLEAN NOT NULL DEFAULT FALSE,
  inherit_jump_host               BOOLEAN NOT NULL DEFAULT TRUE,
  inherit_environment_variables   BOOLEAN NOT NULL DEFAULT TRUE,
  inherit_startup_snippet         BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order                      INTEGER NOT NULL DEFAULT 0,
  path                            TEXT NOT NULL DEFAULT '',
  depth                           INTEGER NOT NULL DEFAULT 0,
  created_at                      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at                      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at                      TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_host_groups_vault_id ON host_groups(vault_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_host_groups_parent_id ON host_groups(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_host_groups_org_id ON host_groups(org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_host_groups_path ON host_groups(path) WHERE deleted_at IS NULL;
CREATE INDEX idx_host_groups_name_trgm ON host_groups USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL;

-- ============================================================
-- TABLE: host_group_members
-- Many-to-many hosts↔groups.
-- ============================================================
CREATE TABLE host_group_members (
  host_id     UUID NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  group_id    UUID NOT NULL REFERENCES host_groups(id) ON DELETE CASCADE,
  added_by    UUID NOT NULL,
  added_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (host_id, group_id)
);

CREATE INDEX idx_host_group_members_group_id ON host_group_members(group_id);

-- ============================================================
-- TABLE: host_labels
-- Flexible key-value label system for hosts.
-- ============================================================
CREATE TABLE host_labels (
  id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  host_id   UUID NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  key       VARCHAR(100) NOT NULL,
  value     VARCHAR(500) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_host_labels_host_key ON host_labels(host_id, key);
CREATE INDEX idx_host_labels_key_value ON host_labels(key, value);

-- ============================================================
-- TABLE: host_known_fingerprints
-- SSH host key fingerprints for TOFU.
-- ============================================================
CREATE TABLE host_known_fingerprints (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  host_id         UUID NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  algorithm       VARCHAR(20) NOT NULL,
  fingerprint     VARCHAR(512) NOT NULL,
  raw_key         TEXT NOT NULL,
  verified        BOOLEAN NOT NULL DEFAULT FALSE,
  verified_by     UUID,
  verified_at     TIMESTAMP WITH TIME ZONE,
  first_seen_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  last_seen_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  revoked         BOOLEAN NOT NULL DEFAULT FALSE,
  revoked_at      TIMESTAMP WITH TIME ZONE,
  revoke_reason   VARCHAR(255)
);

CREATE UNIQUE INDEX idx_known_fingerprints_host_algo ON host_known_fingerprints(host_id, algorithm)
  WHERE revoked = FALSE;
CREATE INDEX idx_known_fingerprints_fingerprint ON host_known_fingerprints(fingerprint);

-- ============================================================
-- TABLE: host_connection_history
-- Log of every SSH connection attempt.
-- ============================================================
CREATE TABLE host_connection_history (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  host_id         UUID NOT NULL,
  user_id         UUID NOT NULL,
  org_id          UUID NOT NULL,
  session_id      UUID,
  client_ip       INET NOT NULL,
  auth_method     VARCHAR(20) NOT NULL,
  key_id          UUID,
  started_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  ended_at        TIMESTAMP WITH TIME ZONE,
  duration_seconds INTEGER,
  bytes_sent      BIGINT NOT NULL DEFAULT 0,
  bytes_received  BIGINT NOT NULL DEFAULT 0,
  exit_code       INTEGER,
  disconnect_reason VARCHAR(255),
  recording_path  TEXT,
  jump_chain      JSONB DEFAULT '[]',
  metadata        JSONB DEFAULT '{}'
) PARTITION BY RANGE (started_at);

CREATE INDEX idx_host_conn_history_host_id ON host_connection_history(host_id, started_at DESC);
CREATE INDEX idx_host_conn_history_user_id ON host_connection_history(user_id, started_at DESC);
CREATE INDEX idx_host_conn_history_started_at ON host_connection_history USING BRIN (started_at);

CREATE TABLE host_connection_history_2026_q2 PARTITION OF host_connection_history
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE host_connection_history_2026_q3 PARTITION OF host_connection_history
  FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');
CREATE TABLE host_connection_history_2026_q4 PARTITION OF host_connection_history
  FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');

-- ============================================================
-- TABLE: jump_host_chains
-- Saved multi-hop jump host configurations.
-- ============================================================
CREATE TABLE jump_host_chains (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id    UUID NOT NULL,
  org_id      UUID NOT NULL,
  name        VARCHAR(255) NOT NULL,
  description TEXT,
  hops        JSONB NOT NULL DEFAULT '[]',
  created_by  UUID NOT NULL,
  created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_jump_chains_vault_id ON jump_host_chains(vault_id);

-- ============================================================
-- RLS Policies (host_db — org-scoped)
-- ============================================================
ALTER TABLE hosts ENABLE ROW LEVEL SECURITY;
ALTER TABLE hosts FORCE ROW LEVEL SECURITY;
CREATE POLICY hosts_org_isolation ON hosts
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE host_groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE host_groups FORCE ROW LEVEL SECURITY;
CREATE POLICY host_groups_org_isolation ON host_groups
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE host_group_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE host_group_members FORCE ROW LEVEL SECURITY;
CREATE POLICY host_group_members_org_isolation ON host_group_members
  USING (host_id IN (SELECT id FROM hosts WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (host_id IN (SELECT id FROM hosts WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE host_labels ENABLE ROW LEVEL SECURITY;
ALTER TABLE host_labels FORCE ROW LEVEL SECURITY;
CREATE POLICY host_labels_org_isolation ON host_labels
  USING (host_id IN (SELECT id FROM hosts WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (host_id IN (SELECT id FROM hosts WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE host_known_fingerprints ENABLE ROW LEVEL SECURITY;
ALTER TABLE host_known_fingerprints FORCE ROW LEVEL SECURITY;
CREATE POLICY host_known_fingerprints_org_isolation ON host_known_fingerprints
  USING (host_id IN (SELECT id FROM hosts WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (host_id IN (SELECT id FROM hosts WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE host_connection_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE host_connection_history FORCE ROW LEVEL SECURITY;
CREATE POLICY host_connection_history_org_isolation ON host_connection_history
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE jump_host_chains ENABLE ROW LEVEL SECURITY;
ALTER TABLE jump_host_chains FORCE ROW LEVEL SECURITY;
CREATE POLICY jump_host_chains_org_isolation ON jump_host_chains
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
