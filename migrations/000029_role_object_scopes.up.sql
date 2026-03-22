-- Scope видимости данных по роли и объекту (ABAC-слой поверх Casbin).
-- object_key в формате как у Casbin obj: "crm:deal", "crm:contact", ...

CREATE TABLE role_object_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    object_key VARCHAR(128) NOT NULL,
    data_scope VARCHAR(32) NOT NULL CHECK (data_scope IN ('all', 'owner', 'department', 'none')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, object_key)
);

CREATE INDEX idx_role_object_scopes_role ON role_object_scopes(role_id);

-- Опционально: отдел для фильтра department (пока без отдельной справочной таблицы).
ALTER TABLE users ADD COLUMN IF NOT EXISTS department_id UUID;

CREATE INDEX idx_users_department_id ON users(department_id) WHERE department_id IS NOT NULL;
