-- =============================================================================
-- ФИНАЛЬНАЯ МИГРАЦИЯ (PRODUCTION-READY)
-- Создаёт все таблицы в актуальном состоянии (сводка миграций 000001–000029).
-- =============================================================================

-- =====================================================
-- 1. Логирование и мониторинг
-- =====================================================
CREATE TABLE request_logs (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms DECIMAL(10, 6) NOT NULL,
    client_ip VARCHAR(45) NOT NULL,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    raw_log TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_request_logs_timestamp ON request_logs(timestamp);

-- =====================================================
-- 2. Пользователи и аутентификация
-- =====================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'USER',
    avatar_url TEXT,
    position VARCHAR(200),
    department_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_department_id ON users(department_id) WHERE department_id IS NOT NULL;

CREATE TABLE registration_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    token VARCHAR(255) NOT NULL UNIQUE,
    invite_token VARCHAR(255),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_registration_tokens_token ON registration_tokens(token);
CREATE INDEX idx_registration_tokens_email ON registration_tokens(email);
CREATE INDEX idx_registration_tokens_expires_at ON registration_tokens(expires_at);

-- =====================================================
-- 3. Workspace (основная единица мультиарендности)
-- =====================================================
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) DEFAULT '#3B82F6',
    logo_path TEXT,
    logo_scale DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    owner_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspaces_owner_id ON workspaces(owner_id);
CREATE INDEX idx_workspaces_created_at ON workspaces(created_at);

CREATE TABLE user_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'MEMBER',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, workspace_id)
);

CREATE INDEX idx_user_workspaces_user_id ON user_workspaces(user_id);
CREATE INDEX idx_user_workspaces_workspace_id ON user_workspaces(workspace_id);

CREATE TABLE user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    current_workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);

-- =====================================================
-- 4. Модули и лицензии
-- =====================================================
CREATE TABLE modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_core BOOLEAN NOT NULL DEFAULT FALSE,
    default_trial_days INTEGER DEFAULT 30,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_modules_code ON modules(code);
CREATE INDEX idx_modules_is_core ON modules(is_core);

CREATE TABLE workspace_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'trial', 'disabled')),
    activated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    settings JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, module_id)
);

CREATE INDEX idx_workspace_modules_workspace_id ON workspace_modules(workspace_id);
CREATE INDEX idx_workspace_modules_module_id ON workspace_modules(module_id);
CREATE INDEX idx_workspace_modules_status ON workspace_modules(status);

CREATE TABLE user_module_licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('all_workspaces', 'single_workspace')),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'cancelled')),
    source VARCHAR(20) NOT NULL DEFAULT 'purchase' CHECK (source IN ('purchase', 'admin_grant')),
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_single_workspace_scope CHECK (
        (scope = 'single_workspace' AND workspace_id IS NOT NULL) OR
        (scope = 'all_workspaces' AND workspace_id IS NULL)
    )
);

CREATE UNIQUE INDEX idx_user_module_licenses_all ON user_module_licenses (user_id, module_id) WHERE scope = 'all_workspaces';
CREATE UNIQUE INDEX idx_user_module_licenses_single ON user_module_licenses (user_id, module_id, workspace_id) WHERE scope = 'single_workspace';
CREATE INDEX idx_user_module_licenses_user_id ON user_module_licenses(user_id);
CREATE INDEX idx_user_module_licenses_module_id ON user_module_licenses(module_id);
CREATE INDEX idx_user_module_licenses_status ON user_module_licenses(status);

-- =====================================================
-- 5. Привычки (Habits Module)
-- =====================================================
CREATE TABLE habits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) NOT NULL DEFAULT '#3B82F6',
    icon VARCHAR(100),
    target_days INTEGER DEFAULT 7,
    daily_goal INTEGER DEFAULT 1,
    preferred_time TIME,
    category VARCHAR(100),
    user_id UUID NOT NULL REFERENCES users(id),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    schedule_type VARCHAR(20) NOT NULL DEFAULT 'recurring' CHECK (schedule_type IN ('recurring', 'one_time')),
    recurring_days INTEGER[] DEFAULT ARRAY[0,1,2,3,4,5,6],
    one_time_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT check_recurring_days CHECK (
        schedule_type != 'recurring' 
        OR (recurring_days IS NOT NULL AND array_length(recurring_days, 1) > 0)
    ),
    CONSTRAINT check_schedule_fields CHECK (
        (schedule_type = 'recurring' AND recurring_days IS NOT NULL AND one_time_date IS NULL)
        OR
        (schedule_type = 'one_time' AND one_time_date IS NOT NULL)
    )
);

