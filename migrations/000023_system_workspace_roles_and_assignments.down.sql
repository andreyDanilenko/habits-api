-- Откат миграции 000023_system_workspace_roles_and_assignments

-- 1. Удаляем триггер и функцию создания системных ролей
DROP TRIGGER IF EXISTS tr_create_system_roles ON workspaces;
DROP FUNCTION IF EXISTS fn_create_system_roles();

-- 2. Удаляем назначения ролей (оставляем таблицу, так как она создаётся в 000022)
DELETE FROM user_role_assignments;

-- 3. Удаляем только системные роли (кастомные роли остаются)
DELETE FROM workspace_roles WHERE is_system = true;

