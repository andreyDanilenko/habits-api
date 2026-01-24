-- Таблица для хранения истории изменений привычек
CREATE TABLE habit_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Тип действия: CREATED, UPDATED, DELETED, COMPLETED
    action VARCHAR(50) NOT NULL,
    
    -- Что изменилось (JSON для гибкости)
    -- Пример: {"title": {"old": "Старое", "new": "Новое"}, "color": {"old": "#3B82F6", "new": "#10B981"}}
    changes JSONB,
    
    -- Метаданные (IP, user agent, workspace_id и т.д.)
    -- Пример: {"ip": "192.168.1.1", "user_agent": "Mozilla/5.0...", "workspace_id": "..."}
    metadata JSONB,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX idx_habit_history_habit_id ON habit_history(habit_id);
CREATE INDEX idx_habit_history_user_id ON habit_history(user_id);
CREATE INDEX idx_habit_history_action ON habit_history(action);
CREATE INDEX idx_habit_history_created_at ON habit_history(created_at);
-- Составной индекс для частого запроса: история конкретной привычки, отсортированная по дате
CREATE INDEX idx_habit_history_habit_created ON habit_history(habit_id, created_at DESC);

-- Комментарии для документации
COMMENT ON TABLE habit_history IS 'История изменений привычек: создание, обновление, удаление, выполнение';
COMMENT ON COLUMN habit_history.action IS 'Тип действия: CREATED, UPDATED, DELETED, COMPLETED';
COMMENT ON COLUMN habit_history.changes IS 'JSON объект с изменениями полей. Для UPDATED содержит old/new значения';
COMMENT ON COLUMN habit_history.metadata IS 'Дополнительные метаданные: IP, user agent, workspace_id и т.д.';
