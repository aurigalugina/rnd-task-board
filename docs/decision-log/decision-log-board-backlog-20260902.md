# Board Backlog: Menu Planning Mentah Sebelum Assign PIC/Tanggal/Big Task

**Date:** 2026-09-02
**Status:** Implemented

## Konteks / Masalah

User butuh cara mencatat kerjaan yang **sudah kebayang tapi belum jelas**
siapa yang pegang, kapan dikerjakan, atau bahkan masuk Big Task yang mana --
"gue udah bisa mapping task pekerjaan nya dulu listing dulu... semacam
backlog yang nantinya itu akan jadi pilihan daily task dan di bigtask yang
mana". Struktur data lama (Daily Task) tidak punya tempat untuk ini --
Daily Task WAJIB punya PIC, start/end date, DAN terikat ke satu Big Task
tertentu sejak awal dibuat.

Didiskusikan lewat serangkaian clarify sebelum implementasi (lihat riwayat
percakapan 2026-09-02):
- Scope: **per board** (bukan per Big Task atau global lintas board).
- Visibility: **List = semua user bisa lihat** (transparan).
- Manage (Create/Update/Delete): awalnya diusulkan super_user/SPV, tapi
  user KOREKSI eksplisit: **"jangan terpaut sama role... pake flagging aja
  bro di user"** -- harus flag independen di user model, bukan role/access_level.
- UI: **tab baru "Backlog" di /boards**, sejajar tab Big Task.
- Alur promote: **item TETAP ADA/reusable** setelah dipromosikan jadi Daily
  Task (bukan sekali pakai/hilang) -- cocok untuk kerjaan recurring.
- Bonus request: **Excel import**, dengan **template harus disediakan
  LEBIH DULU** sebelum parser dibangun -- "sediain dulu templatenya biar
  user bisa isi tanpa ada kesalahan".

## Keputusan

### Skema DB (migration 0029)
```sql
CREATE TABLE board_backlog_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE daily_tasks ADD COLUMN source_backlog_item_id UUID
    REFERENCES board_backlog_items(id) ON DELETE SET NULL;

ALTER TABLE users ADD COLUMN can_manage_backlog BOOLEAN NOT NULL DEFAULT false;
```
- `board_backlog_items`: entitas independen, cuma judul (wajib) + deskripsi
  (opsional) -- SENGAJA tidak punya PIC/tanggal/big_task_id, itu justru
  poin utamanya (belum ditentukan).
- `daily_tasks.source_backlog_item_id` **ON DELETE SET NULL** (bukan
  CASCADE) -- kalau backlog item dihapus, Daily Task yang sudah dibuat
  dari situ TETAP ADA (cuma link-nya putus). Backlog item boleh dihapus
  tanpa mengganggu histori kerja yang sudah berjalan.
- `users.can_manage_backlog`: flag boolean independen, default `false`,
  pola identik dengan `task_scope_visibility` (kolom terpisah, bukan
  role/access_level) -- persis permintaan user.

### Backend: package baru `backlog`
- `canManageBacklog(isSuperUser, userCanManage bool) bool` -- fungsi murni:
  `isSuperUser OR userCanManage`. super_user selalu bisa kelola (bypass,
  tidak perlu di-flag manual satu-satu); selain itu HARUS flag true.
- `ListByBoard` (GET) -- **TIDAK digate** can_manage_backlog, semua user
  authenticated bisa lihat (permintaan eksplisit: transparan).
- `Create`/`Update`/`Delete` -- digate `requireManagePermission()` (helper
  yang query `can_manage_backlog` user dari DB lalu panggil
  `canManageBacklog`).
- `PromotedCount` di response List -- dihitung via subquery `COUNT(*)`
  dari `daily_tasks` yang `source_backlog_item_id` mengarah ke item itu --
  badge "Nx dipakai" di UI, TANPA perlu kolom counter terpisah yang bisa
  jadi stale.
- `DownloadTemplate` (GET) + `Import` (POST) -- pakai `excelize` (sama
  library dengan `dataport` package export/import Big Task/Daily Task
  yang sudah ada), tapi package terpisah (`backlog`, bukan ditambahkan ke
  `dataport`) karena scope-nya spesifik ke satu entitas simpel, tidak
  perlu ikut kompleksitas resolusi board/big-task/daily-task/day-entry
  berlapis yang dipunya `dataport.Import`.

