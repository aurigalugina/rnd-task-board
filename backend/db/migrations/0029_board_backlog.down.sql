ALTER TABLE users DROP COLUMN can_manage_backlog;
DROP INDEX IF EXISTS idx_daily_tasks_source_backlog;
ALTER TABLE daily_tasks DROP COLUMN source_backlog_item_id;
DROP TABLE board_backlog_items;
