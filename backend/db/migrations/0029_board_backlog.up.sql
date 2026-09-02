-- Board Backlog: item planning mentah per board, sebelum ditentukan
-- Big Task/PIC/tanggal-nya. Reusable (bisa di-"promote" jadi Daily Task
-- berkali-kali, item aslinya tidak hilang) -- lihat
-- decision-log-board-backlog-20260902.md.
CREATE TABLE board_backlog_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_board_backlog_items_board ON board_backlog_items(board_id);

-- Menandai Daily Task yang lahir dari sebuah backlog item ("promote").
-- ON DELETE SET NULL: kalau backlog item-nya dihapus, Daily Task yang
-- sudah dibuat dari situ TETAP ADA (cuma linknya putus) -- backlog item
-- itu sendiri boleh dihapus tanpa mengganggu histori kerja yang sudah jalan.
ALTER TABLE daily_tasks ADD COLUMN source_backlog_item_id UUID
    REFERENCES board_backlog_items(id) ON DELETE SET NULL;
CREATE INDEX idx_daily_tasks_source_backlog ON daily_tasks(source_backlog_item_id);

-- Flag independen dari access_level/roles: siapa yang boleh kelola
-- (tambah/edit/hapus) backlog item -- permintaan user eksplisit,
-- "jangan terpaut sama role", mirip pola task_scope_visibility.
ALTER TABLE users ADD COLUMN can_manage_backlog BOOLEAN NOT NULL DEFAULT false;
