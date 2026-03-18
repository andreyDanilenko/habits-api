-- Модули: trial по умолчанию, tasks всегда бесплатен (is_core)
-- default_trial_days: NULL/0 = платный по умолчанию, >0 = триал N дней для новых workspace

ALTER TABLE modules ADD COLUMN IF NOT EXISTS default_trial_days INTEGER DEFAULT 30;
COMMENT ON COLUMN modules.default_trial_days IS 'Триал для новых workspace. NULL/0 = платный. 30 = 30 дней триала.';

-- tasks всегда бесплатен, по умолчанию в каждом workspace
UPDATE modules SET is_core = TRUE WHERE code = 'tasks';

-- Добавляем tasks в workspace, где его ещё нет (на случай workspace, созданных между 000028 и этой миграцией)
INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
SELECT w.id, m.id, 'active', NOW()
FROM workspaces w
CROSS JOIN modules m
WHERE m.code = 'tasks'
AND NOT EXISTS (SELECT 1 FROM workspace_modules wm WHERE wm.workspace_id = w.id AND wm.module_id = m.id)
ON CONFLICT (workspace_id, module_id) DO NOTHING;

-- Не-core модули: 30 дней триала по умолчанию
UPDATE modules SET default_trial_days = 30 WHERE is_core = FALSE AND (default_trial_days IS NULL OR default_trial_days = 0);

-- Триггер: при создании workspace добавляем core (active) и non-core с триалом
DROP TRIGGER IF EXISTS tr_workspace_enable_core_modules ON workspaces;
DROP FUNCTION IF EXISTS fn_workspace_enable_core_modules();

CREATE OR REPLACE FUNCTION fn_workspace_enable_modules()
RETURNS TRIGGER AS $$
BEGIN
    -- Core: всегда active
    INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
    SELECT NEW.id, m.id, 'active', NOW()
    FROM modules m
    WHERE m.is_core = TRUE
    ON CONFLICT (workspace_id, module_id) DO NOTHING;

    -- Non-core с default_trial_days > 0: триал
    INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at, expires_at)
    SELECT NEW.id, m.id, 'trial', NOW(), NOW() + (m.default_trial_days || ' days')::INTERVAL
    FROM modules m
    WHERE m.is_core = FALSE AND m.default_trial_days > 0
    ON CONFLICT (workspace_id, module_id) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_workspace_enable_modules
    AFTER INSERT ON workspaces
    FOR EACH ROW
    EXECUTE FUNCTION fn_workspace_enable_modules();

-- Существующие workspace: добавляем модули с триалом, которых ещё нет
INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at, expires_at)
SELECT w.id, m.id, 'trial', NOW(), NOW() + (m.default_trial_days || ' days')::INTERVAL
FROM workspaces w
CROSS JOIN modules m
WHERE m.is_core = FALSE AND m.default_trial_days > 0
AND NOT EXISTS (SELECT 1 FROM workspace_modules wm WHERE wm.workspace_id = w.id AND wm.module_id = m.id)
ON CONFLICT (workspace_id, module_id) DO NOTHING;