CREATE INDEX idx_habits_user_id ON habits(user_id);
CREATE INDEX idx_habits_workspace_id ON habits(workspace_id);
CREATE INDEX idx_habits_schedule_type ON habits(schedule_type);
CREATE INDEX idx_habits_one_time_date ON habits(one_time_date) WHERE schedule_type = 'one_time';
CREATE INDEX idx_habits_is_active ON habits(is_active);
CREATE INDEX idx_habits_recurring_days ON habits USING GIN(recurring_days);
CREATE INDEX idx_habits_created_at ON habits(created_at);
CREATE INDEX idx_habits_category ON habits(category);

CREATE TABLE habit_completions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id),
    date DATE NOT NULL,
    notes TEXT,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    time TIME,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_completions_habit_id ON habit_completions(habit_id);
CREATE INDEX idx_completions_user_id ON habit_completions(user_id);
CREATE INDEX idx_completions_workspace_id ON habit_completions(workspace_id);
CREATE INDEX idx_completions_date ON habit_completions(date);
CREATE INDEX idx_completions_habit_user_date ON habit_completions(habit_id, user_id, date);
CREATE INDEX idx_completions_time ON habit_completions(time);

CREATE TABLE habit_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    changes JSONB,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_habit_history_habit_id ON habit_history(habit_id);
CREATE INDEX idx_habit_history_user_id ON habit_history(user_id);
CREATE INDEX idx_habit_history_action ON habit_history(action);
CREATE INDEX idx_habit_history_created_at ON habit_history(created_at);
CREATE INDEX idx_habit_history_habit_created ON habit_history(habit_id, created_at DESC);

CREATE TABLE habit_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL,
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) NOT NULL,
    icon VARCHAR(100),
    target_days INTEGER NOT NULL,
    daily_goal INTEGER NOT NULL,
    preferred_time TIME,
    category VARCHAR(100),
    schedule_type VARCHAR(20) NOT NULL,
    recurring_days INTEGER[],
    one_time_date DATE,
    is_active BOOLEAN NOT NULL,
    valid_from DATE NOT NULL,
    valid_to DATE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_habit_versions_habit_id ON habit_versions(habit_id);
CREATE INDEX idx_habit_versions_user_workspace_dates ON habit_versions(user_id, workspace_id, valid_from, valid_to);

CREATE TABLE journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description TEXT NOT NULL DEFAULT '',
    mood INTEGER CHECK (mood IS NULL OR (mood >= 1 AND mood <= 5)),
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    tags TEXT[] NOT NULL DEFAULT '{}',
    content_type VARCHAR(20) NOT NULL DEFAULT 'text' CHECK (content_type IN ('text', 'markdown')),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_journal_entries_workspace_id ON journal_entries(workspace_id);
CREATE INDEX idx_journal_entries_user_workspace ON journal_entries(user_id, workspace_id, date DESC);
CREATE INDEX idx_journal_entries_date ON journal_entries(workspace_id, date DESC);

-- =====================================================
-- 6. Заметки (Notes Module)
-- =====================================================
CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notes_workspace_id ON notes(workspace_id);
CREATE INDEX idx_notes_user_workspace ON notes(user_id, workspace_id, created_at DESC);

-- =====================================================
-- 7. CRM Module
-- =====================================================
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
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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

