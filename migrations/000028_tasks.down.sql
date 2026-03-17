-- NOTJIRA-23: Откат миграции tasks

DROP TABLE IF EXISTS task_entity_links;
DROP TABLE IF EXISTS tasks;

-- Удаляем права из permission_catalog
DELETE FROM permission_catalog WHERE module_code = 'tasks';

-- Удаляем workspace_modules для tasks
DELETE FROM workspace_modules WHERE module_id IN (SELECT id FROM modules WHERE code = 'tasks');

-- Удаляем модуль
DELETE FROM modules WHERE code = 'tasks';
