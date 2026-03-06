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

