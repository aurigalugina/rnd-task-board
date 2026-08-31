-- Rollback migration 0026: Remove task_scope_visibility from users

ALTER TABLE users DROP CONSTRAINT check_task_scope_visibility;
ALTER TABLE users DROP COLUMN task_scope_visibility;
