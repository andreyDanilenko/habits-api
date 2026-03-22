DROP INDEX IF EXISTS idx_users_department_id;
ALTER TABLE users DROP COLUMN IF EXISTS department_id;
DROP TABLE IF EXISTS role_object_scopes;
