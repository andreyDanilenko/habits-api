CREATE TABLE habits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50) NOT NULL DEFAULT '#3B82F6',
    icon VARCHAR(100),
    target_days INTEGER DEFAULT 7,
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_habits_user_id ON habits(user_id);
CREATE INDEX idx_habits_workspace_id ON habits(workspace_id);
CREATE INDEX idx_habits_created_at ON habits(created_at);
