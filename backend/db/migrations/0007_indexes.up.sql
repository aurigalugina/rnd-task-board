CREATE INDEX idx_big_tasks_board_id ON big_tasks(board_id);
CREATE INDEX idx_daily_tasks_big_task_id ON daily_tasks(big_task_id);
CREATE INDEX idx_day_entries_daily_task_id ON day_entries(daily_task_id);
CREATE INDEX idx_day_entries_entry_date ON day_entries(entry_date);
CREATE INDEX idx_comments_big_task_id ON comments(big_task_id);
CREATE INDEX idx_comments_daily_task_id ON comments(daily_task_id);
CREATE INDEX idx_cheat_sheet_items_board_id ON cheat_sheet_items(board_id);
CREATE INDEX idx_weekly_push_log_lookup ON weekly_push_log(big_task_id, week_start);
