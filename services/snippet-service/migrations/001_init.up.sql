CREATE TABLE IF NOT EXISTS snippets (
    id UUID PRIMARY KEY,
    org_id UUID,
    created_by UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    tags TEXT[] DEFAULT '{}',
    description TEXT,
    is_public BOOLEAN NOT NULL DEFAULT false,
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_snippets_org_id ON snippets(org_id);
CREATE INDEX IF NOT EXISTS idx_snippets_created_by ON snippets(created_by);
CREATE INDEX IF NOT EXISTS idx_snippets_language ON snippets(language);
CREATE INDEX IF NOT EXISTS idx_snippets_is_public ON snippets(is_public);
CREATE INDEX IF NOT EXISTS idx_snippets_tags ON snippets USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_snippets_updated_at ON snippets(updated_at DESC);
