-- Add parent_id for threaded replies
ALTER TABLE task_comments ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES task_comments(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_task_comments_parent ON task_comments(parent_id);
