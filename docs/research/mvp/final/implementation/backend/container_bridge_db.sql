-- ============================================================
-- container_bridge_db.sql — HelixTerminator Container Bridge Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- TABLE: kubernetes_clusters
-- Registered K8s clusters.
-- ============================================================
CREATE TABLE kubernetes_clusters (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL,
  name              VARCHAR(255) NOT NULL,
  kubeconfig        BYTEA NOT NULL,
  context_name      VARCHAR(255) NOT NULL,
  api_server_url    TEXT NOT NULL,
  ca_cert           BYTEA,
  namespace         VARCHAR(255) NOT NULL DEFAULT 'default',
  status            VARCHAR(20) NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'inactive', 'error')),
  last_connected_at TIMESTAMP WITH TIME ZONE,
  metadata          JSONB DEFAULT '{}',
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kubernetes_clusters_org_id ON kubernetes_clusters(org_id) WHERE status = 'active';

-- ============================================================
-- TABLE: container_registries
-- Docker/Podman registry registrations.
-- ============================================================
CREATE TABLE container_registries (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  name          VARCHAR(255) NOT NULL,
  registry_url  TEXT NOT NULL,
  registry_type VARCHAR(20) NOT NULL
                  CHECK (registry_type IN ('docker_hub', 'harbor', 'ecr', 'gcr', 'acr', 'private')),
  auth_config   JSONB DEFAULT '{}',
  status        VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_container_registries_org_id ON container_registries(org_id);

-- ============================================================
-- TABLE: pod_sessions
-- Container exec/shell session records.
-- ============================================================
CREATE TABLE pod_sessions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL,
  user_id       UUID NOT NULL,
  cluster_id    UUID NOT NULL REFERENCES kubernetes_clusters(id) ON DELETE CASCADE,
  namespace     VARCHAR(255) NOT NULL,
  pod_name      VARCHAR(255) NOT NULL,
  container_name VARCHAR(255),
  command       TEXT NOT NULL DEFAULT '/bin/sh',
  status        VARCHAR(20) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'closed', 'error')),
  started_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  ended_at      TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_pod_sessions_org_id ON pod_sessions(org_id, started_at DESC);
CREATE INDEX idx_pod_sessions_user_id ON pod_sessions(user_id, started_at DESC);

-- ============================================================
-- RLS Policies
-- ============================================================
ALTER TABLE kubernetes_clusters ENABLE ROW LEVEL SECURITY;
ALTER TABLE kubernetes_clusters FORCE ROW LEVEL SECURITY;
CREATE POLICY kubernetes_clusters_org_isolation ON kubernetes_clusters
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE container_registries ENABLE ROW LEVEL SECURITY;
ALTER TABLE container_registries FORCE ROW LEVEL SECURITY;
CREATE POLICY container_registries_org_isolation ON container_registries
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE pod_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE pod_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY pod_sessions_org_isolation ON pod_sessions
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);
