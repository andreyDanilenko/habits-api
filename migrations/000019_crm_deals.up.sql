-- =============================================================================
-- CRM: Сделки
-- =============================================================================

CREATE TABLE crm_deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(500) NOT NULL,
    contact_id UUID,
    company_id UUID,
    budget DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    pipeline_id UUID NOT NULL REFERENCES crm_pipelines(id),
    stage_id UUID NOT NULL REFERENCES crm_stages(id),
    expected_close_date DATE,
    actual_close_date DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    lost_reason TEXT,
    description TEXT,
    source VARCHAR(100),
    probability INTEGER,
    tags TEXT[] DEFAULT '{}',
    owner_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_crm_deals_workspace ON crm_deals(workspace_id);
CREATE INDEX idx_crm_deals_pipeline ON crm_deals(pipeline_id);
CREATE INDEX idx_crm_deals_stage ON crm_deals(stage_id);
CREATE INDEX idx_crm_deals_owner ON crm_deals(owner_id);
CREATE INDEX idx_crm_deals_status ON crm_deals(status);
CREATE INDEX idx_crm_deals_deleted ON crm_deals(workspace_id) WHERE deleted_at IS NULL;
