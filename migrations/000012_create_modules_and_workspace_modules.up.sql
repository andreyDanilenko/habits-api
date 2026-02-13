-- Справочник всех модулей системы (habits, crm, ...)
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

-- Какие модули включены/оплачены в конкретном workspace
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

-- Сид: модуль habits как core, crm заготовка на будущее
INSERT INTO modules (id, code, name, description, is_core) VALUES
    (gen_random_uuid(), 'habits', 'Привычки', 'Трекер привычек и календарь', TRUE),
    (gen_random_uuid(), 'crm', 'CRM', 'Контакты и сделки', FALSE);

-- Для всех уже существующих workspaces включаем модуль habits
INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
SELECT w.id, (SELECT id FROM modules WHERE code = 'habits' LIMIT 1), 'active', w.created_at
FROM workspaces w
ON CONFLICT (workspace_id, module_id) DO NOTHING;

-- Триггер: при создании нового workspace автоматически включаем модуль habits
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
    EXECUTE FUNCTION fn_workspace_enable_core_modules();
