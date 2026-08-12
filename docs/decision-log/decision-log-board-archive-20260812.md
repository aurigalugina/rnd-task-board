# Decision Log — Board Archive (super_user only)

**Tanggal:** 2026-08-12
**Konteks:** board-archive

## Konteks/Masalah

Board yang sudah tidak relevan (project selesai/batal) tetap muncul selamanya di Dashboard dan tab kerja Boards, bikin daftar makin panjang tanpa cara membersihkannya. User minta kemampuan "archive" board — navigasi ke daftar board yang diarsipkan ditaruh di dropdown panel user (di atas Settings), dan halaman itu (plus aksi archive-nya) dibatasi `super_user` saja.

## Keputusan

**Data model**: `boards.archived_at TIMESTAMPTZ NULL` + `boards.archived_by UUID NULL REFERENCES users(id)` (migration `0020`). Existence-pattern seperti `big_task_signoffs`/`change_requests.reviewed_at` — keberadaan `archived_at` = state diarsipkan, bukan boolean terpisah. `archived_by` audit trail siapa yang mengarsipkan (pola sama seperti `weekly_push_log.pushed_by`).

**Akses dua axis, keduanya di-cek in-handler (bukan `RequireRole` middleware)**: `GET /boards/archive`, `PATCH /boards/{id}/archive`, `PATCH /boards/{id}/unarchive` semuanya cek `auth.IsSuperUser(ctx)` sendiri lalu 403 kalau bukan — pola identik `weeklyplan.TeamStatus` (lihat `decision-log-hr-mapping-super-user-20260810.md`), karena `access_level` konsep terpisah dari `roles` many-to-many, bukan sesuatu yang digate lewat `RequireRole`.

**Efek archive BEDA per halaman (bukan "hilang total" atau "cuma ditandai" secara seragam)** — ini keputusan eksplisit user:
- **Dashboard & tab Boards**: board archived HILANG TOTAL. Dicapai dengan satu perubahan minimal: `board.List` (`GET /boards`) sekarang `WHERE archived_at IS NULL`. Endpoint ini dipakai BARENG oleh Dashboard (aggregasi client-side) dan halaman Boards (tab list kerja) — jadi satu filter otomatis menghilangkan board archived dari keduanya sekaligus, tanpa perlu sentuh dua tempat.
- **Weekly Plan & Review Queue**: board archived TETAP MUNCUL APA ADANYA. Query di `weeklyplan.List/Push` dan `reviewqueue.List` JOIN langsung ke `boards`/`daily_tasks` tanpa lewat `board.List` — SENGAJA TIDAK disentuh sama sekali, supaya laporan personal ke HR & antrean review tidak "kehilangan" riwayat cuma karena project-nya sudah diarsipkan.

**Aksi archive/unarchive dari halaman Boards** (bukan dari halaman Board Archive) — tombol "🗄 Archive board" di pojok kanan `.board-pills-row`, cuma muncul kalau `super_user` DAN ada board yang lagi dipilih. Klik memicu modal popup konfirmasi ("Yakin archive board 'X'? ... Ya, archive / Batal") sebelum benar-benar `PATCH .../archive` — REVISI (awalnya inline confirm di bawah tab row, user komplain "UX-nya jelek") jadi modal beneran (`.overlay`/`.modal-box`/`.panel-header`, pola sama seperti `ProfileModal.svelte`/`SettingsModal.svelte`, termasuk Escape-to-close + klik-backdrop-to-close). Bukan `window.confirm()` browser (app ini tidak pakai dialog native di tempat lain). Setelah sukses, board dihapus dari array lokal dan seleksi otomatis pindah ke board pertama yang tersisa (atau kosong kalau sudah tidak ada board).

**Halaman Board Archive** (`/boards/archive`) menampilkan list board archived (nama, deskripsi, "Diarsipkan oleh X · tanggal") + tombol "Unarchive" per baris, reuse class `.queue-row`/`.queue-main`/`.approve-btn` yang sudah ada (pola sama seperti Review Queue) — bukan komponen baru dari nol.

**Nav dropdown**: item "🗄 Board Archive" disisipkan di antara "👤 My Profile" dan "⚙️ Settings" di dropdown user panel (`+layout.svelte`), dibungkus `{#if isSuperUser}`. Berbeda dari My Profile/Settings (modal), Board Archive adalah `<a href="/boards/archive">` beneran (route, bukan modal) — karena isinya listing+aksi yang lebih pas sebagai halaman penuh, bukan pop-up kecil.

## Alasan

- **Existence-pattern (`archived_at` nullable timestamp) bukan boolean**: konsisten sama pola computed-state lain di app ini (sign-off, review, triase change request) — sekaligus dapat "gratis" timestamp buat ditampilkan di UI ("Diarsipkan pada...") tanpa kolom terpisah.
- **In-handler check bukan `RequireRole` group wrapper**: ngikutin pola yang SUDAH ada persis buat kasus `access_level` (beda axis dari `roles`) — bikin pola baru (mis. middleware khusus access_level) cuma nambah dua cara berbeda buat hal yang konsepnya sama.
- **Filter cukup di `GET /boards`, bukan di setiap endpoint yang menyebut board**: karena Dashboard & Boards page memang SATU-SATUNYA konsumer endpoint ini, satu filter di situ otomatis benar buat keduanya, tanpa filter berulang atau field `archived` bocor ke response yang tidak perlu tahu soal itu.
- **Weekly Plan/Review Queue tidak difilter**: keduanya adalah laporan/antrean level TASK (bukan level project) — project boleh selesai/diarsipkan, tapi riwayat kerja & antrean review individual tetap valid dan harus tetap bisa dipertanggungjawabkan.
- **Modal popup (bukan inline confirm, bukan `window.confirm()`)**: aksi ini menyembunyikan board dari Dashboard & tab kerja — cukup konsekuensial buat butuh jeda perhatian penuh (modal + backdrop) ketimbang baris inline yang gampang ke-skip dianggap "cuma UI biasa". Tetap bukan dialog native browser, demi konsistensi visual dengan modal lain di app ini.

## Dampak/File Terpengaruh

- `backend/db/migrations/0020_boards_archive.up/down.sql` (baru).
- `backend/internal/board/handler.go` — `List` ditambah filter `archived_at IS NULL`; handler baru `ListArchived`/`Archive`/`Unarchive` (semua super_user only, cek in-handler).
- `backend/cmd/api/main.go` — route baru `GET /boards/archive`, `PATCH /boards/{boardID}/archive`, `PATCH /boards/{boardID}/unarchive` (grup `protected` biasa, tanpa `RequireRole`).
- `frontend/src/lib/types.ts` — type baru `ArchivedBoard`.
- `frontend/src/routes/boards/archive/+page.svelte` (baru) — halaman listing + unarchive, guard `isSuperUser` di frontend (backend tetap sumber kebenaran otorisasi).
- `frontend/src/routes/boards/+page.svelte` — tombol "Archive board" + inline confirm (super_user only).
- `frontend/src/routes/+layout.svelte` — item dropdown "Board Archive" (super_user only), reactive `isSuperUser`.
- `frontend/src/app.css` — `.dropdown-item` dapat `text-decoration:none` (dipakai juga sebagai `<a>`, bukan cuma `<button>`).
- `docs/05-api-contract.md`, `docs/06-db-design.md` — dokumentasi endpoint & skema baru.
