# Decision Log — Dashboard Diagregasi per Board (Project), Bukan per Big Task

**Tanggal:** 2026-08-10
**Konteks:** dashboard-board-level-aggregation

## Konteks/Masalah

Setelah adopsi Design System (mockup), Dashboard lintas board (`routes/+page.svelte`) diagregasi di client dari `GET /boards` + `GET /boards/{id}/big-tasks`, tapi unit hitung yang dipakai di semua stat card, donut chart, progress chart, dan tabel adalah **satu baris per Big Task** — bukan per board. User melaporkan (dengan screenshot) ini salah secara konsep: "Dashboard ini POV-nya mestinya per board, board = project, saat ini per big task." Vision Product-nya Dashboard memang buat lihat progress tim per **project** (Vision Product §7), jadi unit hitungnya harus board, sesuai juga dengan pendekatan awal Fase 1 yang memakai `GET /boards/{id}/summary` (matriks per board) sebelum Dashboard ini di-redesign ulang jadi cross-board di Design System phase.

Masalahnya: satu board bisa punya banyak Big Task dengan status/verdict campur (misal 1 selesai, 1 belum mulai). Belum ada rule yang mendefinisikan bagaimana status/verdict SATU board (project) dilihat sebagai satu kesatuan dari kumpulan Big Task-nya — ini keputusan bisnis baru, dikonfirmasi ke user via `AskUserQuestion` sebelum implementasi.

## Keputusan

**Status board** (belum berjalan / sedang berjalan / sudah selesai / di hold) — all-or-nothing per bucket, default jatuh ke "sedang berjalan" kalau campuran:
- `done` HANYA kalau **SEMUA** Big Task di board itu `signed` (persis sama seperti `project_status` yang sudah ada di `GET /boards/{id}/summary`).
- `hold` HANYA kalau **SEMUA** Big Task (yang belum signed) `on_hold`.
- `not_started` HANYA kalau **SEMUA** Big Task `actual_pct === 0 && !on_hold`, ATAU board belum punya Big Task sama sekali (total = 0).
- Selain ketiga kondisi unanimous di atas (campuran apapun) → `running`. Ini defaultnya, bukan exception — project dianggap "in progress" selama belum seragam 100% ke salah satu state.

**Verdict board** (Won/Lose) — asimetris dengan status, karena filosofi "netral sampai ada titik keputusan" yang sudah dipakai untuk verdict Big Task individual (BRD RULE-04/05/06):
- `lose` kalau **ADA MINIMAL SATU** Big Task di board itu ber-verdict `lose` — sinyal negatif tidak boleh "ketutupan" rata-rata/mayoritas, walau status board-nya sendiri sudah `done`.
- `won` HANYA kalau status board `done` (semua signed) DAN tidak ada satupun yang `lose`.
- Selain itu → netral, tidak dihitung di donut Won/Lose.

**Progress (actual vs expected %) per board** — rata-rata `actual_pct`/`expected_pct` semua Big Task di board itu (bukan berbobot, bukan cuma yang `running`).

**Deadline terdekat per board** — `min(days_left)` dari Big Task yang belum `signed` di board itu (fallback ke semua Big Task kalau semuanya sudah signed).

Ditolak: opsi "mayoritas" (voting berdasar jumlah Big Task terbanyak) dan opsi "weakest-link ke arah belum jalan" (satu Big Task belum mulai bikin seluruh board dianggap belum jalan) — user pilih rekomendasi all-or-nothing/netral-sampai-keputusan di atas untuk kedua pertanyaan.

## Alasan

- **Konsisten dengan computed field yang sudah ada**: rule `done` di atas SAMA PERSIS dengan `project_status` di `GET /boards/{id}/summary` (`backend/internal/board/handler.go`) — tidak menciptakan definisi baru yang bertentangan dengan yang sudah difinalisasi (RULE-08/FR-BRD-07).
- **Verdict lose tidak simetris dengan status** karena secara bisnis, satu kegagalan (lose) adalah sinyal yang harus tetap terlihat SPV walau bagian lain board-nya sukses — beda dengan status "berjalan/belum/hold" yang murni deskriptif progress, bukan sinyal risiko.
- **Agregasi tetap computed-at-read di frontend** (bukan endpoint backend baru) — Big Task per board sudah di-fetch buat kebutuhan lain (progress chart perlu actual/expected mentah per Big Task, bukan cuma count), jadi agregasi ke level board dilakukan di fungsi murni `aggregateBoards` (`lib/dashboardStats.ts`), diuji lewat Vitest, mengikuti pola computed-field yang sudah ada di project ini (tidak disimpan sebagai kolom/endpoint baru).
- **Board tanpa Big Task tetap dihitung** (sebagai `not_started`) — sebelumnya board kosong tidak pernah muncul di agregasi manapun (tidak menghasilkan baris); dengan POV per-board yang benar, board yang baru dibuat dan belum ada Big Task-nya tetap valid disebut "belum berjalan", bukan hilang dari hitungan.

## Dampak/File Terpengaruh

- `frontend/src/lib/dashboardStats.ts` — `BigTaskRow` (input mentah, sebelumnya `BoardRow`) → `aggregateBoards()` (baru) → `BoardAgg` (satu baris per board) → `computeDashboardStats()` (sekarang menerima `BoardAgg[]`, bukan Big Task rows langsung).
- `frontend/src/lib/dashboardStats.test.ts` — test lama (per Big Task) diganti/ditambah test `aggregateBoards` mencakup semua kombinasi status/verdict di atas.
- `frontend/src/routes/+page.svelte` — stat card, donut "Status project"/"Hasil project", progress chart, tabel "Deadline terdekat"/"Berstatus lose" semua sekarang render per board (nama board sebagai label utama, bukan nama Big Task).
- CLAUDE.md — ringkasan rule ini disinkronkan ke bagian Design System/Dashboard.
