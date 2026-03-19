-- =============================================================================
-- CRM: Контакты
-- =============================================================================

CREATE TABLE crm_contacts (
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

CREATE TABLE crm_contact_phones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES crm_contacts(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    number VARCHAR(50) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE crm_contact_emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES crm_contacts(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    address VARCHAR(255) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_crm_contacts_workspace ON crm_contacts(workspace_id);
CREATE INDEX idx_crm_contacts_company ON crm_contacts(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_crm_contacts_owner ON crm_contacts(owner_id);
CREATE INDEX idx_crm_contacts_deleted ON crm_contacts(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_contact_phones_contact ON crm_contact_phones(contact_id);
CREATE INDEX idx_crm_contact_emails_contact ON crm_contact_emails(contact_id);
