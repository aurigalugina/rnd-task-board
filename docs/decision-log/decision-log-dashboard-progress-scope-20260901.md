# Dashboard: Board "Done" Tetap Ditampilkan (Supersedes 20260810)

**Date:** 2026-09-01
**Status:** Implemented

## Konteks / Masalah

Sejak `decision-log-dashboard-progress-scope-20260810.md`, board berstatus
`done` (semua Big Task sudah sign-off) otomatis disembunyikan dari:
- Progress chart (`GroupedBarChart` — "Progress: actual vs expected (%)")
- Attention-list (baris per-board di bawah chart)
- Tabel "Deadline terdekat" (`nearestDeadline`)

Alasan awal: board `done` dianggap "sudah terwakili" di stat card/donut
summary, jadi gak perlu dipantau progress-nya lagi.

User (Lugi) melaporkan ini bikin project yang baru saja selesai langsung
"menghilang" dari dashboard walau belum sengaja diarsipkan — padahal dia
ingin proses archive itu manual (aksi eksplisit, lewat `GET/POST
/boards/archive`, super_user only — lihat
`decision-log-board-archive-20260812.md`), bukan otomatis berdasarkan status.

## Keputusan

`computeDashboardStats()` di `frontend/src/lib/dashboardStats.ts` diubah:
`activeBoards` sekarang = **semua board yang dikembalikan backend**, TIDAK
lagi memfilter keluar `status === 'done'`. Backend `/boards` sendiri sudah
otomatis mengecualikan board yang di-archive manual, jadi filter status di
frontend ini murni redundan/kontraproduktif dengan alur "archive manual"
yang sudah ada.

Board `done` sekarang tetap muncul di:
- Progress chart & attention-list (dengan actual/expected 100% atau
  mendekati, sesuai data real)
- Tabel "Deadline terdekat" (kalau memang masuk 5 terdekat)

Mekanisme SATU-SATUNYA untuk menghilangkan board dari dashboard adalah
archive manual — bukan status apapun.

## Alasan

- User punya kontrol eksplisit kapan sebuah project "selesai dipantau" vs
  "selesai dikerjakan" — dua hal beda. Status `done` cuma bilang semua Big
  Task sudah sign-off, bukan berarti board itu sudah gak relevan dilihat.
- Auto-hide berdasarkan status menciptakan "hilang diam-diam" yang
  membingungkan (project selesai kemarin, hari ini gak kelihatan lagi di
  progress overview tanpa aksi apapun dari user).
- Archive manual (sudah ada di codebase, super_user only) adalah tempat yang
  tepat untuk "membersihkan" dashboard dari board yang sudah tidak dipakai
  lagi — bukan diotomatisasi diam-diam lewat status field.

## Dampak / File Terpengaruh

- `frontend/src/lib/dashboardStats.ts` — `computeDashboardStats()`:
  `activeBoards = boards` (bukan lagi `.filter(b => b.status !== 'done')`)
- `frontend/src/lib/dashboardStats.test.ts` — test lama
  "excludes ONLY done boards..." diganti dengan test baru yang memverifikasi
  board `done` TETAP muncul di `activeBoards`/`nearestDeadline`
- `frontend/src/routes/+page.svelte` — TIDAK ada perubahan (binding ke
  `activeBoards` sudah otomatis ikut perilaku baru)
- Tidak ada perubahan backend — endpoint `/boards`, `/boards/archive` sudah
  sesuai kebutuhan (archive manual sudah jadi satu-satunya jalur hide)

## Alternatif yang Ditolak

- **Tambah toggle "tampilkan board selesai" di UI**: menambah kompleksitas
  UI untuk masalah yang sebenarnya sudah punya solusi (archive manual).
  Kalau user benar-benar mau board itu hilang dari dashboard, archive adalah
  aksi yang tepat & sudah ada.
- **Auto-archive setelah N hari status done**: berisiko archive board yang
  masih relevan dipantau/di-reference tim, dan menambah scope di luar
  request user ("bisa di kembalikan bisa di munculkan ga bro").
