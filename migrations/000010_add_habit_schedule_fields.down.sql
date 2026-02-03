-- Откат миграции: удаление полей для гибкого расписания

-- Удаляем индексы
DROP INDEX IF EXISTS idx_habits_recurring_days;
DROP INDEX IF EXISTS idx_habits_is_active;
DROP INDEX IF EXISTS idx_habits_one_time_date;
DROP INDEX IF EXISTS idx_habits_schedule_type;

-- Удаляем CHECK ограничения
ALTER TABLE habits DROP CONSTRAINT IF EXISTS check_schedule_fields;
ALTER TABLE habits DROP CONSTRAINT IF EXISTS check_recurring_days;
ALTER TABLE habits DROP CONSTRAINT IF EXISTS check_schedule_type;

-- Удаляем колонки
ALTER TABLE habits
DROP COLUMN IF EXISTS is_active,
DROP COLUMN IF EXISTS one_time_date,
DROP COLUMN IF EXISTS recurring_days,
DROP COLUMN IF EXISTS schedule_type;
