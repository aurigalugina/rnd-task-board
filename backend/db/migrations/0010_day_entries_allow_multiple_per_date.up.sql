ALTER TABLE day_entries DROP CONSTRAINT day_entries_daily_task_id_entry_date_key;
ALTER TABLE day_entries ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
