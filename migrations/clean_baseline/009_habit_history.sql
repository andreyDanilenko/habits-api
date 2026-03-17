-- =============================================================================
-- HABITS: История изменений привычек
-- =============================================================================

CREATE TABLE habit_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    changes JSONB,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_habit_history_habit_id ON habit_history(habit_id);
CREATE INDEX idx_habit_history_user_id ON habit_history(user_id);
CREATE INDEX idx_habit_history_action ON habit_history(action);
CREATE INDEX idx_habit_history_created_at ON habit_history(created_at);
CREATE INDEX idx_habit_history_habit_created ON habit_history(habit_id, created_at DESC);

COMMENT ON TABLE habit_history IS 'История изменений привычек';
COMMENT ON COLUMN habit_history.action IS 'CREATED, UPDATED, DELETED, COMPLETED';
