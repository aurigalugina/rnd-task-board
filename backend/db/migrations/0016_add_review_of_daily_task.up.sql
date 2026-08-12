-- Menandai daily task sebagai "task review" dari daily task lain (hasil
-- clone-review). NULL = daily task biasa. Dipakai Review Queue untuk
-- menampilkan daily task asal yang direview -- lihat
-- decision-log-bigtask-members-refactor-20260811.md.
ALTER TABLE daily_tasks ADD COLUMN review_of_daily_task_id UUID REFERENCES daily_tasks(id) ON DELETE SET NULL;
CREATE INDEX idx_daily_tasks_review_of ON daily_tasks(review_of_daily_task_id);
