DROP INDEX IF EXISTS idx_completions_workspace_id;

ALTER TABLE habit_completions
    DROP COLUMN IF EXISTS workspace_id;

DROP INDEX IF EXISTS idx_habit_versions_user_workspace_dates;
DROP INDEX IF EXISTS idx_habit_versions_habit_id;

DROP TABLE IF EXISTS habit_versions;

