-- =============================================================================
-- SHARED: Контрагенты
-- =============================================================================

CREATE TABLE counterparties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'client' CHECK (type IN ('client', 'supplier', 'both')),
    email VARCHAR(255),
    phone VARCHAR(50),
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_counterparties_workspace_id ON counterparties(workspace_id);
CREATE INDEX idx_counterparties_type ON counterparties(workspace_id, type);
COMMENT ON TABLE counterparties IS 'Контрагенты (клиенты/поставщики)';
