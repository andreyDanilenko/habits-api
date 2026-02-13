-- Shared Schema (движок ERP): общие сущности для всех модулей
-- Контрагенты и валюты — единый ID в CRM, Финансах, Закупках и т.д.

-- Валюты (на воркспейс: своя база валют)
CREATE TABLE currencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    code VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, code)
);
CREATE INDEX idx_currencies_workspace_id ON currencies(workspace_id);
COMMENT ON TABLE currencies IS 'Shared: справочник валют воркспейса. Ссылаются модули Финансы, CRM, Склад.';

-- Контрагенты (клиенты/поставщики) — один ID во всех модулях
CREATE TABLE counterparties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'client' CHECK (type IN ('client', 'supplier', 'both')),
    email VARCHAR(255),
    phone VARCHAR(50),
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_counterparties_workspace_id ON counterparties(workspace_id);
CREATE INDEX idx_counterparties_type ON counterparties(workspace_id, type);
COMMENT ON TABLE counterparties IS 'Shared: контрагенты (клиенты/поставщики). Один ID в CRM, Финансах, Закупках.';

-- Модуль «Заметки» (простой модуль)
CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notes_workspace_id ON notes(workspace_id);
CREATE INDEX idx_notes_user_workspace ON notes(user_id, workspace_id, created_at DESC);
COMMENT ON TABLE notes IS 'Модуль Заметки: простые заметки в рамках воркспейса.';

-- Заглушки модулей в справочнике (для отображения на фронте и включения/отключения)
INSERT INTO modules (id, code, name, description, is_core) VALUES
    (gen_random_uuid(), 'notes', 'Заметки', 'Простые заметки по воркспейсу', FALSE),
    (gen_random_uuid(), 'inventory', 'Склад', 'Учёт остатков и номенклатуры (в разработке)', FALSE),
    (gen_random_uuid(), 'finance', 'Финансы', 'Проводки и отчёты (в разработке)', FALSE),
    (gen_random_uuid(), 'hr', 'HR', 'Сотрудники и роли (в разработке)', FALSE);
