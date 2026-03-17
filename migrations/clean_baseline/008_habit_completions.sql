-- =============================================================================
-- HABITS: Выполнения привычек
-- =============================================================================

CREATE TABLE habit_completions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID,
    date DATE NOT NULL,
    notes TEXT,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    time TIME,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(habit_id, date, user_id)
);

CREATE INDEX idx_completions_habit_id ON habit_completions(habit_id);
CREATE INDEX idx_completions_user_id ON habit_completions(user_id);
CREATE INDEX idx_completions_date ON habit_completions(date);
CREATE INDEX idx_completions_habit_user_date ON habit_completions(habit_id, user_id, date);
CREATE INDEX idx_completions_time ON habit_completions(time);
CREATE INDEX idx_completions_workspace_id ON habit_completions(workspace_id);
