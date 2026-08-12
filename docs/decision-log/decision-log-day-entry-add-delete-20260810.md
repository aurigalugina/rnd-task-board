# Decision Log — Day Entry Bisa Dihapus & Bisa Lebih dari Satu per Tanggal

**Tanggal:** 2026-08-10
**Konteks:** day-entry-add-delete

## Konteks/Masalah

Generate Daily Task selalu insert TEPAT SATU baris `day_entries` per tanggal kalender dalam rentang `[start_date, end_date]` (SRS FR-DLY-01/02, `UNIQUE (daily_task_id, entry_date)`). User laporkan dua kebutuhan nyata yang gak ke-cover pola ini:

1. Kalau Daily Task rentangnya ≥7 hari, otomatis ada hari weekend ("lembur") ke-generate — tapi PIC-nya belum tentu mau/bisa lembur. Perlu bisa HAPUS baris hari itu (bukan cuma nge-toggle "Belum", karena keberadaan baris itu sendiri ikut jadi penyebut `actual_pct`).
2. Kalau Daily Task rentangnya cuma 1 hari tapi PIC mau breakdown kerjaannya jadi lebih dari satu task/rencana di hari yang sama, sekarang gak bisa — constraint unique bikin cuma boleh 1 baris per tanggal.

## Keputusan

- **`UNIQUE (daily_task_id, entry_date)` DICABUT** — satu Daily Task sekarang boleh punya lebih dari satu `day_entries` di tanggal yang sama, dan boleh juga NOL kalau semuanya dihapus.
- **`DELETE /day-entries/{id}`** (baru) — hapus satu baris permanen (bukan soft-delete/flag).
- **`POST /daily-tasks/{daily_task_id}/day-entries`** (baru) — tambah SATU baris baru untuk tanggal tertentu, body `{ entry_date, planned_text? }`. Dipicu dari tombol "+" per baris tanggal di UI (bukan date-picker bebas) — jadi secara praktik entry_date yang dikirim selalu salah satu tanggal yang sudah ada di Daily Task itu, walau backend-nya sendiri tidak memvalidasi/membatasi ke rentang `start_date`–`end_date` (lihat alasan).
- **`day_entries` nambah kolom `created_at`** — dipakai `ORDER BY entry_date, created_at` di query list, supaya urutan antar-baris yang tanggalnya sama stabil (insertion order), tidak ikut geser kalau salah satu baris di-edit belakangan (`updated_at` berubah tiap PATCH, jadi TIDAK dipakai buat sorting).
- **`actual_pct` TETAP dihitung dari SEMUA baris `day_entries` yang ada** (`done/total`, tidak berubah rumusnya) — otomatis konsisten begitu baris dihapus (penyebut ikut berkurang) atau ditambah (penyebut ikut nambah), tanpa perlu logic baru. Daily Task dengan nol `day_entries` (semua dihapus) tetap valid, `actual_pct` fallback ke 0 (bukan error/crash).

## Alasan

- **Hapus permanen, bukan soft-delete/flag "dikecualikan"**: lebih simpel dan sudah cukup — day_entries bukan record audit/finansial yang perlu jejak, dan `actual_pct` sudah otomatis benar begitu baris hilang dari perhitungan (tidak perlu kolom `excluded`/`is_active` tambahan).
- **`POST` day-entries TIDAK validasi rentang start_date/end_date Daily Task**: constraint itu cuma relevan buat generate OTOMATIS awal (FR-DLY-01/02, masih berlaku apa adanya). Endpoint baru ini eksplisit buat kasus "nambah manual di luar pola otomatis", jadi dibiarkan fleksibel — mempersempitnya cuma menambah kompleksitas validasi tanpa kebutuhan bisnis konkret yang minta itu.
- **Kolom `created_at` ditambah (bukan pakai `updated_at`/`id` buat sorting)**: `updated_at` berubah tiap edit teks/status, kalau dipakai buat `ORDER BY`, baris yang tanggalnya sama bisa lompat urutan pas salah satunya diedit — UX membingungkan (list "tergeser" tanpa alasan jelas dari sisi user). `id` (UUID random) juga tidak representasikan urutan insert. `created_at` murni-append, tidak pernah berubah setelah insert.

## Dampak/File Terpengaruh

- `backend/db/migrations/0010_day_entries_allow_multiple_per_date.up.sql`/`.down.sql` (baru) — drop unique constraint, tambah kolom `created_at`.
- `backend/internal/dailytask/handler.go` — `loadDays` (`ORDER BY entry_date, created_at`), handler baru `AddDayEntry`/`DeleteDayEntry`.
- `backend/cmd/api/main.go` — route baru `POST /daily-tasks/{dailyTaskID}/day-entries`, `DELETE /day-entries/{dayEntryID}`.
- `docs/05-api-contract.md` §5 — dokumentasi endpoint baru.
- `docs/06-db-design.md` §3.8 — update skema `day_entries` (kolom baru, constraint dicabut).
- `frontend/src/lib/components/DailyTaskPanel.svelte` — tombol "+" (tambah entry di tanggal yang sama) dan "🗑" (hapus baris) per row di tabel Day Entry.
