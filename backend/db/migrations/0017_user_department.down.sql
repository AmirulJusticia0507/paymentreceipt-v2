DROP INDEX IF EXISTS idx_users_department;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_department_fk;
ALTER TABLE users DROP COLUMN IF EXISTS department_id;
