-- =============================================================================
-- Workspace branding: logo + scale
-- =============================================================================

ALTER TABLE workspaces
  ADD COLUMN IF NOT EXISTS logo_path TEXT;

ALTER TABLE workspaces
  ADD COLUMN IF NOT EXISTS logo_scale DOUBLE PRECISION NOT NULL DEFAULT 1.0;