### Backend: Daily Task Create menerima `source_backlog_item_id` opsional
`createDailyTaskRequest.SourceBacklogItemID *string` -- kalau diisi,
`insertDailyTaskWithDays()` menyimpannya di kolom baru. Ini yang menautkan
Daily Task hasil "promote" balik ke backlog item asalnya. Field ini
OPSIONAL dan backward-compatible -- form create Daily Task biasa (bukan
lewat promote) tetap jalan normal tanpa field ini (nil).

### Frontend
- `BacklogSection.svelte` (komponen baru) -- list item + form
  tambah/edit (gated `canManage`) + toolbar Template/Import (gated
  `canManage`) + tombol "Jadikan Daily Task" (SEMUA user bisa promote,
  bukan cuma yang punya `can_manage_backlog` -- promote itu aksi create
  Daily Task biasa, ikut aturan permission Daily Task yang sudah ada,
  BUKAN aturan manage-backlog).
- Tab baru "Backlog" di `/boards/+page.svelte`, sejajar tab "Big Task" --
  toggle `boardTab` state, `{#key selectedBoardId}` supaya komponen
  re-mount bersih saat pindah board.
- Modal promote: pilih Big Task (dropdown semua Big Task di board itu) ->
  field PIC muncul dibatasi ke ANGGOTA Big Task terpilih (pola sama
  persis dengan form create Daily Task biasa) -> tanggal mulai/selesai ->
  submit. Judul Daily Task otomatis terisi dari judul backlog item.
- `SettingsModal.svelte` (Manajemen User) -- checkbox baru "Boleh kelola
  backlog" di form edit user, sejajar dropdown `task_scope_visibility`
  yang sudah ada -- ini satu-satunya tempat toggle flag `can_manage_backlog`
  (cuma bisa diakses admin/spv, sama seperti field lain di form itu).

### Template Excel -- 2 SHEET terpisah (bug ditemukan & diperbaiki saat verifikasi)
Percobaan pertama menaruh instruksi/petunjuk sebagai baris tambahan DI
SHEET YANG SAMA (di bawah header+contoh) -- saat diverifikasi manual,
ternyata parser Import salah mengira baris-baris instruksi itu adalah
baris data (karena parser cuma skip 1 baris header, lalu proses semua
baris sisanya). Diperbaiki dengan memisah jadi 2 sheet: sheet "Backlog"
HANYA berisi header + 1 baris contoh (murni data), sheet "Petunjuk"
berisi instruksi bebas. `Import` membaca sheet "Backlog" secara eksplisit
by name (dengan fallback ke sheet pertama kalau nama itu tidak ada, mis.
user rename sheet-nya).

## Alasan

- `board_backlog_items` sebagai tabel independen (bukan `daily_tasks`
  dengan kolom nullable) -- lebih bersih secara semantik: backlog item
  BUKAN "Daily Task yang belum lengkap", dia entitas planning yang beda,
  dengan lifecycle (reusable) yang beda juga.
- Flag `can_manage_backlog` independen dari `access_level`/roles --
  permintaan eksplisit user untuk decoupling permission dari role
  hierarchy, konsisten dengan pola `task_scope_visibility` yang sudah
  established di codebase ini.
- `ON DELETE SET NULL` (bukan CASCADE) untuk `source_backlog_item_id` --
  menghapus item backlog adalah aksi "bersihkan planning yang sudah tidak
  relevan", BUKAN "batalkan semua pekerjaan yang pernah lahir darinya" --
  dua aksi yang secara semantik sangat berbeda, tidak boleh tercampur
  dalam satu operasi delete.
- Template 2 sheet terpisah -- satu-satunya cara menjamin parser Import
  TIDAK PERNAH salah mengira instruksi sebagai data, tanpa perlu heuristik
  rapuh (mis. "skip baris yang mengandung kata 'Petunjuk'").
- Promote TIDAK digate `can_manage_backlog` -- backlog cuma soal SIAPA
  BOLEH BIKIN/EDIT/HAPUS daftar rencana; siapa boleh MENGEKSEKUSI (bikin
  Daily Task beneran) sudah punya aturan sendiri (aturan Daily Task Create
  yang sudah ada) dan tidak boleh tertukar.

