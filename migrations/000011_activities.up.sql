-- =============================================================================
-- ACTIVITIES: Глобальная лента активности
-- =============================================================================

CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    emoji VARCHAR(10),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_activities_user_id ON activities(user_id);
CREATE INDEX idx_activities_workspace_id ON activities(workspace_id);
CREATE INDEX idx_activities_type ON activities(type);
CREATE INDEX idx_activities_created_at ON activities(created_at);
CREATE INDEX idx_activities_user_workspace_created ON activities(user_id, workspace_id, created_at DESC);
CREATE INDEX idx_activities_entity_created ON activities (workspace_id, entity_type, entity_id, created_at DESC);

COMMENT ON TABLE activities IS 'Активность для виджета RecentActivity';
COMMENT ON INDEX idx_activities_entity_created IS 'Для выборки активностей по сущности (task, habit, journal)';
