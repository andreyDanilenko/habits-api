-- Единая миграция: схема прав, роли workspace, синхронизация с user_workspaces.
-- Идемпотентна: безопасно запускать повторно (ON CONFLICT, NOT EXISTS).

-- =============================================================================
-- 1. ТАБЛИЦЫ
-- =============================================================================

-- Каталог всех возможных прав в системе
CREATE TABLE IF NOT EXISTS permission_catalog (
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
CREATE INDEX IF NOT EXISTS idx_permission_catalog_module ON permission_catalog(module_code);
CREATE INDEX IF NOT EXISTS idx_permission_catalog_entity ON permission_catalog(entity_type);
COMMENT ON TABLE permission_catalog IS 'Каталог всех возможных прав в системе';

-- Роли внутри workspace (системные OWNER/ADMIN/MEMBER/GUEST и кастомные)
CREATE TABLE IF NOT EXISTS workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, name)
);
CREATE INDEX IF NOT EXISTS idx_workspace_roles_workspace ON workspace_roles(workspace_id);
COMMENT ON TABLE workspace_roles IS 'Роли внутри workspace (системные и кастомные)';

-- Назначение ролей пользователям (для Casbin и API)
CREATE TABLE IF NOT EXISTS user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id, workspace_id)
);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user ON user_role_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_role ON user_role_assignments(role_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_workspace ON user_role_assignments(workspace_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_lookup ON user_role_assignments(user_id, workspace_id);
COMMENT ON TABLE user_role_assignments IS 'Назначение ролей пользователям в workspace';

-- Индивидуальные права (минуя роли)
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

-- Наследование ролей
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

-- =============================================================================
-- 2. SEED permission_catalog (включая workspace:module:read)
-- =============================================================================

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
    ('workspace', 'module', 'read', 'Просмотр списка модулей workspace', true)
ON CONFLICT (module_code, entity_type, action) DO NOTHING;

-- =============================================================================
-- 3. Системные роли для каждого workspace
-- =============================================================================

INSERT INTO workspace_roles (id, workspace_id, name, is_system, created_at, updated_at)
SELECT gen_random_uuid(), w.id, r.name, true, NOW(), NOW()
FROM workspaces w
CROSS JOIN (VALUES ('OWNER'), ('ADMIN'), ('MEMBER'), ('GUEST')) AS r(name)
WHERE NOT EXISTS (
    SELECT 1 FROM workspace_roles wr
    WHERE wr.workspace_id = w.id AND wr.name = r.name
);

-- =============================================================================
-- 4. Триггер: системные роли при создании workspace
-- =============================================================================

CREATE OR REPLACE FUNCTION fn_create_system_roles()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO workspace_roles (workspace_id, name, is_system, created_at, updated_at)
    VALUES (NEW.id, 'OWNER', true, NOW(), NOW()),
           (NEW.id, 'ADMIN', true, NOW(), NOW()),
           (NEW.id, 'MEMBER', true, NOW(), NOW()),
           (NEW.id, 'GUEST', true, NOW(), NOW());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tr_create_system_roles ON workspaces;
CREATE TRIGGER tr_create_system_roles
    AFTER INSERT ON workspaces
    FOR EACH ROW
    EXECUTE PROCEDURE fn_create_system_roles();

-- =============================================================================
-- 5. user_workspaces: владелец и нормализация
-- =============================================================================

-- Владелец в user_workspaces (если нет)
INSERT INTO user_workspaces (id, user_id, workspace_id, role, created_at)
SELECT gen_random_uuid(), w.owner_id, w.id, 'OWNER', NOW()
FROM workspaces w
WHERE NOT EXISTS (
    SELECT 1 FROM user_workspaces uw
    WHERE uw.user_id = w.owner_id AND uw.workspace_id = w.id
);

-- Владелец: role = OWNER
UPDATE user_workspaces uw
SET role = 'OWNER'
FROM workspaces w
WHERE w.owner_id = uw.user_id AND w.id = uw.workspace_id AND UPPER(TRIM(uw.role)) != 'OWNER';

-- USER -> MEMBER для не-владельцев
UPDATE user_workspaces uw
SET role = 'MEMBER'
FROM workspaces w
WHERE uw.workspace_id = w.id AND uw.user_id != w.owner_id
  AND UPPER(TRIM(uw.role)) = 'USER';

-- =============================================================================
-- 6. Синхронизация user_workspaces -> user_role_assignments
-- =============================================================================

INSERT INTO user_role_assignments (id, user_id, role_id, workspace_id, assigned_by, assigned_at)
SELECT gen_random_uuid(), uw.user_id, wr.id, uw.workspace_id, w.owner_id, NOW()
FROM user_workspaces uw
JOIN workspaces w ON w.id = uw.workspace_id
JOIN workspace_roles wr ON wr.workspace_id = uw.workspace_id
 AND wr.name = CASE
    WHEN UPPER(TRIM(uw.role)) IN ('OWNER', 'ADMIN', 'MEMBER', 'GUEST') THEN UPPER(TRIM(uw.role))
    ELSE 'MEMBER'
 END
WHERE NOT EXISTS (
    SELECT 1 FROM user_role_assignments ura
    WHERE ura.user_id = uw.user_id AND ura.workspace_id = uw.workspace_id
);

-- Владелец в user_role_assignments с ролью OWNER (если нет или неверная роль)
DELETE FROM user_role_assignments ura
USING workspaces w
JOIN workspace_roles wr_owner ON wr_owner.workspace_id = w.id AND wr_owner.name = 'OWNER'
WHERE ura.user_id = w.owner_id AND ura.workspace_id = w.id AND ura.role_id != wr_owner.id;

INSERT INTO user_role_assignments (id, user_id, role_id, workspace_id, assigned_by, assigned_at)
SELECT gen_random_uuid(), w.owner_id, wr.id, w.id, w.owner_id, NOW()
FROM workspaces w
JOIN workspace_roles wr ON wr.workspace_id = w.id AND wr.name = 'OWNER'
WHERE NOT EXISTS (
    SELECT 1 FROM user_role_assignments ura
    WHERE ura.user_id = w.owner_id AND ura.workspace_id = w.id
);