CREATE TABLE crm_pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE crm_stages (
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

CREATE TABLE crm_activities (
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

CREATE TABLE crm_activity_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id UUID NOT NULL REFERENCES crm_activities(id) ON DELETE CASCADE,
    name VARCHAR(500) NOT NULL,
    size INTEGER NOT NULL,
    type VARCHAR(100) NOT NULL,
    url VARCHAR(1000) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE crm_activity_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id UUID NOT NULL REFERENCES crm_activities(id) ON DELETE CASCADE,
    remind_at TIMESTAMPTZ NOT NULL,
    assign_to UUID,
    is_completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы CRM
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
CREATE INDEX idx_crm_activities_workspace ON crm_activities(workspace_id);
CREATE INDEX idx_crm_activities_entity ON crm_activities(entity_type, entity_id);
CREATE INDEX idx_crm_activities_type ON crm_activities(type);
CREATE INDEX idx_crm_activities_created_at ON crm_activities(created_at);
CREATE INDEX idx_crm_activities_important ON crm_activities(is_important) WHERE is_important = true;
CREATE INDEX idx_crm_activities_lookup ON crm_activities(workspace_id, entity_type, entity_id, created_at DESC);
CREATE INDEX idx_crm_activities_deleted ON crm_activities(workspace_id) WHERE deleted_at IS NULL;

CREATE INDEX idx_crm_activity_files_activity ON crm_activity_files(activity_id);
CREATE INDEX idx_crm_activity_reminders_activity ON crm_activity_reminders(activity_id);

-- =====================================================
-- 8. Проекты (Projects Module)
-- =====================================================
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_workspace_id ON projects(workspace_id);

CREATE TABLE project_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, entity_type, entity_id)
);

CREATE INDEX idx_project_entities_project_type ON project_entities(project_id, entity_type);
CREATE INDEX idx_project_entities_entity ON project_entities(entity_type, entity_id);

-- =====================================================
-- 9. Активность (общая лента)
-- =====================================================
CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    emoji VARCHAR(10),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_activities_user_id ON activities(user_id);
CREATE INDEX idx_activities_workspace_id ON activities(workspace_id);
CREATE INDEX idx_activities_type ON activities(type);
CREATE INDEX idx_activities_created_at ON activities(created_at);
CREATE INDEX idx_activities_user_workspace_created ON activities(user_id, workspace_id, created_at DESC);

-- =====================================================
-- 10. Shared (общие справочники)
-- =====================================================
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

-- =====================================================
-- 11. Приглашения
-- =====================================================
CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    invited_by UUID NOT NULL REFERENCES users(id),
    system_role VARCHAR(20) NOT NULL CHECK (system_role IN ('MEMBER', 'GUEST')),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'ACCEPTED', 'EXPIRED', 'CANCELLED')),
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMP
);

CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_email ON invitations(email);
CREATE INDEX idx_invitations_status ON invitations(status);
CREATE INDEX idx_invitations_workspace ON invitations(workspace_id);
CREATE INDEX idx_invitations_expires ON invitations(expires_at) WHERE status = 'PENDING';

-- =====================================================
-- 12. Уведомления
-- =====================================================
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    channel VARCHAR(32) NOT NULL DEFAULT 'activity',
    event_type VARCHAR(64) NOT NULL,
    event_key VARCHAR(255) NOT NULL,
    title TEXT NOT NULL,
    subtitle TEXT,
    payload JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP
);

CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id) WHERE read_at IS NULL;
CREATE UNIQUE INDEX idx_notifications_user_event_key ON notifications(user_id, event_key);

-- =====================================================
-- 13. Задачи (Tasks Module)
-- =====================================================
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

CREATE TABLE task_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES task_comments(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_comments_task ON task_comments(task_id);
CREATE INDEX idx_task_comments_parent ON task_comments(parent_id);
CREATE INDEX idx_task_comments_created_at ON task_comments(task_id, created_at DESC);

-- =====================================================
-- 14. Права доступа и роли
-- =====================================================
CREATE TABLE permission_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_code VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(module_code, entity_type, action)
);

CREATE INDEX idx_permission_catalog_module ON permission_catalog(module_code);
CREATE INDEX idx_permission_catalog_entity ON permission_catalog(entity_type);

CREATE TABLE workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, name)
);

CREATE INDEX idx_workspace_roles_workspace ON workspace_roles(workspace_id);

