ALTER INDEX idx_big_task_members_user RENAME TO idx_big_task_reviewers_user;
ALTER TABLE big_task_members RENAME TO big_task_reviewers;
