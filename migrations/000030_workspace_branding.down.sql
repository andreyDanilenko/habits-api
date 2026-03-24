-- =============================================================================
-- Workspace branding rollback: logo + scale
-- =============================================================================

ALTER TABLE workspaces
  DROP COLUMN IF EXISTS logo_path;

ALTER TABLE workspaces
  DROP COLUMN IF EXISTS logo_scale;

