-- Board Archive (super_user only) -- lihat
-- decision-log-board-archive-20260812.md. archived_at = existence-pattern
-- (konsisten big_task_signoffs/change_requests.reviewed_at), archived_by
-- audit trail siapa yang mengarsipkan.
ALTER TABLE boards ADD COLUMN archived_at TIMESTAMPTZ NULL;
ALTER TABLE boards ADD COLUMN archived_by UUID NULL REFERENCES users(id);
