-- CRM — второй модуль по умолчанию (is_core = TRUE). Habits остаётся core (000012).
-- Новые воркспейсы получают оба модуля автоматически (триггер из 000012).
-- Для уже существующих воркспейсов включаем CRM в workspace_modules.

UPDATE modules SET is_core = TRUE WHERE code = 'crm';

INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
SELECT w.id, (SELECT id FROM modules WHERE code = 'crm' LIMIT 1), 'active', COALESCE(w.created_at, NOW())
FROM workspaces w
ON CONFLICT (workspace_id, module_id) DO NOTHING;