## Dampak / File Terpengaruh

- `backend/db/migrations/0029_board_backlog.{up,down}.sql`
- `backend/internal/backlog/handler.go` (package baru) -- `Item` struct,
  `canManageBacklog()`, `requireManagePermission()`, `ListByBoard`,
  `Create`, `Update`, `Delete`, `DownloadTemplate`, `Import`.
- `backend/internal/backlog/handler_test.go` -- `TestCanManageBacklog`
  (4 kasus).
- `backend/internal/dailytask/handler.go` -- `createDailyTaskRequest.SourceBacklogItemID`,
  `insertDailyTaskWithDays()` signature bertambah param, kedua pemanggil
  (`Create`, `CloneReview`) diupdate.
- `backend/internal/user/handler.go` -- `User.CanManageBacklog`,
  `updateUserRequest.CanManageBacklog`, `List`/`Update`/`loadMe` query +
  scan diupdate.
- `backend/cmd/api/main.go` -- import `backlog`, wiring
  `backlogHandler`, 6 route baru.
- `frontend/src/lib/types.ts` -- `BacklogItem`, `ManagedUser.can_manage_backlog`.
- `frontend/src/lib/stores/authStore.ts` -- `UserSummary.can_manage_backlog`.
- `frontend/src/lib/components/BacklogSection.svelte` (komponen baru).
- `frontend/src/lib/components/SettingsModal.svelte` -- checkbox "Boleh
  kelola backlog" di form edit user.
- `frontend/src/routes/boards/+page.svelte` -- tab switcher Big
  Task/Backlog.
- `frontend/src/app.css` -- `.board-subtabs`, `.backlog-*` (di dalam
  komponen).

## Verifikasi

- `go build ./...` -- sukses. `go test ./...` -- semua paket yang
  disentuh lulus (paket `board` tetap gagal karena bug pre-existing tidak
  terkait, di-flag di sesi-sesi sebelumnya).
- `npm run check` -- 0 errors. `npm run test` -- 129/129 passed.
- Manual end-to-end via curl, mencakup:
  1. `can_manage_backlog=false` (regular_user) -> `POST backlog-items` ->
     **403**.
  2. `can_manage_backlog=true` -> `POST` -> **201**.
  3. `GET backlog-items` dengan `can_manage_backlog=false` -> tetap
     **200** (List transparan, tidak digate).
  4. `GET .../template` -> file XLSX valid ter-download (diverifikasi via
     `zipfile` -- XLSX adalah ZIP).
  5. **Bug ditemukan+diperbaiki**: import template versi pertama (1
     sheet, instruksi di baris bawah) salah membuat 6 item (5 di
     antaranya baris instruksi, bukan data). Diperbaiki dengan 2-sheet
     terpisah, re-test -> `{"created":1}` (cuma baris contoh asli),
     verified via `GET` isinya benar.
  6. Promote: `POST .../daily-tasks` dengan `source_backlog_item_id` ->
     Daily Task lahir, `GET backlog-items` -> `promoted_count` naik jadi
     1, item TETAP ADA.
  7. `DELETE backlog-items/{id}` pada item yang sudah dipromosikan ->
     **204**, lalu verifikasi Daily Task yang dibuat SEBELUMNYA masih
     ada di DB dengan `source_backlog_item_id = NULL` (bukan ikut
     terhapus).
- Migration `0029` applied ke DB lokal, skema diverifikasi via `\d`.
- Local Docker rebuild + container recreate untuk backend + nginx -- OK.

## Alternatif yang Ditolak

- **Backlog CRUD permission = role (super_user/SPV)**: DITOLAK oleh user
  secara eksplisit -- diganti flag independen `can_manage_backlog`.
- **Cascade delete Daily Task saat backlog item dihapus**: ditolak --
  akan menghapus histori kerja yang sudah berjalan hanya karena item
  planning sumbernya dibersihkan, dua aksi semantik yang berbeda.
- **Backlog item "habis" sekali dipromosikan** (hilang/auto-archive):
  ditolak oleh user secara eksplisit -- item harus reusable untuk kerjaan
  recurring.
- **Template 1 sheet dengan instruksi di baris bawah data**: dicoba
  pertama, TERBUKTI BUG (parser salah baca instruksi sebagai data) --
  diganti 2 sheet terpisah.