-- Data scope по роли и объекту (ABAC-слой поверх Casbin; миграция 000029).
-- object_key как у Casbin: crm:deal, crm:contact, ...
CREATE TABLE role_object_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    object_key VARCHAR(128) NOT NULL,
    data_scope VARCHAR(32) NOT NULL CHECK (data_scope IN ('all', 'owner', 'department', 'none')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, object_key)
);

CREATE INDEX idx_role_object_scopes_role ON role_object_scopes(role_id);

CREATE TABLE user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id, workspace_id)
);

CREATE INDEX idx_user_role_assignments_user ON user_role_assignments(user_id);
CREATE INDEX idx_user_role_assignments_role ON user_role_assignments(role_id);
CREATE INDEX idx_user_role_assignments_workspace ON user_role_assignments(workspace_id);
CREATE INDEX idx_user_role_assignments_lookup ON user_role_assignments(user_id, workspace_id);

CREATE TABLE user_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permission_catalog(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    UNIQUE(user_id, workspace_id, permission_id)
);

CREATE INDEX idx_user_permissions_user ON user_permissions(user_id);
CREATE INDEX idx_user_permissions_workspace ON user_permissions(workspace_id);
CREATE INDEX idx_user_permissions_lookup ON user_permissions(user_id, workspace_id);

CREATE TABLE role_inheritance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    child_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    parent_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT check_no_self_inheritance CHECK (child_role_id != parent_role_id),
    UNIQUE(workspace_id, child_role_id, parent_role_id)
);

CREATE INDEX idx_role_inheritance_child ON role_inheritance(child_role_id);
CREATE INDEX idx_role_inheritance_parent ON role_inheritance(parent_role_id);

-- =====================================================
-- 15. Триггеры и функции
-- =====================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_workspaces_updated_at 
    BEFORE UPDATE ON workspaces 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_habits_updated_at 
    BEFORE UPDATE ON habits 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION fn_workspace_enable_modules()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
    SELECT NEW.id, m.id, 'active', NOW()
    FROM modules m
    WHERE m.is_core = TRUE
    ON CONFLICT (workspace_id, module_id) DO NOTHING;

    INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at, expires_at)
    SELECT NEW.id, m.id, 'trial', NOW(), NOW() + (m.default_trial_days || ' days')::INTERVAL
    FROM modules m
    WHERE m.is_core = FALSE AND m.default_trial_days > 0
    ON CONFLICT (workspace_id, module_id) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_workspace_enable_modules
    AFTER INSERT ON workspaces
    FOR EACH ROW
    EXECUTE PROCEDURE fn_workspace_enable_modules();

CREATE OR REPLACE FUNCTION fn_create_system_roles()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO workspace_roles (workspace_id, name, is_system, created_at, updated_at)
    VALUES 
        (NEW.id, 'OWNER', true, NOW(), NOW()),
        (NEW.id, 'ADMIN', true, NOW(), NOW()),
        (NEW.id, 'MEMBER', true, NOW(), NOW()),
        (NEW.id, 'GUEST', true, NOW(), NOW());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_create_system_roles
    AFTER INSERT ON workspaces
    FOR EACH ROW
    EXECUTE PROCEDURE fn_create_system_roles();

-- =====================================================
-- 16. Инициализация модулей
-- =====================================================
INSERT INTO modules (id, code, name, description, is_core, default_trial_days) VALUES
    (gen_random_uuid(), 'habits', 'Привычки', 'Трекер привычек и календарь', TRUE, NULL),
    (gen_random_uuid(), 'crm', 'CRM', 'Контакты и сделки', TRUE, NULL),
    (gen_random_uuid(), 'projects', 'Проекты', 'Группировка сущностей в проекты', TRUE, NULL),
    (gen_random_uuid(), 'tasks', 'Задачи', 'Управление задачами и напоминаниями', TRUE, NULL),
    (gen_random_uuid(), 'notes', 'Заметки', 'Простые заметки по воркспейсу', FALSE, 30),
    (gen_random_uuid(), 'inventory', 'Склад', 'Учёт остатков и номенклатуры (в разработке)', FALSE, 30),
    (gen_random_uuid(), 'finance', 'Финансы', 'Проводки и отчёты (в разработке)', FALSE, 30),
    (gen_random_uuid(), 'hr', 'HR', 'Сотрудники и роли (в разработке)', FALSE, 30)
