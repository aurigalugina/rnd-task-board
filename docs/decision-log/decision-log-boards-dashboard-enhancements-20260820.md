# Decision Log — Boards & Dashboard Enhancement Batch

**Tanggal:** 2026-08-20
**Konteks:** boards-dashboard-enhancements
**Status:** 🟡 Hampir final — 2 poin terakhir (ditandai ⏳ di bawah) masih proposal saya, tunggu konfirmasi user. SISANYA sudah disepakati. Jangan implementasi apa pun sampai user bilang "gas".

## 1a. Cheat Sheet Edit & Delete — FINAL

- `PATCH /boards/{boardID}/cheat-sheet/{itemID}` + `DELETE /boards/{boardID}/cheat-sheet/{itemID}`, super_user only (in-handler check).
- Edit boleh ubah **semua field** termasuk `type` (file/url/note bisa saling diganti) — dikonfirmasi user, beda dari usulan awal saya (title+value doang).
- Kalau item type "file" dihapus/diganti type-nya, file fisik di `UPLOAD_DIR` TIDAK ikut dihapus (modul `upload` gak punya delete-file) — orphan file diterima sebagai limitasi.

## 1b. Board Edit + Kategori + Filter Tim/User — FINAL (kecuali 2 poin ⏳)

### Kategori project/routine
- Kolom baru `boards.category TEXT NULL CHECK (category IN ('project','routine'))` — **nullable, TANPA default**. Board lama tetap NULL ("belum dikategorikan") sampai di-edit manual satu-satu oleh super_user — user EKSPLISIT nolak default otomatis ("nanti gue bisa edit aja sih kategorinya").
- `POST /boards` (Create): field `category` opsional, boleh dipilih user BIASA saat create (form dropdown, default terpilih "Project" tapi bisa diganti). Board baru TIDAK dipaksa NULL — user yang create langsung nentuin.
- `PATCH /boards/{id}` (edit kategori + deskripsi): **super_user only**. Regular user cuma bisa nentuin kategori PAS create, gak bisa ubah lagi setelahnya (harus lewat super_user).
- Dashboard: **toggle filter** (Semua / Project / Routine) — bukan default dipersempit. Board `category IS NULL` cuma muncul di filter "Semua" (gak masuk hitungan "Project" maupun "Routine" sampai dikategorikan) — insentif alami buat beresin kategorisasi data lama.

### Filter Tim & User (fitur baru, disiapkan untuk ke depannya — bakal ada tim lain selain 'R&D')
- **Board ↔ Tim = many-to-many**: tabel baru `board_teams (board_id, team_id)`. Satu board boleh di-assign ke lebih dari satu tim.
- **Level proteksi = UI/query filter dulu, BUKAN enforcement keras server-side.** `GET /boards` (dan turunannya) difilter server-side via query (bukan hack di frontend), tapi endpoint detail per-board (`/boards/{id}/big-tasks`, dst) TIDAK dikunci ketat kalau diakses langsung by ID — konsisten "internal tool kecil, trust-based", MVP dulu. Bisa dinaikkan ke enforcement penuh nanti kalau kebutuhannya berubah (topik terpisah).
- **Regular user**: otomatis kebatasi lihat board yang `board_teams`-nya mengandung `org_team` dia — TANPA picker tim (implisit, gak ada pilihan). Dalam scope itu, ada toggle **"Saya" / "Semua"**.
- **Super user**: PUNYA picker tim ("Semua tim" atau pilih 1 tim spesifik) — cuma super_user yang bisa ganti-ganti tim. Buat dimensi user, dropdown **pilih user tertentu / "Semua"** (bukan cuma toggle biner kayak regular user).
- ⏳ **Proposal saya (perlu dikonfirmasi):** board yang BARU dibuat otomatis ke-assign ke `org_team` si pembuatnya (auto-populate `board_teams` 1 baris saat create) — biar board baru gak "ilang"/invisible sampai ada super_user yang assign tim manual. Super_user tetap bisa nambah/ubah assignment tim board itu belakangan lewat edit (mis. kalau board itu ternyata lintas tim). **Setuju?**
- ⏳ **Proposal saya (perlu dikonfirmasi) — scope filter "Saya"/user tertentu**: filter ini mempersempit Dashboard (stat card, chart, attention-list, section "Tim") ke Big Task yang user itu jadi **anggota** (`member_user_ids` mengandung dia) — bukan cuma "PIC Daily Task". "Semua" = gak ada pembatasan tambahan (selain filter tim yang udah aktif). **Setuju definisi ini, atau maunya berdasarkan PIC Daily Task, bukan member Big Task?**

