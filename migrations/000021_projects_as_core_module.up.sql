-- Модуль «Проекты» делаем core, как habits и crm.
-- Новые воркспейсы будут получать его автоматически через триггер (см. 000012_create_modules_and_workspace_modules).
-- Для уже существующих воркспейсов включаем projects в workspace_modules.

UPDATE modules SET is_core = TRUE WHERE code = 'projects';

INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
SELECT w.id, m.id, 'active', COALESCE(w.created_at, NOW())
FROM workspaces w
JOIN modules m ON m.code = 'projects'
ON CONFLICT (workspace_id, module_id) DO NOTHING;

