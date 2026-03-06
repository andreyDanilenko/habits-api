-- =============================================================================
-- Сводный файл всех миграций (только UP, без DROP)
-- Порядок: 000001 .. 000021, затем constraints/01, constraints/02
-- =============================================================================

-- ========== 000001_create_request_logs ==========
CREATE TABLE IF NOT EXISTS request_logs (
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

CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);

-- ========== 000002_create_users ==========
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'USER',
    avatar_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- ========== 000003_create_habits ==========
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
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_habits_user_id ON habits(user_id);
CREATE INDEX idx_habits_workspace_id ON habits(workspace_id);
CREATE INDEX idx_habits_created_at ON habits(created_at);
CREATE INDEX idx_habits_category ON habits(category);

-- ========== 000004_create_habit_completions ==========
CREATE TABLE habit_completions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL,
    user_id UUID NOT NULL,
    date DATE NOT NULL,
    notes TEXT,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    time TIME,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_completions_habit_id ON habit_completions(habit_id);
CREATE INDEX idx_completions_user_id ON habit_completions(user_id);
CREATE INDEX idx_completions_date ON habit_completions(date);
CREATE INDEX idx_completions_habit_user_date ON habit_completions(habit_id, user_id, date);
CREATE INDEX idx_completions_time ON habit_completions(time);

-- ========== 000005_create_workspaces ==========
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) DEFAULT '#3B82F6',
    owner_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspaces_owner_id ON workspaces(owner_id);
CREATE INDEX idx_workspaces_created_at ON workspaces(created_at);

-- ========== 000006_create_user_workspaces ==========
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

-- ========== 000007_create_user_preferences ==========
CREATE TABLE user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    current_workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);

-- ========== 000008_create_habit_history ==========
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

COMMENT ON TABLE habit_history IS 'История изменений привычек: создание, обновление, удаление, выполнение';
COMMENT ON COLUMN habit_history.action IS 'Тип действия: CREATED, UPDATED, DELETED, COMPLETED';
COMMENT ON COLUMN habit_history.changes IS 'JSON объект с изменениями полей. Для UPDATED содержит old/new значения';
COMMENT ON COLUMN habit_history.metadata IS 'Дополнительные метаданные: IP, user agent, workspace_id и т.д.';

-- ========== 000009_create_activities ==========
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

COMMENT ON TABLE activities IS 'Активность пользователей для отображения в виджете RecentActivity';
COMMENT ON COLUMN activities.type IS 'Тип активности: HABIT_CREATED, HABIT_UPDATED, HABIT_DELETED, HABIT_COMPLETED';
COMMENT ON COLUMN activities.entity_type IS 'Тип сущности: habit, completion, workspace';
COMMENT ON COLUMN activities.entity_id IS 'ID сущности, к которой относится активность';
COMMENT ON COLUMN activities.title IS 'Текст для отображения в виджете';
COMMENT ON COLUMN activities.emoji IS 'Эмодзи для отображения в виджете';

-- ========== 000010_add_habit_schedule_fields ==========
ALTER TABLE habits
ADD COLUMN schedule_type VARCHAR(20) NOT NULL DEFAULT 'recurring',
ADD COLUMN recurring_days INTEGER[] DEFAULT ARRAY[0,1,2,3,4,5,6],
ADD COLUMN one_time_date DATE,
ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE habits
ADD CONSTRAINT check_schedule_type 
CHECK (schedule_type IN ('recurring', 'one_time'));

ALTER TABLE habits
ADD CONSTRAINT check_recurring_days 
CHECK (
  schedule_type != 'recurring' 
  OR (recurring_days IS NOT NULL AND array_length(recurring_days, 1) > 0)
);

ALTER TABLE habits
ADD CONSTRAINT check_schedule_fields 
CHECK (
  (schedule_type = 'recurring' AND recurring_days IS NOT NULL AND one_time_date IS NULL)
  OR
  (schedule_type = 'one_time' AND one_time_date IS NOT NULL)
);

