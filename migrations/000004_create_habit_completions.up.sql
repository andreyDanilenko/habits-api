CREATE TABLE habit_completions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL,
    user_id UUID NOT NULL,
    date DATE NOT NULL,
    notes TEXT,
    rating INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_completions_habit_id ON habit_completions(habit_id);
CREATE INDEX idx_completions_user_id ON habit_completions(user_id);
CREATE INDEX idx_completions_date ON habit_completions(date);
CREATE INDEX idx_completions_habit_user_date ON habit_completions(habit_id, user_id, date);
