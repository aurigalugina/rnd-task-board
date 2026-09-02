# Daily Task: Bisa Dihapus Setelah Day Entries-nya Kosong

**Date:** 2026-09-02
**Status:** Implemented

## Konteks / Masalah

Daily Task tidak punya jalur hapus sama sekali sejak awal dibuat -- tidak
ada endpoint `DELETE /daily-tasks/{id}`, tidak ada tombol di UI. Satu-
satunya cara "menghapus" konten Daily Task adalah lewat `DELETE
/day-entries/{id}` per baris (sudah ada, dipakai buat baris weekend/
"lembur" yang PIC-nya tidak mau kerjakan).

User minta: setelah semua Day Entry suatu Daily Task dihapus (kosong),
Daily Task itu sendiri **harus bisa dihapus juga** -- casing paling umum:
Daily Task salah buat / sudah tidak relevan, semua entrinya sudah
dibersihkan satu-satu, tapi "wadah kosong"-nya masih nyangkut selamanya
di board.

## Keputusan

### Endpoint baru: `DELETE /daily-tasks/{daily_task_id}`
Hapus **permanen** (hard delete, bukan soft-delete) -- konsisten dengan
`DeleteDayEntry` yang sudah ada (Day Entry juga hard delete, bukan
`deleted_at`). Ini BEDA dari Big Task (`bigtask.Delete`) yang soft-delete
-- keduanya konsisten dengan tingkat "kehilangan" masing-masing: Big Task
punya histori sign-off/verdict yang harus tetap bisa diaudit walau
"dihapus", Daily Task/Day Entry cuma data kerja harian yang kalau memang
sudah kosong tidak ada nilai historis untuk dipertahankan.

**Gate wajib (409 Conflict):** ditolak selama Daily Task ini MASIH punya
`day_entries` (dicek via `COUNT(*)` sebelum delete). User HARUS
menghabiskan semua baris day entry-nya dulu (satu-satu lewat
`DeleteDayEntry` yang sudah ada) sebelum bisa menghapus Daily Task-nya
sendiri. Ini permintaan eksplisit user -- bukan keputusan sepihak Claude
Code -- mencegah kehilangan histori rencana/realisasi/progress secara
tidak sengaja hanya dengan sekali klik hapus Daily Task.

**Child records lain** (comments, weekly_push_log) sudah `ON DELETE
CASCADE` di skema DB sejak awal -- tidak perlu penanganan manual di
handler, terhapus otomatis begitu Daily Task-nya dihapus.
`review_of_daily_task_id` (kalau Daily Task lain me-review Daily Task
ini) sudah `ON DELETE SET NULL` -- relasi diputus, Daily Task reviewer-nya
sendiri TIDAK ikut terhapus.

### Frontend (`DailyTaskPanel.svelte`)
Tombol hapus (ikon tempat sampah) baru di header tiap Daily Task card,
sebelah tombol "+ Review" yang sudah ada. **Disabled** (opacity turun,
cursor not-allowed, tooltip menjelaskan kenapa) kalau `dt.days.length >
0` -- guard di sisi client supaya user tidak perlu klik dulu baru tahu
gagal, backend tetap jadi sumber kebenaran (409) untuk kasus race
condition (mis. dua tab terbuka bersamaan).

## Alasan

- Hard delete (bukan soft-delete) untuk konsistensi dengan
  `DeleteDayEntry` yang sudah ada -- kedua entitas ini "child" dari Big
  Task, levelnya sama secara semantik data (day-to-day working data, bukan
  keputusan bisnis/sign-off yang perlu diaudit).
- Gate "harus kosong dulu" adalah permintaan eksplisit user (bukan
  interpretasi Claude Code) -- alternatif "hapus langsung cascade semua
  entries sekaligus" akan jauh lebih berisiko (satu klik menghapus
  histori bertahun-hari tanpa langkah konfirmasi bertahap), dan user
  secara spesifik menyebut precondition "sudah dihapus dan kosong" di
  permintaannya.
- Guard client-side (disabled button) DITAMBAH validasi server-side (409)
  -- pola defense-in-depth yang sama dipakai fitur-fitur lain di sesi ini
  (mis. gate edit Team Today).

## Dampak / File Terpengaruh

- `backend/internal/dailytask/handler.go` -- `DeleteDailyTask()` baru.
- `backend/cmd/api/main.go` -- route `DELETE /daily-tasks/{dailyTaskID}`.
- `frontend/src/lib/components/DailyTaskPanel.svelte` -- tombol hapus di
  header Daily Task card, `deleteDailyTask()`, state
  `deletingDailyTaskId`.
- `frontend/src/app.css` -- `.icon-btn:disabled` (belum ada styling
  disabled sebelumnya untuk tombol ikon manapun -- gap yang baru
  ketemu & sekalian diperbaiki di sini).

## Verifikasi

- `go build ./...` -- sukses. `go test ./...` -- semua paket yang
  disentuh lulus (paket `board` tetap gagal karena bug pre-existing tidak
  terkait, sudah di-flag sebelumnya).
- Tidak ada logic computed murni baru yang perlu unit test terpisah --
  `DeleteDailyTask` murni query COUNT + validasi + DELETE, tidak ada
  kalkulasi/derivasi nilai yang layak diekstrak jadi fungsi murni (beda
  dengan `canEditDayEntry`/`isValidSeverity` di task-task sebelumnya).
- `npm run check` -- 0 errors. `npm run test` -- 129/129 passed (tidak
  ada logic non-trivial baru di frontend, guard `dt.days.length > 0`
  cukup jelas untuk divalidasi manual).
- Manual end-to-end via curl, 3 skenario:
  1. Daily task dengan 2 day entries -> `DELETE /daily-tasks/{id}` ->
     **409**, pesan "hapus dulu semua day entries sebelum bisa hapus
     daily task ini".
  2. Kedua day entries dihapus satu-satu (`DELETE /day-entries/{id}` x2,
     masing-masing 204) -> `DELETE /daily-tasks/{id}` lagi -> **204**.
  3. `DELETE /daily-tasks/{id}` sekali lagi (sudah tidak ada) -> **404**.
- Local Docker rebuild + container recreate untuk backend + nginx -- OK.

## Alternatif yang Ditolak

- **Cascade delete Daily Task langsung menghapus semua day_entries
  sekaligus** (tanpa precondition kosong): ditolak -- user secara
  eksplisit minta precondition "sudah dihapus dan kosong", dan cascade
  langsung jauh lebih berisiko (satu klik = hilang histori bertahun-hari
  tanpa jejak konfirmasi bertahap).
- **Soft-delete (deleted_at) seperti Big Task**: ditolak -- Daily
  Task/Day Entry bukan entitas yang butuh audit trail jangka panjang
  seperti Big Task (yang punya sign-off/verdict); konsisten dengan
  `DeleteDayEntry` yang sudah hard-delete sejak awal.
