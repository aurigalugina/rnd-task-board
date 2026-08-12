ALTER TABLE day_entries ADD COLUMN is_done BOOLEAN NOT NULL DEFAULT false;
UPDATE day_entries SET is_done = (progress_pct = 100);
ALTER TABLE day_entries DROP COLUMN progress_pct;
