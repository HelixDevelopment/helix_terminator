CREATE TABLE IF NOT EXISTS keychain_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    org_id UUID,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('ssh', 'gpg', 'api_key', 'password', 'x509')),
    fingerprint VARCHAR(255),
    public_key TEXT,
    private_key TEXT NOT NULL,
    passphrase TEXT,
    metadata JSONB DEFAULT '{}',
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_keychain_items_user_id ON keychain_items(user_id);
CREATE INDEX IF NOT EXISTS idx_keychain_items_org_id ON keychain_items(org_id);
CREATE INDEX IF NOT EXISTS idx_keychain_items_type ON keychain_items(type);
CREATE INDEX IF NOT EXISTS idx_keychain_items_deleted_at ON keychain_items(deleted_at) WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS keychain_items_updated_at ON keychain_items;
CREATE TRIGGER keychain_items_updated_at
    BEFORE UPDATE ON keychain_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
