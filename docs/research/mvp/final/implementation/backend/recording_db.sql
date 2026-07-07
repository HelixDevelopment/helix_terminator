-- ============================================================
-- recording_db.sql — HelixTerminator Session Recording Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: recording_metadata
-- Session recording metadata.
-- ============================================================
CREATE TABLE recording_metadata (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id        UUID NOT NULL,
  org_id            UUID NOT NULL,
  user_id           UUID NOT NULL,
  host_id           UUID NOT NULL,
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
  signed            BOOLEAN NOT NULL DEFAULT FALSE,
  signature         BYTEA,
  processed         BOOLEAN NOT NULL DEFAULT FALSE,
  processed_at      TIMESTAMP WITH TIME ZONE,
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recording_metadata_session_id ON recording_metadata(session_id);
CREATE INDEX idx_recording_metadata_org_id ON recording_metadata(org_id, created_at DESC);
CREATE INDEX idx_recording_metadata_user_id ON recording_metadata(user_id, created_at DESC);
CREATE INDEX idx_recording_metadata_created_at ON recording_metadata USING BRIN (created_at);

-- ============================================================
-- TABLE: recording_transcripts
-- Full-text transcript of recordings.
-- ============================================================
CREATE TABLE recording_transcripts (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  recording_id  UUID NOT NULL REFERENCES recording_metadata(id) ON DELETE CASCADE,
  session_id    UUID NOT NULL,
  org_id        UUID NOT NULL,
  transcript    TEXT NOT NULL,
  fts_vector    TSVECTOR,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recording_transcripts_recording_id ON recording_transcripts(recording_id);
CREATE INDEX idx_recording_transcripts_fts ON recording_transcripts USING GIN (fts_vector);

CREATE OR REPLACE FUNCTION recording_transcripts_fts_update()
RETURNS TRIGGER AS $$
BEGIN
  NEW.fts_vector := to_tsvector('english', coalesce(NEW.transcript, ''));
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_recording_transcripts_fts
  BEFORE INSERT OR UPDATE ON recording_transcripts
  FOR EACH ROW EXECUTE FUNCTION recording_transcripts_fts_update();

-- ============================================================
-- RLS Policies
-- ============================================================
ALTER TABLE recording_metadata ENABLE ROW LEVEL SECURITY;
ALTER TABLE recording_metadata FORCE ROW LEVEL SECURITY;
CREATE POLICY recording_metadata_org_isolation ON recording_metadata
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE recording_transcripts ENABLE ROW LEVEL SECURITY;
ALTER TABLE recording_transcripts FORCE ROW LEVEL SECURITY;
CREATE POLICY recording_transcripts_org_isolation ON recording_transcripts
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
