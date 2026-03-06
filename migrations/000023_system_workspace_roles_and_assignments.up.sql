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

