-- =============================================================================
-- MODULES: Справочник модулей, workspace_modules, лицензии
-- Core: habits, crm, projects. Опциональные: notes, inventory, finance, hr, tasks
-- =============================================================================

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

COMMENT ON TABLE modules IS 'Справочник модулей ERP';
COMMENT ON TABLE workspace_modules IS 'Активные модули в workspace';
COMMENT ON COLUMN modules.is_core IS 'По умолчанию включается в новом workspace';
COMMENT ON COLUMN modules.default_trial_days IS 'Триал для новых workspace. NULL/0 = платный. 30 = 30 дней триала.';

-- Seed: core (habits, crm, projects, tasks) и опциональные с триалом
INSERT INTO modules (id, code, name, description, is_core, default_trial_days) VALUES
    (gen_random_uuid(), 'habits', 'Привычки', 'Трекер привычек и календарь', TRUE, NULL),
    (gen_random_uuid(), 'crm', 'CRM', 'Контакты и сделки', TRUE, NULL),
    (gen_random_uuid(), 'projects', 'Проекты', 'Группировка сущностей в проекты', TRUE, NULL),
    (gen_random_uuid(), 'tasks', 'Задачи', 'Управление задачами и напоминаниями', TRUE, NULL),
    (gen_random_uuid(), 'notes', 'Заметки', 'Простые заметки по воркспейсу', FALSE, 30),
    (gen_random_uuid(), 'inventory', 'Склад', 'Учёт остатков (в разработке)', FALSE, 30),
    (gen_random_uuid(), 'finance', 'Финансы', 'Проводки и отчёты (в разработке)', FALSE, 30),
    (gen_random_uuid(), 'hr', 'HR', 'Сотрудники и роли (в разработке)', FALSE, 30);

-- Триггер: core = active, non-core с default_trial_days = trial
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

-- Лицензии пользователей на модули
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
