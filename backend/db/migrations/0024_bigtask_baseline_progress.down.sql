DROP INDEX IF EXISTS idx_daily_tasks_one_baseline_per_bigtask;
ALTER TABLE daily_tasks DROP COLUMN IF EXISTS is_baseline;
