-- Journal entries (часть модуля habits, привязка к workspace). Одно поле — описание.
CREATE TABLE journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description TEXT NOT NULL DEFAULT '',
    mood INTEGER CHECK (mood IS NULL OR (mood >= 1 AND mood <= 5)),
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    tags TEXT[] NOT NULL DEFAULT '{}',
    content_type VARCHAR(20) NOT NULL DEFAULT 'text' CHECK (content_type IN ('text', 'markdown')),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_journal_entries_workspace_id ON journal_entries(workspace_id);
CREATE INDEX idx_journal_entries_user_workspace ON journal_entries(user_id, workspace_id, date DESC);
CREATE INDEX idx_journal_entries_date ON journal_entries(workspace_id, date DESC);

COMMENT ON TABLE journal_entries IS 'Записи дневника (модуль habits). Доступ по workspace_id.';
