-- Chat MVP: threads (private/group) and messages.
-- Designed for workspace isolation and realtime fanout via ws:workspace:{id}.

CREATE TABLE IF NOT EXISTS chat_threads (
  id UUID PRIMARY KEY,
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('private', 'group')),
  title TEXT NULL,
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_threads_workspace ON chat_threads(workspace_id);

CREATE TABLE IF NOT EXISTS chat_thread_participants (
  thread_id UUID NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (thread_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_thread_participants_user ON chat_thread_participants(user_id);

-- For private chats: enforce uniqueness for a user pair inside a workspace.
-- We keep user_low_id < user_high_id and unique(workspace_id, user_low_id, user_high_id).
CREATE TABLE IF NOT EXISTS chat_private_threads (
  thread_id UUID PRIMARY KEY REFERENCES chat_threads(id) ON DELETE CASCADE,
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  user_low_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_high_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  CHECK (user_low_id < user_high_id),
  UNIQUE (workspace_id, user_low_id, user_high_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_private_threads_workspace ON chat_private_threads(workspace_id);

CREATE TABLE IF NOT EXISTS chat_messages (
  id UUID PRIMARY KEY,
  thread_id UUID NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_thread_created ON chat_messages(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_chat_messages_workspace_created ON chat_messages(workspace_id, created_at);

