ALTER TABLE habit_completions DROP CONSTRAINT IF EXISTS unique_habit_date_user;
ALTER TABLE habit_completions DROP CONSTRAINT IF EXISTS fk_completions_user;
ALTER TABLE habit_completions DROP CONSTRAINT IF EXISTS fk_completions_habit;
ALTER TABLE habits DROP CONSTRAINT IF EXISTS fk_habits_workspace;
ALTER TABLE habits DROP CONSTRAINT IF EXISTS fk_habits_user;
ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS fk_workspaces_owner;
