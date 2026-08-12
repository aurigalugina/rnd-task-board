TRUNCATE weekly_push_log;
ALTER TABLE weekly_push_log DROP COLUMN daily_task_id CASCADE;
ALTER TABLE weekly_push_log
    ADD COLUMN big_task_id UUID NOT NULL REFERENCES big_tasks(id) ON DELETE CASCADE;
ALTER TABLE weekly_push_log
    ADD CONSTRAINT weekly_push_log_big_task_id_week_start_key UNIQUE (big_task_id, week_start);
