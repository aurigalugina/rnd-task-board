-- big_task_reviewers berubah makna jadi "anggota tim yang terlibat di Big Task"
-- (bukan khusus reviewer) -- lihat decision-log-bigtask-members-refactor-20260811.md.
ALTER TABLE big_task_reviewers RENAME TO big_task_members;
ALTER INDEX idx_big_task_reviewers_user RENAME TO idx_big_task_members_user;