## 1c. Collapse Daily Task + Filter Status — FINAL

- Collapse: **localStorage browser doang, TIDAK persist ke DB** — murni preferensi tampilan, gak ada kolom/endpoint baru.
- **Default: SEMUA Daily Task card collapsed** (bukan cuma opsional expand — defaultnya nge-hide).
- Tambahan: filter **"Tampilkan yang sudah selesai/lampau"** (toggle, default OFF) — defaultnya cuma nampilin yang "ongoing". Definisi "ongoing" (proposal saya, tolong dikonfirmasi juga): Daily Task dianggap SELESAI/LAMPAU (disembunyikan dari default view) kalau `actual_pct === 100` ATAU `end_date` sudah lewat hari ini — sisanya (masih berjalan) dianggap "ongoing" dan tetap muncul default.

## 2a. Chart label tabrakan — SOLUSI DIUBAH (lebih baik dari usulan awal saya)

User usul: board/project dapat **icon atau label warna sendiri yang konsisten**, chart-nya cukup nampilin warna/icon itu (bukan teks nama panjang). Saya cek — ini PAS banget karena **`DonutChart.svelte` yang sudah ada di app ini justru sudah pakai pola PERSIS ini** (warna segmen + legend terpisah di sampingnya, bukan teks nempel di chart). Jadi ini BUKAN pola baru, tinggal diterapin ke `GroupedBarChart` juga — lebih robust daripada usulan awal saya (truncate dinamis + rotasi label), karena gak akan pernah "kehabisan ruang" mau berapa pun banyaknya board.

- Tiap board dapat warna deterministik (hash dari `board.id`, bukan dipilih manual — konsisten otomatis, gak nambah UI buat "pilih warna").
- Chart: bar tanpa teks nama di bawahnya (atau teks dihapus total), warna bar = warna board itu.
- Legend terpisah di samping/bawah chart: swatch warna + nama board (teks bisa wrap normal di situ, gak dibatasi lebar kolom bar).

## 2b. Start/Due Date di Dashboard — FINAL

Masuk tabel "Deadline terdekat" (`Nama project | Start | Due | Sisa hari | Actual`):
- **Start** = `MIN(start_date)` semua Big Task board itu.
- **Due** = deadline dari Big Task yang sama yang jadi sumber angka "Sisa hari" (biar konsisten, bukan `MAX(deadline)` terpisah).

## Dampak/File Terpengaruh (perkiraan)

- **Migration baru:** `boards.category`, tabel `board_teams (board_id, team_id)`.
- `backend/internal/cheatsheet/handler.go` — `Update`/`Delete`.
- `backend/internal/board/handler.go` — `Update` (deskripsi+kategori, super_user only); `Create` terima `category` opsional + auto-insert `board_teams` (tim pembuat); `List` terima query filter `team_id`/`category` (server-side).
- `backend/internal/user/handler.go` (`ProgressSummary`) — kemungkinan perlu terima filter tambahan (tim/user) kalau scope 1b diterapkan ke situ juga.
- `backend/cmd/api/main.go` — route baru (cheat-sheet PATCH/DELETE, board PATCH).
- `frontend/src/lib/components/GroupedBarChart.svelte` — ganti ke warna+legend (bukan truncate+rotate).
- `frontend/src/lib/dashboardStats.ts` — `aggregateBoards`/`BoardAgg` dapat `startDate`/`dueDate`; kemungkinan util warna deterministik per board.
- `frontend/src/routes/+page.svelte` — kolom Start/Due; UI filter tim (super_user)/user (semua)/kategori.
- `frontend/src/lib/components/DailyTaskPanel.svelte` — collapse (localStorage) + toggle filter status.
- `frontend/src/routes/boards/+page.svelte` / `CheatSheetSection.svelte` — tombol Edit/Delete cheat sheet, tombol Edit board, dropdown kategori di form Create.
- `docs/06-db-design.md`/`05-api-contract.md` — update skema/endpoint baru.

## Belum diputuskan (2 poin terakhir, tunggu konfirmasi user)

- [ ] Board baru auto-assign ke tim pembuatnya saat create — setuju?
- [ ] Scope filter "Saya"/user tertentu = berdasarkan member Big Task, bukan PIC Daily Task — setuju?
- [ ] Definisi "ongoing" buat filter status Daily Task (1c) — `actual_pct<100 AND end_date>=hari ini` — setuju?
