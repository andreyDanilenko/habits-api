-- Таблица ожидающих подтверждения регистраций.
-- Пользователь не создаётся в users до подтверждения email по ссылке.
CREATE TABLE registration_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_registration_tokens_token ON registration_tokens(token);
CREATE INDEX idx_registration_tokens_email ON registration_tokens(email);
CREATE INDEX idx_registration_tokens_expires_at ON registration_tokens(expires_at);