ON CONFLICT (code) DO NOTHING;

-- =====================================================
-- 17. Инициализация прав
-- =====================================================
INSERT INTO permission_catalog (module_code, entity_type, action, name, is_system) VALUES
    ('crm', 'deal', 'create', 'Создание сделки', true),
    ('crm', 'deal', 'read', 'Просмотр сделки', true),
    ('crm', 'deal', 'update', 'Редактирование сделки', true),
    ('crm', 'deal', 'delete', 'Удаление сделки', true),
    ('crm', 'deal', 'move', 'Перемещение сделки по этапам', true),
    ('crm', 'contact', 'create', 'Создание контакта', true),
    ('crm', 'contact', 'read', 'Просмотр контакта', true),
    ('crm', 'contact', 'update', 'Редактирование контакта', true),
    ('crm', 'contact', 'delete', 'Удаление контакта', true),
    ('crm', 'company', 'create', 'Создание компании', true),
    ('crm', 'company', 'read', 'Просмотр компании', true),
    ('crm', 'company', 'update', 'Редактирование компании', true),
    ('crm', 'company', 'delete', 'Удаление компании', true),
    ('crm', 'pipeline', 'manage', 'Управление воронками продаж', true),
    ('crm', 'activity', 'create', 'Создание CRM-активности', true),
    ('crm', 'activity', 'read', 'Просмотр CRM-активности', true),
    ('crm', 'activity', 'update', 'Редактирование CRM-активности', true),
    ('crm', 'activity', 'delete', 'Удаление CRM-активности', true),
    ('crm', 'export', 'deals', 'Экспорт сделок', true),
    ('habits', 'habit', 'create', 'Создание привычки', true),
    ('habits', 'habit', 'read', 'Просмотр привычки', true),
    ('habits', 'habit', 'update', 'Редактирование привычки', true),
    ('habits', 'habit', 'delete', 'Удаление привычки', true),
    ('habits', 'habit', 'complete', 'Отметка выполнения привычки', true),
    ('habits', 'journal', 'create', 'Создание записи в журнале', true),
    ('habits', 'journal', 'read', 'Просмотр записи в журнале', true),
    ('habits', 'journal', 'update', 'Редактирование записи в журнале', true),
    ('habits', 'journal', 'delete', 'Удаление записи в журнале', true),
    ('projects', 'project', 'create', 'Создание проекта', true),
    ('projects', 'project', 'read', 'Просмотр проекта', true),
    ('projects', 'project', 'update', 'Редактирование проекта', true),
    ('projects', 'project', 'delete', 'Удаление проекта', true),
    ('projects', 'entity', 'attach', 'Привязка сущности к проекту', true),
    ('projects', 'entity', 'detach', 'Отвязка сущности от проекта', true),
    ('workspace', 'member', 'invite', 'Приглашение участников в workspace', true),
    ('workspace', 'member', 'remove', 'Удаление участников из workspace', true),
    ('workspace', 'role', 'manage', 'Управление ролями workspace', true),
    ('workspace', 'module', 'manage', 'Управление модулями workspace', true),
    ('workspace', 'module', 'read', 'Просмотр списка модулей workspace', true),
    ('tasks', 'task', 'create', 'Создание задачи', true),
    ('tasks', 'task', 'read', 'Просмотр задачи', true),
    ('tasks', 'task', 'update', 'Редактирование задачи', true),
    ('tasks', 'task', 'delete', 'Удаление задачи', true)
ON CONFLICT (module_code, entity_type, action) DO NOTHING;

-- =====================================================
-- 18. Интеграции (Telegram + универсальные провайдеры)
-- =====================================================
CREATE TABLE user_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_user_integrations_provider ON user_integrations(provider);
CREATE INDEX idx_user_integrations_external_id ON user_integrations(external_id);
