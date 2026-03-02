-- Core: проекты (контекст связки модулей) и привязка сущностей модулей к проектам (вариант B — без project_id в модулях).
-- При split Core: projects и project_entities остаются в БД Core.

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_workspace_id ON projects(workspace_id);

-- Привязка сущностей модулей к проектам (модули не хранят project_id).
-- entity_type: 'crm_deal', 'task' и т.д.; entity_id — id в БД модуля.
CREATE TABLE project_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, entity_type, entity_id)
);

CREATE INDEX idx_project_entities_project_type ON project_entities(project_id, entity_type);
CREATE INDEX idx_project_entities_entity ON project_entities(entity_type, entity_id);
