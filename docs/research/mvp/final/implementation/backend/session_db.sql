-- ============================================================
-- session_db.sql — SSH Proxy, Terminal, SFTP, Port Forwarding
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: ssh_sessions
-- SSH terminal session records.
-- ============================================================
CREATE TABLE ssh_sessions (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id               UUID NOT NULL,
  host_id               UUID NOT NULL,
  vault_id              UUID NOT NULL,
  org_id                UUID NOT NULL,
  client_ip             INET NOT NULL,
  user_agent            TEXT,
  terminal_cols         SMALLINT NOT NULL DEFAULT 80,
  terminal_rows         SMALLINT NOT NULL DEFAULT 24,
  terminal_type         VARCHAR(50) NOT NULL DEFAULT 'xterm-256color',
  auth_method           VARCHAR(20),
  key_id                UUID,
  recording_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
  recording_path        TEXT,
  recording_size_bytes  BIGINT DEFAULT 0,
  collab_enabled        BOOLEAN NOT NULL DEFAULT FALSE,
  read_only             BOOLEAN NOT NULL DEFAULT FALSE,
  status                VARCHAR(20) NOT NULL DEFAULT 'connecting'
                          CHECK (status IN (
                            'connecting', 'connected', 'disconnected',
                            'error', 'terminated'
                          )),
  reason                TEXT,
  ticket_ref            VARCHAR(255),
  startup_snippet_id    UUID,
  jump_chain            JSONB DEFAULT '[]',
  exit_code             INTEGER,
  disconnect_reason     TEXT,
  bytes_sent            BIGINT NOT NULL DEFAULT 0,
  bytes_received        BIGINT NOT NULL DEFAULT 0,
  commands_count        INTEGER NOT NULL DEFAULT 0,
  resize_count          INTEGER NOT NULL DEFAULT 0,
  started_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  connected_at          TIMESTAMP WITH TIME ZONE,
  ended_at              TIMESTAMP WITH TIME ZONE,
  duration_seconds      INTEGER
) PARTITION BY RANGE (started_at);

CREATE INDEX idx_ssh_sessions_user_id ON ssh_sessions(user_id, started_at DESC);
CREATE INDEX idx_ssh_sessions_host_id ON ssh_sessions(host_id, started_at DESC);
CREATE INDEX idx_ssh_sessions_org_id ON ssh_sessions(org_id, started_at DESC);
CREATE INDEX idx_ssh_sessions_status ON ssh_sessions(status, started_at DESC)
  WHERE status IN ('connecting', 'connected');
CREATE INDEX idx_ssh_sessions_started_at ON ssh_sessions USING BRIN (started_at);

CREATE TABLE ssh_sessions_2026_q2 PARTITION OF ssh_sessions
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE ssh_sessions_2026_q3 PARTITION OF ssh_sessions
  FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');
CREATE TABLE ssh_sessions_2026_q4 PARTITION OF ssh_sessions
  FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');

-- ============================================================
-- TABLE: session_events
-- Per-event recording of terminal I/O.
-- ============================================================
CREATE TABLE session_events (
  id            BIGSERIAL,
  session_id    UUID NOT NULL,
  occurred_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  elapsed_ms    BIGINT NOT NULL,
  direction     CHAR(1) NOT NULL CHECK (direction IN ('i', 'o')),
  data          BYTEA NOT NULL,
  event_type    VARCHAR(20) NOT NULL DEFAULT 'data'
                  CHECK (event_type IN ('data', 'resize', 'marker', 'metadata'))
) PARTITION BY RANGE (occurred_at);

CREATE INDEX idx_session_events_session_id ON session_events(session_id, occurred_at);
CREATE INDEX idx_session_events_occurred_at ON session_events USING BRIN (occurred_at);

CREATE TABLE session_events_2026_q2 PARTITION OF session_events
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE session_events_2026_q3 PARTITION OF session_events
  FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');

