CREATE TABLE IF NOT EXISTS ai_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    org_id UUID,
    prompt TEXT NOT NULL,
    context TEXT,
    model VARCHAR(100) NOT NULL,
    max_tokens INT DEFAULT 2048,
    temperature FLOAT DEFAULT 0.7,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    response TEXT,
    tokens_used INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_requests_user_id ON ai_requests(user_id);
CREATE INDEX idx_ai_requests_org_id ON ai_requests(org_id);
CREATE INDEX idx_ai_requests_status ON ai_requests(status);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ai_requests_updated_at
    BEFORE UPDATE ON ai_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
