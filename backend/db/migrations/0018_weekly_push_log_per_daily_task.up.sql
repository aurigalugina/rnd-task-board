-- weekly_push_log granularity diubah dari per-big-task menjadi per-daily-task.
-- Alasan: push ke HR sekarang per Daily Task (bukan Big Task), agar judul_task,
-- uraian_task, dan tanggal rencana/due_date bisa dipetakan dengan akurat ke
-- entri harian masing-masing (lihat docs/decision-log).
-- Dev data di weekly_push_log tidak kritikal (bisa di-push ulang), jadi TRUNCATE aman.
TRUNCATE weekly_push_log;
ALTER TABLE weekly_push_log DROP COLUMN big_task_id CASCADE;
ALTER TABLE weekly_push_log
    ADD COLUMN daily_task_id UUID NOT NULL REFERENCES daily_tasks(id) ON DELETE CASCADE;
ALTER TABLE weekly_push_log
    ADD CONSTRAINT weekly_push_log_daily_task_id_week_start_key UNIQUE (daily_task_id, week_start);