CREATE INDEX idx_habits_schedule_type ON habits(schedule_type);
CREATE INDEX idx_habits_one_time_date ON habits(one_time_date) WHERE schedule_type = 'one_time';
CREATE INDEX idx_habits_is_active ON habits(is_active);
CREATE INDEX idx_habits_recurring_days ON habits USING GIN(recurring_days);

COMMENT ON COLUMN habits.schedule_type IS 'Тип расписания: recurring (регулярная) или one_time (разовая)';
COMMENT ON COLUMN habits.recurring_days IS 'Массив дней недели для регулярных привычек: 0=воскресенье, 1=понедельник, ..., 6=суббота';
COMMENT ON COLUMN habits.one_time_date IS 'Конкретная дата выполнения для разовых привычек';
COMMENT ON COLUMN habits.is_active IS 'Активна ли привычка (можно временно отключить)';

-- ========== 000011_create_habit_versions ==========
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
CREATE INDEX idx_habit_versions_user_workspace_dates
    ON habit_versions(user_id, workspace_id, valid_from, valid_to);

ALTER TABLE habit_completions
    ADD COLUMN IF NOT EXISTS workspace_id UUID;

UPDATE habit_completions hc
SET workspace_id = h.workspace_id
FROM habits h
WHERE hc.habit_id = h.id
  AND hc.workspace_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_completions_workspace_id
    ON habit_completions(workspace_id);

-- ========== 000012_create_modules_and_workspace_modules ==========
CREATE TABLE modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_core BOOLEAN NOT NULL DEFAULT FALSE,
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

COMMENT ON TABLE modules IS 'Справочник модулей ERP: habits, crm и т.д.';
COMMENT ON TABLE workspace_modules IS 'Какие модули активны в workspace (оплачены/включены). Доступ к сущностям модуля проверять по workspace_id + активная запись здесь.';
COMMENT ON COLUMN modules.is_core IS 'Базовый модуль (например habits) — по умолчанию доступен в новом workspace';

INSERT INTO modules (id, code, name, description, is_core) VALUES
    (gen_random_uuid(), 'habits', 'Привычки', 'Трекер привычек и календарь', TRUE),
    (gen_random_uuid(), 'crm', 'CRM', 'Контакты и сделки', FALSE);

INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
SELECT w.id, (SELECT id FROM modules WHERE code = 'habits' LIMIT 1), 'active', w.created_at
FROM workspaces w
ON CONFLICT (workspace_id, module_id) DO NOTHING;

