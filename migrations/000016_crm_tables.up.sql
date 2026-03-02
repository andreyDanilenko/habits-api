-- CRM: contacts
CREATE TABLE IF NOT EXISTS crm_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    company_id UUID,
    position VARCHAR(200),
    birthday DATE,
    tags TEXT[] DEFAULT '{}',
    owner_id UUID NOT NULL,
    created_by UUID NOT NULL,
    updated_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    custom_fields JSONB DEFAULT '{}',
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS crm_contact_phones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES crm_contacts(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    number VARCHAR(50) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS crm_contact_emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES crm_contacts(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    address VARCHAR(255) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false
);

-- CRM: companies (created before crm_contacts so contact.company_id can reference it)
CREATE TABLE IF NOT EXISTS crm_companies (
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

CREATE TABLE IF NOT EXISTS crm_company_contacts (
    company_id UUID NOT NULL REFERENCES crm_companies(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES crm_contacts(id) ON DELETE CASCADE,
    position VARCHAR(200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, contact_id)
);

-- CRM: pipelines and stages
CREATE TABLE IF NOT EXISTS crm_pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crm_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES crm_pipelines(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    order_index INTEGER NOT NULL,
    color VARCHAR(20),
    probability INTEGER NOT NULL,
    is_final BOOLEAN NOT NULL DEFAULT false,
    is_lost BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- CRM: deals
CREATE TABLE IF NOT EXISTS crm_deals (
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

-- Indexes
CREATE INDEX idx_crm_contacts_workspace ON crm_contacts(workspace_id);
CREATE INDEX idx_crm_contacts_company ON crm_contacts(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_crm_contacts_owner ON crm_contacts(owner_id);
CREATE INDEX idx_crm_contacts_deleted ON crm_contacts(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_contact_phones_contact ON crm_contact_phones(contact_id);
CREATE INDEX idx_crm_contact_emails_contact ON crm_contact_emails(contact_id);

CREATE INDEX idx_crm_companies_workspace ON crm_companies(workspace_id);
CREATE INDEX idx_crm_companies_inn ON crm_companies(inn) WHERE inn IS NOT NULL;
CREATE INDEX idx_crm_companies_owner ON crm_companies(owner_id);
CREATE INDEX idx_crm_companies_deleted ON crm_companies(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_company_contacts_company ON crm_company_contacts(company_id);
CREATE INDEX idx_crm_company_contacts_contact ON crm_company_contacts(contact_id);

CREATE INDEX idx_crm_pipelines_workspace ON crm_pipelines(workspace_id);
CREATE INDEX idx_crm_stages_pipeline ON crm_stages(pipeline_id);

CREATE INDEX idx_crm_deals_workspace ON crm_deals(workspace_id);
CREATE INDEX idx_crm_deals_pipeline ON crm_deals(pipeline_id);
CREATE INDEX idx_crm_deals_stage ON crm_deals(stage_id);
CREATE INDEX idx_crm_deals_owner ON crm_deals(owner_id);
CREATE INDEX idx_crm_deals_status ON crm_deals(status);
CREATE INDEX idx_crm_deals_deleted ON crm_deals(workspace_id) WHERE deleted_at IS NULL;
