-- Rollback migration 0025: Remove soft delete columns from big_tasks

ALTER TABLE big_tasks DROP COLUMN deleted_by;
ALTER TABLE big_tasks DROP COLUMN deleted_at;
