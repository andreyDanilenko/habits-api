-- Лицензии пользователя на модули: куплен для всех воркспейсов или для одного.
-- Включение/выключение в конкретном workspace (workspace_modules) — отдельно:
-- куплен + отключён для видимости = можно включить без повторной покупки.
CREATE TABLE user_module_licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('all_workspaces', 'single_workspace')),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'cancelled')),
    source VARCHAR(20) NOT NULL DEFAULT 'purchase' CHECK (source IN ('purchase', 'admin_grant')),
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_single_workspace_scope CHECK (
        (scope = 'single_workspace' AND workspace_id IS NOT NULL) OR
        (scope = 'all_workspaces' AND workspace_id IS NULL)
    )
);

CREATE UNIQUE INDEX idx_user_module_licenses_all
    ON user_module_licenses (user_id, module_id)
    WHERE scope = 'all_workspaces';

CREATE UNIQUE INDEX idx_user_module_licenses_single
    ON user_module_licenses (user_id, module_id, workspace_id)
    WHERE scope = 'single_workspace';

CREATE INDEX idx_user_module_licenses_user_id ON user_module_licenses(user_id);
CREATE INDEX idx_user_module_licenses_module_id ON user_module_licenses(module_id);
CREATE INDEX idx_user_module_licenses_status ON user_module_licenses(status);

COMMENT ON TABLE user_module_licenses IS 'Лицензия пользователя на модуль: для всех воркспейсов или для одного. Проверяется при включении модуля во воркспейсе.';
COMMENT ON COLUMN user_module_licenses.scope IS 'all_workspaces = можно включать в любом своём воркспейсе; single_workspace = только в указанном workspace_id';
COMMENT ON COLUMN user_module_licenses.source IS 'purchase = оплата; admin_grant = выдача админом';
