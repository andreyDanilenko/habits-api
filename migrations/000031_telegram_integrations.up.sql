CREATE TABLE IF NOT EXISTS telegram_user_links (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    telegram_chat_id TEXT NOT NULL,
    telegram_user_id TEXT NOT NULL,
    telegram_username TEXT,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS telegram_link_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_link_tokens_user_id
    ON telegram_link_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_telegram_link_tokens_expires_at
    ON telegram_link_tokens(expires_at);