-- ============================================================
-- TABLE: session_recordings
-- Metadata for session recording files.
-- ============================================================
CREATE TABLE session_recordings (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id        UUID NOT NULL,
  storage_path      TEXT NOT NULL,
  storage_backend   VARCHAR(20) NOT NULL DEFAULT 's3'
                      CHECK (storage_backend IN ('s3', 'gcs', 'azure_blob', 'local')),
  file_size_bytes   BIGINT NOT NULL DEFAULT 0,
  duration_seconds  INTEGER,
  format            VARCHAR(20) NOT NULL DEFAULT 'asciicast_v2',
  terminal_cols     SMALLINT NOT NULL,
  terminal_rows     SMALLINT NOT NULL,
  checksum_sha256   VARCHAR(64),
  compressed        BOOLEAN NOT NULL DEFAULT TRUE,
  encryption_key_id UUID,
  processed         BOOLEAN NOT NULL DEFAULT FALSE,
  processed_at      TIMESTAMP WITH TIME ZONE,
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_session_recordings_session_id ON session_recordings(session_id);
CREATE INDEX idx_session_recordings_created_at ON session_recordings USING BRIN (created_at);

-- ============================================================
-- TABLE: sftp_sessions
-- SFTP session records.
-- ============================================================
CREATE TABLE sftp_sessions (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID NOT NULL,
  host_id           UUID NOT NULL,
  org_id            UUID NOT NULL,
  client_ip         INET NOT NULL,
  cwd               TEXT NOT NULL DEFAULT '/',
  transfer_mode     VARCHAR(10) NOT NULL DEFAULT 'binary'
                      CHECK (transfer_mode IN ('binary', 'ascii')),
  server_version    VARCHAR(100),
  status            VARCHAR(20) NOT NULL DEFAULT 'connected'
                      CHECK (status IN ('connected', 'closed', 'error', 'expired')),
  files_uploaded    INTEGER NOT NULL DEFAULT 0,
  files_downloaded  INTEGER NOT NULL DEFAULT 0,
  bytes_uploaded    BIGINT NOT NULL DEFAULT 0,
  bytes_downloaded  BIGINT NOT NULL DEFAULT 0,
  started_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  ended_at          TIMESTAMP WITH TIME ZONE,
  expires_at        TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_sftp_sessions_user_id ON sftp_sessions(user_id);
CREATE INDEX idx_sftp_sessions_host_id ON sftp_sessions(host_id);
CREATE INDEX idx_sftp_sessions_started_at ON sftp_sessions USING BRIN (started_at);

-- ============================================================
-- TABLE: sftp_transfers
-- Individual SFTP file transfer records.
-- ============================================================
CREATE TABLE sftp_transfers (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sftp_session_id   UUID NOT NULL REFERENCES sftp_sessions(id) ON DELETE CASCADE,
  user_id           UUID NOT NULL,
  host_id           UUID NOT NULL,
  direction         VARCHAR(10) NOT NULL CHECK (direction IN ('upload', 'download')),
  local_filename    VARCHAR(1024),
  remote_path       TEXT NOT NULL,
  file_size_bytes   BIGINT NOT NULL DEFAULT 0,
  bytes_transferred BIGINT NOT NULL DEFAULT 0,
  checksum_sha256   VARCHAR(64),
  status            VARCHAR(20) NOT NULL DEFAULT 'completed'
                      CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'cancelled')),
  error_message     TEXT,
  duration_ms       INTEGER,
  transferred_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (transferred_at);

CREATE INDEX idx_sftp_transfers_session_id ON sftp_transfers(sftp_session_id);
CREATE INDEX idx_sftp_transfers_user_id ON sftp_transfers(user_id, transferred_at DESC);
CREATE INDEX idx_sftp_transfers_transferred_at ON sftp_transfers USING BRIN (transferred_at);

CREATE TABLE sftp_transfers_2026_q2 PARTITION OF sftp_transfers
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE sftp_transfers_2026_q3 PARTITION OF sftp_transfers
  FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');

-- ============================================================
-- TABLE: port_forward_rules
-- Port forwarding rule definitions.
-- ============================================================
CREATE TABLE port_forward_rules (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id               UUID NOT NULL,
  host_id               UUID NOT NULL,
  vault_id              UUID NOT NULL,
  org_id                UUID NOT NULL,
  name                  VARCHAR(255) NOT NULL,
  description           TEXT,
  type                  VARCHAR(20) NOT NULL
                          CHECK (type IN ('local', 'remote', 'dynamic')),
  local_address         VARCHAR(255) NOT NULL DEFAULT '127.0.0.1',
  local_port            INTEGER NOT NULL CHECK (local_port BETWEEN 1 AND 65535),
  remote_address        VARCHAR(255),
  remote_port           INTEGER CHECK (remote_port BETWEEN 1 AND 65535),
  bind_address          VARCHAR(255),
  auto_start            BOOLEAN NOT NULL DEFAULT FALSE,
  status                VARCHAR(20) NOT NULL DEFAULT 'inactive'
                          CHECK (status IN ('active', 'inactive', 'error')),
  sort_order            INTEGER NOT NULL DEFAULT 0,
  created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at            TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_pf_rules_user_id ON port_forward_rules(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pf_rules_host_id ON port_forward_rules(host_id) WHERE deleted_at IS NULL;

-- ============================================================
-- TABLE: port_forward_connections
-- Active and historical port forwarding connections.
-- ============================================================
CREATE TABLE port_forward_connections (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id         UUID NOT NULL REFERENCES port_forward_rules(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL,
  host_id         UUID NOT NULL,
  ssh_session_id  UUID,
  status          VARCHAR(20) NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'closed', 'error')),
  bytes_sent      BIGINT NOT NULL DEFAULT 0,
  bytes_received  BIGINT NOT NULL DEFAULT 0,
  started_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  ended_at        TIMESTAMP WITH TIME ZONE,
  error_message   TEXT
);

CREATE INDEX idx_pf_connections_rule_id ON port_forward_connections(rule_id);
CREATE INDEX idx_pf_connections_started_at ON port_forward_connections USING BRIN (started_at);

-- ============================================================
-- RLS Policies (session_db — org-scoped)
-- ============================================================
ALTER TABLE ssh_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE ssh_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY ssh_sessions_org_isolation ON ssh_sessions
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE session_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE session_events FORCE ROW LEVEL SECURITY;
CREATE POLICY session_events_org_isolation ON session_events
  USING (session_id IN (SELECT id FROM ssh_sessions WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (session_id IN (SELECT id FROM ssh_sessions WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE session_recordings ENABLE ROW LEVEL SECURITY;
ALTER TABLE session_recordings FORCE ROW LEVEL SECURITY;
CREATE POLICY session_recordings_org_isolation ON session_recordings
  USING (session_id IN (SELECT id FROM ssh_sessions WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (session_id IN (SELECT id FROM ssh_sessions WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE sftp_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sftp_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY sftp_sessions_org_isolation ON sftp_sessions
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE sftp_transfers ENABLE ROW LEVEL SECURITY;
ALTER TABLE sftp_transfers FORCE ROW LEVEL SECURITY;
CREATE POLICY sftp_transfers_org_isolation ON sftp_transfers
  USING (sftp_session_id IN (SELECT id FROM sftp_sessions WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (sftp_session_id IN (SELECT id FROM sftp_sessions WHERE org_id = current_setting('app.current_org', true)::uuid));

ALTER TABLE port_forward_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE port_forward_rules FORCE ROW LEVEL SECURITY;
CREATE POLICY port_forward_rules_org_isolation ON port_forward_rules
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE port_forward_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE port_forward_connections FORCE ROW LEVEL SECURITY;
CREATE POLICY port_forward_connections_org_isolation ON port_forward_connections
  USING (rule_id IN (SELECT id FROM port_forward_rules WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (rule_id IN (SELECT id FROM port_forward_rules WHERE org_id = current_setting('app.current_org', true)::uuid));
