-- Откат миграции 000022_permissions_schema_and_seed

-- Удаляем таблицы в порядке, обратном созданию, с учётом зависимостей

DROP TABLE IF EXISTS role_inheritance;

DROP TABLE IF EXISTS user_permissions;

DROP TABLE IF EXISTS user_role_assignments;

DROP TABLE IF EXISTS workspace_roles;

DROP TABLE IF EXISTS permission_catalog;

