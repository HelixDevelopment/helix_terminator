-- ============================================================
-- auth_db.sql — HelixTerminator Auth Service Database Schema
-- PostgreSQL 17.2
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "citext";

-- ============================================================
-- TABLE: users
-- Core user identity. Owns authentication credentials.
-- ============================================================
CREATE TABLE users (
  id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email                     CITEXT NOT NULL,
  email_verified_at         TIMESTAMP WITH TIME ZONE,
  email_pending             CITEXT,
  email_pending_token       VARCHAR(255),
  email_pending_expires_at  TIMESTAMP WITH TIME ZONE,
  password_hash             VARCHAR(255),
  display_name              VARCHAR(100) NOT NULL,
  avatar_url                TEXT,
  bio                       TEXT,
  locale                    VARCHAR(20) NOT NULL DEFAULT 'en-US',
  timezone                  VARCHAR(100) NOT NULL DEFAULT 'UTC',
  status                    VARCHAR(20) NOT NULL DEFAULT 'active'
                              CHECK (status IN ('active', 'suspended', 'pending_deletion', 'deleted')),
  failed_login_attempts     INTEGER NOT NULL DEFAULT 0,
  locked_until              TIMESTAMP WITH TIME ZONE,
  last_login_at             TIMESTAMP WITH TIME ZONE,
  last_login_ip             INET,
  password_changed_at       TIMESTAMP WITH TIME ZONE,
  terms_accepted_at         TIMESTAMP WITH TIME ZONE,
  terms_version             VARCHAR(20),
  deletion_requested_at     TIMESTAMP WITH TIME ZONE,
  deletion_scheduled_at     TIMESTAMP WITH TIME ZONE,
  created_at                TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at                TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at                TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email_pending ON users(email_pending) WHERE email_pending IS NOT NULL;
CREATE INDEX idx_users_status ON users(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users USING BRIN (created_at);
CREATE INDEX idx_users_deletion_scheduled ON users(deletion_scheduled_at)
  WHERE deletion_scheduled_at IS NOT NULL;

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ============================================================
-- TABLE: user_sessions
-- Active authentication sessions.
-- ============================================================
CREATE TABLE user_sessions (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  device_id         UUID,
  ip_address        INET NOT NULL,
  user_agent        TEXT,
  location_city     VARCHAR(100),
  location_country  VARCHAR(10),
  mfa_verified      BOOLEAN NOT NULL DEFAULT FALSE,
  mfa_method        VARCHAR(20) CHECK (mfa_method IN ('totp', 'fido2', 'backup_code', NULL)),
  status            VARCHAR(20) NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'expired', 'revoked')),
  revoked_reason    VARCHAR(100),
  last_active_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  expires_at        TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_status ON user_sessions(status, expires_at)
  WHERE status = 'active';
CREATE INDEX idx_user_sessions_expires_at ON user_sessions USING BRIN (expires_at);

-- ============================================================
-- TABLE: refresh_tokens
-- Refresh token storage for token rotation.
-- ============================================================
CREATE TABLE refresh_tokens (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id      UUID NOT NULL REFERENCES user_sessions(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash      VARCHAR(255) NOT NULL UNIQUE,
  family          UUID NOT NULL,
  generation      INTEGER NOT NULL DEFAULT 1,
  ip_address      INET,
  user_agent      TEXT,
  used_at         TIMESTAMP WITH TIME ZONE,
  revoked         BOOLEAN NOT NULL DEFAULT FALSE,
  revoked_at      TIMESTAMP WITH TIME ZONE,
  revoked_reason  VARCHAR(100),
  created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  expires_at      TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_session_id ON refresh_tokens(session_id);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens USING BRIN (expires_at);

-- ============================================================
-- TABLE: device_tokens
-- Trusted device registrations.
-- ============================================================
CREATE TABLE device_tokens (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name             VARCHAR(255) NOT NULL,
  fingerprint      VARCHAR(512) NOT NULL,
  platform         VARCHAR(255),
  user_agent       TEXT,
  trusted          BOOLEAN NOT NULL DEFAULT TRUE,
  last_seen_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  last_seen_ip     INET,
  revoked          BOOLEAN NOT NULL DEFAULT FALSE,
  revoked_at       TIMESTAMP WITH TIME ZONE,
  created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_device_tokens_user_fingerprint ON device_tokens(user_id, fingerprint)
  WHERE revoked = FALSE;
CREATE INDEX idx_device_tokens_user_id ON device_tokens(user_id);

-- ============================================================
-- TABLE: mfa_totp_credentials
-- TOTP MFA credentials.
-- ============================================================
CREATE TABLE mfa_totp_credentials (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  encrypted_secret TEXT NOT NULL,
  issuer           VARCHAR(100) NOT NULL DEFAULT 'HelixTerminator',
  algorithm        VARCHAR(20) NOT NULL DEFAULT 'SHA1',
  digits           INTEGER NOT NULL DEFAULT 6,
  period           INTEGER NOT NULL DEFAULT 30,
  enabled          BOOLEAN NOT NULL DEFAULT TRUE,
  last_used_at     TIMESTAMP WITH TIME ZONE,
  last_used_code   VARCHAR(10),
  created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_mfa_totp_user_enabled ON mfa_totp_credentials(user_id)
  WHERE enabled = TRUE;

-- ============================================================
-- TABLE: mfa_totp_backup_codes
-- One-time backup codes for TOTP recovery.
-- ============================================================
CREATE TABLE mfa_totp_backup_codes (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash    VARCHAR(255) NOT NULL,
  used         BOOLEAN NOT NULL DEFAULT FALSE,
  used_at      TIMESTAMP WITH TIME ZONE,
  used_ip      INET,
  created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_codes_user_id ON mfa_totp_backup_codes(user_id)
  WHERE used = FALSE;

-- ============================================================
-- TABLE: mfa_fido2_credentials
-- WebAuthn/FIDO2 credential registrations.
-- ============================================================
CREATE TABLE mfa_fido2_credentials (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  credential_id        BYTEA NOT NULL UNIQUE,
  credential_id_b64    TEXT NOT NULL,
  name                 VARCHAR(255) NOT NULL,
  public_key_cbor      BYTEA NOT NULL,
  aaguid               UUID,
  sign_count           BIGINT NOT NULL DEFAULT 0,
  transports           TEXT[] DEFAULT '{}',
  backup_eligible      BOOLEAN NOT NULL DEFAULT FALSE,
  backup_state         BOOLEAN NOT NULL DEFAULT FALSE,
  attestation_type     VARCHAR(50),
  attestation_data     JSONB,
  last_used_at         TIMESTAMP WITH TIME ZONE,
  enabled              BOOLEAN NOT NULL DEFAULT TRUE,
  created_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fido2_user_id ON mfa_fido2_credentials(user_id) WHERE enabled = TRUE;
CREATE INDEX idx_fido2_credential_id ON mfa_fido2_credentials(credential_id_b64);

-- ============================================================
-- TABLE: api_keys
-- API key management.
-- ============================================================
CREATE TABLE api_keys (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  org_id           UUID,
  name             VARCHAR(100) NOT NULL,
  description      TEXT,
  key_hash         VARCHAR(255) NOT NULL UNIQUE,
  key_prefix       VARCHAR(20) NOT NULL,
  scopes           TEXT[] NOT NULL DEFAULT '{}',
  allowed_ips      CIDR[] DEFAULT '{}',
  last_used_at     TIMESTAMP WITH TIME ZONE,
  last_used_ip     INET,
  revoked          BOOLEAN NOT NULL DEFAULT FALSE,
  revoked_at       TIMESTAMP WITH TIME ZONE,
  revoked_reason   VARCHAR(255),
  expires_at       TIMESTAMP WITH TIME ZONE,
  created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id) WHERE revoked = FALSE;
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

-- ============================================================
-- TABLE: login_history
-- Immutable login event log.
-- ============================================================
CREATE TABLE login_history (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  event_type        VARCHAR(50) NOT NULL
                      CHECK (event_type IN (
                        'login_success', 'login_failure', 'logout',
                        'mfa_success', 'mfa_failure', 'token_refresh',
                        'api_key_used', 'sso_login', 'password_reset'
                      )),
  ip_address        INET,
  user_agent        TEXT,
  device_id         UUID,
  session_id        UUID,
  failure_reason    VARCHAR(255),
  metadata          JSONB DEFAULT '{}',
  occurred_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_history_user_id ON login_history(user_id);
CREATE INDEX idx_login_history_occurred_at ON login_history USING BRIN (occurred_at);
CREATE INDEX idx_login_history_event_type ON login_history(event_type, occurred_at DESC);

-- ============================================================
-- TABLE: password_history
-- Stores hashes of previous passwords (prevents reuse).
-- ============================================================
CREATE TABLE password_history (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  password_hash VARCHAR(255) NOT NULL,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_history_user_id ON password_history(user_id);

-- ============================================================
-- TABLE: sso_providers
-- Configured SSO provider integrations per organization.
-- ============================================================
CREATE TABLE sso_providers (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id                UUID NOT NULL,
  provider              VARCHAR(50) NOT NULL
                          CHECK (provider IN ('github', 'google', 'azure', 'okta', 'saml', 'oidc')),
  slug                  VARCHAR(100) NOT NULL,
  display_name          VARCHAR(255),
  client_id             VARCHAR(512),
  encrypted_client_secret TEXT,
  discovery_url         TEXT,
  authorization_url     TEXT,
  token_url             TEXT,
  userinfo_url          TEXT,
  jwks_uri              TEXT,
  scopes                TEXT[] DEFAULT ARRAY['openid', 'email', 'profile'],
  attribute_mapping     JSONB DEFAULT '{}',
  enabled               BOOLEAN NOT NULL DEFAULT TRUE,
  enforce_for_domain    VARCHAR(255),
  created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_sso_providers_org_provider ON sso_providers(org_id, provider)
  WHERE enabled = TRUE;
CREATE INDEX idx_sso_providers_slug ON sso_providers(slug);

-- ============================================================
-- TABLE: sso_identities
-- Links a local user to a remote SSO identity.
-- ============================================================
CREATE TABLE sso_identities (
  id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider_id             UUID NOT NULL REFERENCES sso_providers(id) ON DELETE CASCADE,
  subject                 VARCHAR(512) NOT NULL,
  encrypted_access_token  BYTEA,
  encrypted_refresh_token BYTEA,
  token_key_id            UUID,
  token_expires_at        TIMESTAMP WITH TIME ZONE,
  profile_data            JSONB DEFAULT '{}',
  created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_sso_identities_provider_subject ON sso_identities(provider_id, subject);
CREATE INDEX idx_sso_identities_user_id ON sso_identities(user_id);

-- ============================================================
-- TABLE: jwt_blocklist
-- Blocklisted JTIs (revoked tokens before expiry).
-- ============================================================
CREATE TABLE jwt_blocklist (
  jti         VARCHAR(255) PRIMARY KEY,
  user_id     UUID NOT NULL,
  revoked_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  expires_at  TIMESTAMP WITH TIME ZONE NOT NULL,
  reason      VARCHAR(100)
);

CREATE INDEX idx_jwt_blocklist_expires_at ON jwt_blocklist USING BRIN (expires_at);

-- ============================================================
-- RLS Policies (auth_db — self-scoped)
-- ============================================================
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY users_self_only ON users
  USING (id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY user_sessions_self_only ON user_sessions
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY refresh_tokens_self_only ON refresh_tokens
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE device_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY device_tokens_self_only ON device_tokens
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE mfa_totp_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE mfa_totp_credentials FORCE ROW LEVEL SECURITY;
CREATE POLICY mfa_totp_credentials_self_only ON mfa_totp_credentials
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE mfa_totp_backup_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE mfa_totp_backup_codes FORCE ROW LEVEL SECURITY;
CREATE POLICY mfa_totp_backup_codes_self_only ON mfa_totp_backup_codes
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE mfa_fido2_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE mfa_fido2_credentials FORCE ROW LEVEL SECURITY;
CREATE POLICY mfa_fido2_credentials_self_only ON mfa_fido2_credentials
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE login_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE login_history FORCE ROW LEVEL SECURITY;
CREATE POLICY login_history_self_only ON login_history
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE password_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE password_history FORCE ROW LEVEL SECURITY;
CREATE POLICY password_history_self_only ON password_history
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE sso_identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE sso_identities FORCE ROW LEVEL SECURITY;
CREATE POLICY sso_identities_self_only ON sso_identities
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);

ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY api_keys_self_or_org ON api_keys
  USING (
    (org_id IS NULL AND user_id = current_setting('app.current_user_id', true)::uuid)
    OR (org_id IS NOT NULL AND org_id = current_setting('app.current_org', true)::uuid)
  )
  WITH CHECK (
    (org_id IS NULL AND user_id = current_setting('app.current_user_id', true)::uuid)
    OR (org_id IS NOT NULL AND org_id = current_setting('app.current_org', true)::uuid)
  );

ALTER TABLE sso_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE sso_providers FORCE ROW LEVEL SECURITY;
CREATE POLICY sso_providers_org_isolation ON sso_providers
  USING (org_id = current_setting('app.current_org', true)::uuid)
  WITH CHECK (org_id = current_setting('app.current_org', true)::uuid);

ALTER TABLE jwt_blocklist ENABLE ROW LEVEL SECURITY;
ALTER TABLE jwt_blocklist FORCE ROW LEVEL SECURITY;
CREATE POLICY jwt_blocklist_self_only ON jwt_blocklist
  USING (user_id = current_setting('app.current_user_id', true)::uuid)
  WITH CHECK (user_id = current_setting('app.current_user_id', true)::uuid);