CREATE OR REPLACE FUNCTION fn_workspace_enable_core_modules()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
    SELECT NEW.id, m.id, 'active', NOW()
    FROM modules m
    WHERE m.is_core = TRUE
    ON CONFLICT (workspace_id, module_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_workspace_enable_core_modules
    AFTER INSERT ON workspaces
    FOR EACH ROW
    EXECUTE PROCEDURE fn_workspace_enable_core_modules();

-- ========== 000013_create_user_module_licenses ==========
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

CREATE UNIQUE INDEX idx_user_module_licenses_all
    ON user_module_licenses (user_id, module_id)
    WHERE scope = 'all_workspaces';

CREATE UNIQUE INDEX idx_user_module_licenses_single
    ON user_module_licenses (user_id, module_id, workspace_id)
    WHERE scope = 'single_workspace';

CREATE INDEX idx_user_module_licenses_user_id ON user_module_licenses(user_id);
CREATE INDEX idx_user_module_licenses_module_id ON user_module_licenses(module_id);
CREATE INDEX idx_user_module_licenses_status ON user_module_licenses(status);

COMMENT ON TABLE user_module_licenses IS 'Лицензия пользователя на модуль: для всех воркспейсов или для одного. Проверяется при включении модуля во воркспейсе.';
COMMENT ON COLUMN user_module_licenses.scope IS 'all_workspaces = можно включать в любом своём воркспейсе; single_workspace = только в указанном workspace_id';
COMMENT ON COLUMN user_module_licenses.source IS 'purchase = оплата; admin_grant = выдача админом';

-- ========== 000014_shared_schema_and_notes ==========
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
COMMENT ON TABLE currencies IS 'Shared: справочник валют воркспейса. Ссылаются модули Финансы, CRM, Склад.';

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
COMMENT ON TABLE counterparties IS 'Shared: контрагенты (клиенты/поставщики). Один ID в CRM, Финансах, Закупках.';

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
COMMENT ON TABLE notes IS 'Модуль Заметки: простые заметки в рамках воркспейса.';

INSERT INTO modules (id, code, name, description, is_core) VALUES
    (gen_random_uuid(), 'notes', 'Заметки', 'Простые заметки по воркспейсу', FALSE),
    (gen_random_uuid(), 'inventory', 'Склад', 'Учёт остатков и номенклатуры (в разработке)', FALSE),
    (gen_random_uuid(), 'finance', 'Финансы', 'Проводки и отчёты (в разработке)', FALSE),
    (gen_random_uuid(), 'hr', 'HR', 'Сотрудники и роли (в разработке)', FALSE);

-- ========== 000015_create_journal_entries ==========
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

COMMENT ON TABLE journal_entries IS 'Записи дневника (модуль habits). Доступ по workspace_id.';

-- ========== 000016_crm_tables ==========
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

-- ========== 000017_crm_activities ==========
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

-- ========== 000018_projects_and_project_entities ==========
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

-- ========== constraints/01_foreign_keys ==========
-- 1. Workspaces -> Users 
ALTER TABLE workspaces 
ADD CONSTRAINT fk_workspaces_owner 
FOREIGN KEY (owner_id) REFERENCES users(id);

-- 2. Habits -> Users
ALTER TABLE habits 
ADD CONSTRAINT fk_habits_user 
FOREIGN KEY (user_id) REFERENCES users(id);

-- 3. Habits -> Workspaces
ALTER TABLE habits 
ADD CONSTRAINT fk_habits_workspace 
FOREIGN KEY (workspace_id) REFERENCES workspaces(id);

-- 4. Habit Completions -> Habits (удаляем каскадом если привычка удалена)
ALTER TABLE habit_completions 
ADD CONSTRAINT fk_completions_habit 
FOREIGN KEY (habit_id) REFERENCES habits(id) ON DELETE CASCADE;

-- 5. Habit Completions -> Users
ALTER TABLE habit_completions 
ADD CONSTRAINT fk_completions_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 6. Unique constraint: одна запись completion на день для привычки
ALTER TABLE habit_completions 
ADD CONSTRAINT unique_habit_date_user 
UNIQUE (habit_id, date, user_id);

-- ========== constraints/02_triggers ==========
-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггер для таблицы workspaces
CREATE TRIGGER update_workspaces_updated_at 
    BEFORE UPDATE ON workspaces 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Триггер для таблицы habits
CREATE TRIGGER update_habits_updated_at 
    BEFORE UPDATE ON habits 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- ========== 000019_crm_as_core_module ==========
-- CRM — второй модуль по умолчанию (is_core = TRUE). Habits остаётся core (000012).
-- Новые воркспейсы получают оба модуля автоматически (триггер из 000012).
-- Для уже существующих воркспейсов включаем CRM в workspace_modules.

UPDATE modules SET is_core = TRUE WHERE code = 'crm';

INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
SELECT w.id, (SELECT id FROM modules WHERE code = 'crm' LIMIT 1), 'active', COALESCE(w.created_at, NOW())
FROM workspaces w
ON CONFLICT (workspace_id, module_id) DO NOTHING;

-- ========== 000020_add_projects_module ==========
-- Модуль «Проекты»: группировка сущностей из CRM и других модулей в контексты (см. docs/PROJECTS_SPEC.md).
INSERT INTO modules (id, code, name, description, is_core) VALUES
    (gen_random_uuid(), 'projects', 'Проекты', 'Группировка контактов, компаний, сделок и других сущностей в проекты', FALSE);

-- ========== 000021_projects_as_core_module ==========
-- Модуль «Проекты» делаем core, как habits и crm.
-- Новые воркспейсы будут получать его автоматически через триггер (см. 000012_create_modules_and_workspace_modules).
-- Для уже существующих воркспейсов включаем projects в workspace_modules.

UPDATE modules SET is_core = TRUE WHERE code = 'projects';

INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
SELECT w.id, m.id, 'active', COALESCE(w.created_at, NOW())
FROM workspaces w
JOIN modules m ON m.code = 'projects'
ON CONFLICT (workspace_id, module_id) DO NOTHING;

-- ========== 000022_permissions_schema_and_seed ==========
-- Создание таблиц для гибкой системы ролей и прав доступа

-- 1. Каталог всех возможных прав в системе
CREATE TABLE IF NOT EXISTS permission_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_code VARCHAR(50) NOT NULL,      -- crm, habits, projects, workspace
    entity_type VARCHAR(50) NOT NULL,      -- deal, contact, company, habit, journal, member, role, module, ...
    action VARCHAR(50) NOT NULL,           -- create, read, update, delete, manage, export, move, complete, attach, detach
    name VARCHAR(255) NOT NULL,            -- "Создание сделки" (для UI)
    description TEXT,
    is_system BOOLEAN DEFAULT false,       -- системные права нельзя удалить
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(module_code, entity_type, action)
);

