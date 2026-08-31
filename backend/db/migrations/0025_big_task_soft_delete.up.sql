-- Migration 0025: Add soft delete support to big_tasks
-- 
-- Adds deleted_at column to big_tasks table to support soft deletes.
-- Foreign keys from daily_tasks, comments, and other child tables will
-- cascade the visibility filter via application-level queries (deleted_at IS NULL).
-- Hard delete remains possible via manual SQL only (safety feature).

ALTER TABLE big_tasks ADD COLUMN deleted_at TIMESTAMPTZ NULL;
ALTER TABLE big_tasks ADD COLUMN deleted_by UUID NULL REFERENCES users(id);

-- Audit: track who soft-deleted and when
COMMENT ON COLUMN big_tasks.deleted_at IS 'Soft delete timestamp; NULL = not deleted. Used to hide archived Big Tasks from queries without dropping the row.';
COMMENT ON COLUMN big_tasks.deleted_by IS 'User ID who performed the soft delete; NULL if never deleted.';
