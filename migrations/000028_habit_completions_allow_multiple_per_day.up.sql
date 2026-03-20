-- Несколько выполнений одной привычки за день (daily_goal > 1).
-- Ранее UNIQUE(habit_id, date, user_id) + upsert оставляли одну строку на день.
ALTER TABLE habit_completions
    DROP CONSTRAINT IF EXISTS habit_completions_habit_id_date_user_id_key;