CREATE INDEX IF NOT EXISTS idx_permission_catalog_module ON permission_catalog(module_code);
CREATE INDEX IF NOT EXISTS idx_permission_catalog_entity ON permission_catalog(entity_type);

COMMENT ON TABLE permission_catalog IS 'Каталог всех возможных прав в системе';
COMMENT ON COLUMN permission_catalog.module_code IS 'Код модуля (crm, habits, projects, workspace)';
COMMENT ON COLUMN permission_catalog.entity_type IS 'Тип сущности (deal, contact, habit, journal, member, role, module)';
COMMENT ON COLUMN permission_catalog.action IS 'Действие (create, read, update, delete, manage, export, ...)';

-- 2. Роли внутри workspace (системные и кастомные)
CREATE TABLE IF NOT EXISTS workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,              -- "Sales Manager"
    description TEXT,
    is_system BOOLEAN DEFAULT false,         -- true для OWNER, ADMIN, MEMBER, GUEST
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_workspace_roles_workspace ON workspace_roles(workspace_id);

COMMENT ON TABLE workspace_roles IS 'Роли внутри workspace (системные и кастомные)';
COMMENT ON COLUMN workspace_roles.is_system IS 'Системные роли (OWNER, ADMIN, MEMBER, GUEST) нельзя удалить';

