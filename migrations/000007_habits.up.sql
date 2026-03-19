-- =============================================================================
-- HABITS: Привычки (полная схема: базовые поля + schedule)
-- =============================================================================

CREATE TABLE habits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) NOT NULL DEFAULT '#3B82F6',
    icon VARCHAR(100),
    target_days INTEGER DEFAULT 7,
    daily_goal INTEGER DEFAULT 1,
    preferred_time TIME,
    category VARCHAR(100),
    user_id UUID NOT NULL REFERENCES users(id),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    schedule_type VARCHAR(20) NOT NULL DEFAULT 'recurring',
    recurring_days INTEGER[] DEFAULT ARRAY[0,1,2,3,4,5,6],
    one_time_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT check_schedule_type CHECK (schedule_type IN ('recurring', 'one_time')),
    CONSTRAINT check_recurring_days CHECK (
        schedule_type != 'recurring' OR (recurring_days IS NOT NULL AND array_length(recurring_days, 1) > 0)
    ),
    CONSTRAINT check_schedule_fields CHECK (
        (schedule_type = 'recurring' AND recurring_days IS NOT NULL AND one_time_date IS NULL) OR
        (schedule_type = 'one_time' AND one_time_date IS NOT NULL)
    )
);

CREATE INDEX idx_habits_user_id ON habits(user_id);
CREATE INDEX idx_habits_workspace_id ON habits(workspace_id);
CREATE INDEX idx_habits_created_at ON habits(created_at);
CREATE INDEX idx_habits_category ON habits(category);
CREATE INDEX idx_habits_schedule_type ON habits(schedule_type);
CREATE INDEX idx_habits_one_time_date ON habits(one_time_date) WHERE schedule_type = 'one_time';
CREATE INDEX idx_habits_is_active ON habits(is_active);
CREATE INDEX idx_habits_recurring_days ON habits USING GIN(recurring_days);

CREATE TRIGGER update_habits_updated_at
    BEFORE UPDATE ON habits
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
