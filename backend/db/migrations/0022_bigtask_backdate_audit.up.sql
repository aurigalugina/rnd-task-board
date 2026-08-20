-- Verdict backfill/sign-off historis -- lihat
-- decision-log-verdict-backfill-signoff-20260820.md.
-- signed_at_backdated_by: NULL = sign-off normal (signed_at = waktu klik asli).
-- Terisi (FK ke users) = super_user yang override signed_at manual ke tanggal
-- lampau (backfill data historical) -- audit trail biar bisa dibedakan dari
-- sign-off real-time asli. signed_by TETAP actor yang klik, gak berubah makna.
ALTER TABLE big_task_signoffs ADD COLUMN signed_at_backdated_by UUID NULL REFERENCES users(id);

-- updated_by: siapa super_user yang terakhir edit judul/tanggal Big Task
-- (PATCH /big-tasks/{id}, super_user only) -- updated_at sudah ada dari awal,
-- tinggal ditambah siapanya.
ALTER TABLE big_tasks ADD COLUMN updated_by UUID NULL REFERENCES users(id);
