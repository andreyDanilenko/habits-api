-- =============================================================================
-- CRM: Компании
-- =============================================================================

CREATE TABLE crm_companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    inn VARCHAR(12),
    kpp VARCHAR(9),
    ogrn VARCHAR(15),
    phone VARCHAR(50),
    email VARCHAR(255),
    website VARCHAR(255),
    legal_address JSONB,
    actual_address JSONB,
    tags TEXT[] DEFAULT '{}',
    owner_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE crm_company_contacts (
    company_id UUID NOT NULL REFERENCES crm_companies(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES crm_contacts(id) ON DELETE CASCADE,
    position VARCHAR(200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, contact_id)
);

CREATE INDEX idx_crm_companies_workspace ON crm_companies(workspace_id);
CREATE INDEX idx_crm_companies_inn ON crm_companies(inn) WHERE inn IS NOT NULL;
CREATE INDEX idx_crm_companies_owner ON crm_companies(owner_id);
CREATE INDEX idx_crm_companies_deleted ON crm_companies(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_company_contacts_company ON crm_company_contacts(company_id);
CREATE INDEX idx_crm_company_contacts_contact ON crm_company_contacts(contact_id);
