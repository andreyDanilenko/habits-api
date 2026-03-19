-- =============================================================================
-- NOTIFICATIONS: Уведомления
-- =============================================================================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    channel VARCHAR(32) NOT NULL DEFAULT 'activity',
    event_type VARCHAR(64) NOT NULL,
    event_key VARCHAR(255) NOT NULL,
    title TEXT NOT NULL,
    subtitle TEXT,
    payload JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP
);

CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id) WHERE read_at IS NULL;
CREATE UNIQUE INDEX idx_notifications_user_event_key ON notifications(user_id, event_key);
