ALTER TABLE day_entries DROP COLUMN created_at;
ALTER TABLE day_entries ADD CONSTRAINT day_entries_daily_task_id_entry_date_key UNIQUE (daily_task_id, entry_date);
