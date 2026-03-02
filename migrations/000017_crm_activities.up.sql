-- CRM Activity Feed (SPEC_BACK_2)
CREATE TABLE IF NOT EXISTS crm_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(20) NOT NULL,
    entity_id UUID NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}',
    is_important BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL,
    created_by_name VARCHAR(255) NOT NULL DEFAULT 'Система',
    created_by_avatar VARCHAR(500),
    is_editable BOOLEAN NOT NULL DEFAULT false,
    is_deletable BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS crm_activity_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id UUID NOT NULL REFERENCES crm_activities(id) ON DELETE CASCADE,
    name VARCHAR(500) NOT NULL,
    size INTEGER NOT NULL,
    type VARCHAR(100) NOT NULL,
    url VARCHAR(1000) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crm_activity_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id UUID NOT NULL REFERENCES crm_activities(id) ON DELETE CASCADE,
    remind_at TIMESTAMPTZ NOT NULL,
    assign_to UUID,
    is_completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crm_activities_workspace ON crm_activities(workspace_id);
CREATE INDEX idx_crm_activities_entity ON crm_activities(entity_type, entity_id);
CREATE INDEX idx_crm_activities_type ON crm_activities(type);
CREATE INDEX idx_crm_activities_created_at ON crm_activities(created_at);
CREATE INDEX idx_crm_activities_important ON crm_activities(is_important) WHERE is_important = true;
CREATE INDEX idx_crm_activities_lookup ON crm_activities(workspace_id, entity_type, entity_id, created_at DESC);
CREATE INDEX idx_crm_activities_deleted ON crm_activities(workspace_id) WHERE deleted_at IS NULL;

CREATE INDEX idx_crm_activity_files_activity ON crm_activity_files(activity_id);
CREATE INDEX idx_crm_activity_reminders_activity ON crm_activity_reminders(activity_id);
