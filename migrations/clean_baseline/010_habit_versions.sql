-- =============================================================================
-- HABITS: Версии привычек
-- =============================================================================

CREATE TABLE habit_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL,
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) NOT NULL,
    icon VARCHAR(100),
    target_days INTEGER NOT NULL,
    daily_goal INTEGER NOT NULL,
    preferred_time TIME,
    category VARCHAR(100),
    schedule_type VARCHAR(20) NOT NULL,
    recurring_days INTEGER[],
    one_time_date DATE,
    is_active BOOLEAN NOT NULL,
    valid_from DATE NOT NULL,
    valid_to DATE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_habit_versions_habit_id ON habit_versions(habit_id);
CREATE INDEX idx_habit_versions_user_workspace_dates
    ON habit_versions(user_id, workspace_id, valid_from, valid_to);
