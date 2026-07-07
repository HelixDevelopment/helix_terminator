-- ============================================================
-- snippet_db.sql — HelixTerminator Snippet Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- TABLE: snippet_categories
-- Hierarchical snippet categorization.
-- ============================================================
CREATE TABLE snippet_categories (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id    UUID NOT NULL,
  org_id      UUID NOT NULL,
  parent_id   UUID REFERENCES snippet_categories(id) ON DELETE SET NULL,
  name        VARCHAR(255) NOT NULL,
  description TEXT,
  color       VARCHAR(20),
  icon        VARCHAR(50),
  sort_order  INTEGER NOT NULL DEFAULT 0,
  created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_snippet_categories_vault_id ON snippet_categories(vault_id);
CREATE INDEX idx_snippet_categories_parent_id ON snippet_categories(parent_id);

-- ============================================================
-- TABLE: snippets
-- Command snippets / scripts.
-- ============================================================
CREATE TABLE snippets (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id            UUID NOT NULL,
  org_id              UUID NOT NULL,
  created_by          UUID NOT NULL,
  category_id         UUID REFERENCES snippet_categories(id) ON DELETE SET NULL,
  name                VARCHAR(255) NOT NULL,
  description         TEXT,
  content             TEXT NOT NULL,
  language            VARCHAR(50) NOT NULL DEFAULT 'bash',
  interpreter         VARCHAR(255),
  shebang             VARCHAR(255),
  tags                TEXT[] NOT NULL DEFAULT '{}',
  parameters          JSONB NOT NULL DEFAULT '[]',
  shared              BOOLEAN NOT NULL DEFAULT FALSE,
  pinned              BOOLEAN NOT NULL DEFAULT FALSE,
  executions_count    BIGINT NOT NULL DEFAULT 0,
  last_executed_at    TIMESTAMP WITH TIME ZONE,
  fts_vector          TSVECTOR,
  created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at          TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_snippets_vault_id ON snippets(vault_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_snippets_category_id ON snippets(category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_snippets_tags ON snippets USING GIN (tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_snippets_name_trgm ON snippets USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_snippets_content_trgm ON snippets USING GIN (content gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_snippets_fts ON snippets USING GIN (fts_vector) WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION snippets_fts_update()
RETURNS TRIGGER AS $$
BEGIN
  NEW.fts_vector :=
    setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.description, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(NEW.content, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_snippets_fts
  BEFORE INSERT OR UPDATE ON snippets
  FOR EACH ROW EXECUTE FUNCTION snippets_fts_update();

-- ============================================================
-- TABLE: snippet_executions
-- Log of snippet executions.
-- ============================================================
CREATE TABLE snippet_executions (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  snippet_id        UUID NOT NULL,
  user_id           UUID NOT NULL,
  org_id            UUID NOT NULL,
  execution_mode    VARCHAR(20) NOT NULL DEFAULT 'parallel'
                      CHECK (execution_mode IN ('parallel', 'sequential')),
  host_count        INTEGER NOT NULL DEFAULT 0,
  parameters        JSONB DEFAULT '{}',
  status            VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
  completed_count   INTEGER NOT NULL DEFAULT 0,
  failed_count      INTEGER NOT NULL DEFAULT 0,
  timeout_seconds   INTEGER NOT NULL DEFAULT 60,
  started_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  completed_at      TIMESTAMP WITH TIME ZONE
) PARTITION BY RANGE (started_at);

CREATE INDEX idx_snippet_executions_snippet_id ON snippet_executions(snippet_id);
CREATE INDEX idx_snippet_executions_user_id ON snippet_executions(user_id, started_at DESC);
CREATE INDEX idx_snippet_executions_started_at ON snippet_executions USING BRIN (started_at);

CREATE TABLE snippet_executions_2026_q2 PARTITION OF snippet_executions
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE snippet_executions_2026_q3 PARTITION OF snippet_executions
  FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');

-- ============================================================
-- TABLE: snippet_execution_results
-- Per-host results for each execution.
-- ============================================================
CREATE TABLE snippet_execution_results (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  execution_id    UUID NOT NULL,
  host_id         UUID NOT NULL,
  host_name       VARCHAR(255) NOT NULL,
  session_id      UUID,
  status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'running', 'success', 'failure', 'timeout')),
  exit_code       INTEGER,
  stdout          TEXT,
  stderr          TEXT,
  error_message   TEXT,
  started_at      TIMESTAMP WITH TIME ZONE,
  completed_at    TIMESTAMP WITH TIME ZONE,
  duration_ms     INTEGER
);

CREATE INDEX idx_exec_results_execution_id ON snippet_execution_results(execution_id);

-- ============================================================
-- RLS Policies (snippet_db — org-scoped)
-- ============================================================
ALTER TABLE snippet_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE snippet_categories FORCE ROW LEVEL SECURITY;
CREATE POLICY snippet_categories_org_isolation ON snippet_categories
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE snippets ENABLE ROW LEVEL SECURITY;
ALTER TABLE snippets FORCE ROW LEVEL SECURITY;
CREATE POLICY snippets_org_isolation ON snippets
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE snippet_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE snippet_executions FORCE ROW LEVEL SECURITY;
CREATE POLICY snippet_executions_org_isolation ON snippet_executions
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE snippet_execution_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE snippet_execution_results FORCE ROW LEVEL SECURITY;
CREATE POLICY snippet_execution_results_org_isolation ON snippet_execution_results
  USING (execution_id IN (SELECT id FROM snippet_executions WHERE org_id = current_setting('app.current_org', true)::uuid))
  WITH CHECK (execution_id IN (SELECT id FROM snippet_executions WHERE org_id = current_setting('app.current_org', true)::uuid));