-- 3. Назначение ролей пользователям в workspace
CREATE TABLE IF NOT EXISTS user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),   -- кто назначил
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, role_id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user ON user_role_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_role ON user_role_assignments(role_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_workspace ON user_role_assignments(workspace_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_lookup ON user_role_assignments(user_id, workspace_id);

COMMENT ON TABLE user_role_assignments IS 'Назначение ролей пользователям в workspace';
COMMENT ON COLUMN user_role_assignments.assigned_by IS 'ID пользователя, назначившего роль';

-- 4. Индивидуальные права пользователя (минуя роли)
CREATE TABLE IF NOT EXISTS user_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permission_catalog(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,

    UNIQUE(user_id, workspace_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_user_permissions_user ON user_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_permissions_workspace ON user_permissions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_user_permissions_lookup ON user_permissions(user_id, workspace_id);

COMMENT ON TABLE user_permissions IS 'Индивидуальные права для конкретных пользователей';

-- 5. Наследование ролей (иерархия)
CREATE TABLE IF NOT EXISTS role_inheritance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    child_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    parent_role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT check_no_self_inheritance CHECK (child_role_id != parent_role_id),
    UNIQUE(workspace_id, child_role_id, parent_role_id)
);

CREATE INDEX IF NOT EXISTS idx_role_inheritance_child ON role_inheritance(child_role_id);
CREATE INDEX IF NOT EXISTS idx_role_inheritance_parent ON role_inheritance(parent_role_id);

COMMENT ON TABLE role_inheritance IS 'Иерархия наследования ролей';

-- 6. Наполнение permission_catalog базовым набором прав

-- CRM права
INSERT INTO permission_catalog (module_code, entity_type, action, name, is_system)
VALUES
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

    ('crm', 'export', 'deals', 'Экспорт сделок', true)
ON CONFLICT (module_code, entity_type, action) DO NOTHING;

-- Habits права
INSERT INTO permission_catalog (module_code, entity_type, action, name, is_system)
VALUES
    ('habits', 'habit', 'create', 'Создание привычки', true),
    ('habits', 'habit', 'read', 'Просмотр привычки', true),
    ('habits', 'habit', 'update', 'Редактирование привычки', true),
    ('habits', 'habit', 'delete', 'Удаление привычки', true),
    ('habits', 'habit', 'complete', 'Отметка выполнения привычки', true),

    ('habits', 'journal', 'create', 'Создание записи в журнале', true),
    ('habits', 'journal', 'read', 'Просмотр записи в журнале', true),
    ('habits', 'journal', 'update', 'Редактирование записи в журнале', true),
    ('habits', 'journal', 'delete', 'Удаление записи в журнале', true)
ON CONFLICT (module_code, entity_type, action) DO NOTHING;

-- Projects права
INSERT INTO permission_catalog (module_code, entity_type, action, name, is_system)
VALUES
    ('projects', 'project', 'create', 'Создание проекта', true),
    ('projects', 'project', 'read', 'Просмотр проекта', true),
    ('projects', 'project', 'update', 'Редактирование проекта', true),
    ('projects', 'project', 'delete', 'Удаление проекта', true),

    ('projects', 'entity', 'attach', 'Привязка сущности к проекту', true),
    ('projects', 'entity', 'detach', 'Отвязка сущности от проекта', true)
ON CONFLICT (module_code, entity_type, action) DO NOTHING;

-- Workspace управление
INSERT INTO permission_catalog (module_code, entity_type, action, name, is_system)
VALUES
    ('workspace', 'member', 'invite', 'Приглашение участников в workspace', true),
    ('workspace', 'member', 'remove', 'Удаление участников из workspace', true),
    ('workspace', 'role', 'manage', 'Управление ролями workspace', true),
    ('workspace', 'module', 'manage', 'Управление модулями workspace', true)
ON CONFLICT (module_code, entity_type, action) DO NOTHING;

-- ========== 000023_system_workspace_roles_and_assignments ==========
-- Создание системных ролей по всем существующим workspace
-- и перенос текущих назначений из user_workspaces в user_role_assignments

-- 1. Системные роли для каждого workspace
INSERT INTO workspace_roles (id, workspace_id, name, is_system, created_at, updated_at)
SELECT gen_random_uuid(), id, 'OWNER', true, NOW(), NOW() FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'ADMIN', true, NOW(), NOW() FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'MEMBER', true, NOW(), NOW() FROM workspaces
UNION ALL
SELECT gen_random_uuid(), id, 'GUEST', true, NOW(), NOW() FROM workspaces;

-- 2. Перенос существующих назначений ролей из user_workspaces
INSERT INTO user_role_assignments (id, user_id, role_id, workspace_id, assigned_at)
SELECT 
    gen_random_uuid(),
    uw.user_id,
    wr.id,
    uw.workspace_id,
    uw.created_at
FROM user_workspaces uw
JOIN workspace_roles wr 
  ON wr.workspace_id = uw.workspace_id 
 AND wr.name = uw.role;

-- 3. Триггер: автоматическое создание системных ролей при создании нового workspace
CREATE OR REPLACE FUNCTION fn_create_system_roles()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO workspace_roles (workspace_id, name, is_system, created_at, updated_at) VALUES
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
    EXECUTE FUNCTION fn_create_system_roles();
