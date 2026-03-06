-- Откат миграции 000022_permissions_and_roles

DROP TRIGGER IF EXISTS tr_create_system_roles ON workspaces;
DROP FUNCTION IF EXISTS fn_create_system_roles();

DROP TABLE IF EXISTS role_inheritance;
DROP TABLE IF EXISTS user_permissions;
DROP TABLE IF EXISTS user_role_assignments;
DROP TABLE IF EXISTS workspace_roles;
DROP TABLE IF EXISTS permission_catalog;
