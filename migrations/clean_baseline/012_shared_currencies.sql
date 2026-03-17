-- =============================================================================
-- SHARED: Валюты
-- =============================================================================

CREATE TABLE currencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    code VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, code)
);

CREATE INDEX idx_currencies_workspace_id ON currencies(workspace_id);
COMMENT ON TABLE currencies IS 'Справочник валют воркспейса';
