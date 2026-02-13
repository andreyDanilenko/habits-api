DROP TRIGGER IF EXISTS tr_workspace_enable_core_modules ON workspaces;
DROP FUNCTION IF EXISTS fn_workspace_enable_core_modules();

DROP TABLE IF EXISTS workspace_modules;
DROP TABLE IF EXISTS modules;
