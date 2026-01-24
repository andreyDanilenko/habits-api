-- Таблица для хранения активности пользователей (для виджета RecentActivity)
CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Тип активности
    -- HABIT_CREATED, HABIT_UPDATED, HABIT_DELETED, HABIT_COMPLETED
    type VARCHAR(50) NOT NULL,
    
    -- Ссылка на сущность (habit, completion и т.д.)
    entity_type VARCHAR(50) NOT NULL,  -- 'habit', 'completion', 'workspace'
    entity_id UUID NOT NULL,
    
    -- Данные для отображения в виджете
    title VARCHAR(255) NOT NULL,  -- 'Завершена привычка "Чтение"'
    emoji VARCHAR(10),             -- '✅', '➕', '✏️', '🗑️'
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX idx_activities_user_id ON activities(user_id);
CREATE INDEX idx_activities_workspace_id ON activities(workspace_id);
CREATE INDEX idx_activities_type ON activities(type);
CREATE INDEX idx_activities_created_at ON activities(created_at);
-- Составной индекс для частого запроса: последние активности пользователя в workspace
CREATE INDEX idx_activities_user_workspace_created ON activities(user_id, workspace_id, created_at DESC);

-- Комментарии для документации
COMMENT ON TABLE activities IS 'Активность пользователей для отображения в виджете RecentActivity';
COMMENT ON COLUMN activities.type IS 'Тип активности: HABIT_CREATED, HABIT_UPDATED, HABIT_DELETED, HABIT_COMPLETED';
COMMENT ON COLUMN activities.entity_type IS 'Тип сущности: habit, completion, workspace';
COMMENT ON COLUMN activities.entity_id IS 'ID сущности, к которой относится активность';
COMMENT ON COLUMN activities.title IS 'Текст для отображения в виджете';
COMMENT ON COLUMN activities.emoji IS 'Эмодзи для отображения в виджете';
