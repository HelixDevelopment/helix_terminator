CREATE TABLE IF NOT EXISTS ssh_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    host_id UUID NOT NULL,
    host_address TEXT NOT NULL,
    username TEXT NOT NULL,
    auth_type TEXT NOT NULL CHECK (auth_type IN ('password', 'key', 'agent')),
    connection_status TEXT NOT NULL DEFAULT 'connecting' CHECK (connection_status IN ('connecting', 'connected', 'disconnected', 'error')),
    connected_at TIMESTAMPTZ,
    disconnected_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ssh_sessions_user_id ON ssh_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_ssh_sessions_host_id ON ssh_sessions(host_id);
CREATE INDEX IF NOT EXISTS idx_ssh_sessions_status ON ssh_sessions(connection_status);

CREATE TABLE IF NOT EXISTS ssh_channels (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES ssh_sessions(id) ON DELETE CASCADE,
    channel_type TEXT NOT NULL CHECK (channel_type IN ('session', 'direct-tcpip')),
    local_port INTEGER,
    remote_port INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ssh_channels_session_id ON ssh_channels(session_id);
