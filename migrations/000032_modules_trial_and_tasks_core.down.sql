-- Откат: tasks снова не core, убираем default_trial_days
DROP TRIGGER IF EXISTS tr_workspace_enable_modules ON workspaces;
DROP FUNCTION IF EXISTS fn_workspace_enable_modules();

CREATE OR REPLACE FUNCTION fn_workspace_enable_core_modules()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
    SELECT NEW.id, m.id, 'active', NOW()
    FROM modules m
    WHERE m.is_core = TRUE
    ON CONFLICT (workspace_id, module_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_workspace_enable_core_modules
    AFTER INSERT ON workspaces
    FOR EACH ROW
    EXECUTE FUNCTION fn_workspace_enable_core_modules();

UPDATE modules SET is_core = FALSE WHERE code = 'tasks';
ALTER TABLE modules DROP COLUMN IF EXISTS default_trial_days;
