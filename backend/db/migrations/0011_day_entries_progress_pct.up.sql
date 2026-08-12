ALTER TABLE day_entries ADD COLUMN progress_pct SMALLINT NOT NULL DEFAULT 0 CHECK (progress_pct BETWEEN 0 AND 100);
UPDATE day_entries SET progress_pct = CASE WHEN is_done THEN 100 ELSE 0 END;
ALTER TABLE day_entries DROP COLUMN is_done;
