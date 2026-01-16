-- 1. Workspaces -> Users 
ALTER TABLE workspaces 
ADD CONSTRAINT fk_workspaces_owner 
FOREIGN KEY (owner_id) REFERENCES users(id);

-- 2. Habits -> Users
ALTER TABLE habits 
ADD CONSTRAINT fk_habits_user 
FOREIGN KEY (user_id) REFERENCES users(id);

-- 3. Habits -> Workspaces
ALTER TABLE habits 
ADD CONSTRAINT fk_habits_workspace 
FOREIGN KEY (workspace_id) REFERENCES workspaces(id);

-- 4. Habit Completions -> Habits (удаляем каскадом если привычка удалена)
ALTER TABLE habit_completions 
ADD CONSTRAINT fk_completions_habit 
FOREIGN KEY (habit_id) REFERENCES habits(id) ON DELETE CASCADE;

-- 5. Habit Completions -> Users
ALTER TABLE habit_completions 
ADD CONSTRAINT fk_completions_user 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 6. Unique constraint: одна запись completion на день для привычки
ALTER TABLE habit_completions 
ADD CONSTRAINT unique_habit_date_user 
UNIQUE (habit_id, date, user_id);
