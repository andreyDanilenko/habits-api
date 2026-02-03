-- Добавление полей для гибкого расписания привычек

-- Добавляем новые поля в таблицу habits
ALTER TABLE habits
ADD COLUMN schedule_type VARCHAR(20) NOT NULL DEFAULT 'recurring',
ADD COLUMN recurring_days INTEGER[] DEFAULT ARRAY[0,1,2,3,4,5,6],
ADD COLUMN one_time_date DATE,
ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;

-- Добавляем CHECK ограничение для schedule_type
ALTER TABLE habits
ADD CONSTRAINT check_schedule_type 
CHECK (schedule_type IN ('recurring', 'one_time'));

-- Добавляем CHECK ограничение для recurring_days (дни недели 0-6)
ALTER TABLE habits
ADD CONSTRAINT check_recurring_days 
CHECK (
  schedule_type != 'recurring' 
  OR (recurring_days IS NOT NULL AND array_length(recurring_days, 1) > 0)
);

-- Добавляем CHECK ограничение: для recurring нужны recurring_days, для one_time нужна one_time_date
ALTER TABLE habits
ADD CONSTRAINT check_schedule_fields 
CHECK (
  (schedule_type = 'recurring' AND recurring_days IS NOT NULL AND one_time_date IS NULL)
  OR
  (schedule_type = 'one_time' AND one_time_date IS NOT NULL)
);

-- Создаем индексы для оптимизации запросов
CREATE INDEX idx_habits_schedule_type ON habits(schedule_type);
CREATE INDEX idx_habits_one_time_date ON habits(one_time_date) WHERE schedule_type = 'one_time';
CREATE INDEX idx_habits_is_active ON habits(is_active);
CREATE INDEX idx_habits_recurring_days ON habits USING GIN(recurring_days);

-- Комментарии к полям
COMMENT ON COLUMN habits.schedule_type IS 'Тип расписания: recurring (регулярная) или one_time (разовая)';
COMMENT ON COLUMN habits.recurring_days IS 'Массив дней недели для регулярных привычек: 0=воскресенье, 1=понедельник, ..., 6=суббота';
COMMENT ON COLUMN habits.one_time_date IS 'Конкретная дата выполнения для разовых привычек';
COMMENT ON COLUMN habits.is_active IS 'Активна ли привычка (можно временно отключить)';
