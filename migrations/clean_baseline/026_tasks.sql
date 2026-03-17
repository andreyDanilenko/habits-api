-- =============================================================================
-- TASKS: Задачи (полная схема, типы: task, bug, feature, meeting, call, email, lunch, other)
-- =============================================================================

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL DEFAULT 'task' CHECK (type IN ('task', 'bug', 'feature', 'meeting', 'call', 'email', 'lunch', 'other')),
    priority VARCHAR(20) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled')),
    due_date TIMESTAMP NOT NULL,
    due_time VARCHAR(10),
    reminder_date TIMESTAMP,
    duration INTEGER,
    completed_at TIMESTAMP,
    completed_by UUID REFERENCES users(id),
    completion_note TEXT,
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    recurring_pattern JSONB,
    parent_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    assignee_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    spent_minutes INTEGER DEFAULT 0
);

CREATE INDEX idx_tasks_workspace ON tasks(workspace_id, deleted_at);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_status ON tasks(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_due_date ON tasks(due_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_reminder ON tasks(reminder_date) WHERE reminder_date IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_tasks_created_by ON tasks(created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_parent ON tasks(parent_id) WHERE parent_id IS NOT NULL AND deleted_at IS NULL;

COMMENT ON TABLE tasks IS 'Задачи workspace. Полиморфная связь с CRM через task_entity_links.';

CREATE TABLE task_entity_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    entity_name VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_links_task ON task_entity_links(task_id);
CREATE INDEX idx_task_links_entity ON task_entity_links(entity_type, entity_id);

COMMENT ON TABLE task_entity_links IS 'Связь задачи с сущностями (crm_deal, crm_contact, crm_company).';

INSERT INTO permission_catalog (module_code, entity_type, action, name, is_system)
VALUES
    ('tasks', 'task', 'create', 'Создание задачи', true),
    ('tasks', 'task', 'read', 'Просмотр задачи', true),
    ('tasks', 'task', 'update', 'Редактирование задачи', true),
    ('tasks', 'task', 'delete', 'Удаление задачи', true)
ON CONFLICT (module_code, entity_type, action) DO NOTHING;
