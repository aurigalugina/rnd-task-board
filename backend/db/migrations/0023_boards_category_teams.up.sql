-- Kategori board (project/routine) + relasi board<->tim -- lihat
-- decision-log-boards-dashboard-enhancements-20260820.md.
-- category NULLABLE TANPA default -- board lama tetap "belum dikategorikan"
-- sampai di-edit manual (sengaja, keputusan eksplisit user).
ALTER TABLE boards ADD COLUMN category TEXT NULL CHECK (category IN ('project', 'routine'));

-- board_teams: many-to-many (satu board boleh di-assign ke lebih dari satu
-- tim). Board baru otomatis ke-assign ke org_team pembuatnya saat create;
-- super_user bisa tambah/ubah assignment lewat edit board.
CREATE TABLE board_teams (
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES referensi_tim(id) ON DELETE CASCADE,
    PRIMARY KEY (board_id, team_id)
);
