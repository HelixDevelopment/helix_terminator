-- 001_init.sql
-- Create certificate_authorities table
CREATE TABLE IF NOT EXISTS certificate_authorities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    ca_cert_pem TEXT NOT NULL,
    ca_key_pem TEXT NOT NULL,
    serial_number BIGINT NOT NULL DEFAULT 0,
    validity_days INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_certificate_authorities_org_id ON certificate_authorities(org_id);
CREATE INDEX idx_certificate_authorities_deleted_at ON certificate_authorities(deleted_at) WHERE deleted_at IS NULL;

-- Create certificates table
CREATE TABLE IF NOT EXISTS certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ca_id UUID NOT NULL REFERENCES certificate_authorities(id) ON DELETE CASCADE,
    org_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    cert_pem TEXT NOT NULL,
    key_pem TEXT NOT NULL,
    serial_number BIGINT NOT NULL,
    subject JSONB,
    issuer JSONB,
    not_before TIMESTAMPTZ NOT NULL,
    not_after TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revocation_reason TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_certificates_ca_id ON certificates(ca_id);
CREATE INDEX idx_certificates_org_id ON certificates(org_id);
CREATE INDEX idx_certificates_status ON certificates(status);
CREATE INDEX idx_certificates_not_after ON certificates(not_after);

-- Trigger function for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers
CREATE TRIGGER certificate_authorities_updated_at
    BEFORE UPDATE ON certificate_authorities
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER certificates_updated_at
    BEFORE UPDATE ON certificates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
