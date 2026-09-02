# Team Today: Gate Edit ke Pemilik Entry ATAU Akses "Lihat Semua Orang"

**Date:** 2026-09-02
**Status:** Implemented

## Konteks / Masalah

Team Today (lihat `decision-log-team-today-inline-edit-20260902.md`) baru
saja dapat kemampuan inline edit -- tapi TANPA permission check apapun:
siapapun yang authenticated bisa PATCH day entry milik siapapun (backend
`UpdateDayEntry` tidak pernah punya permission check sejak awal dibuat).

User minta pembatasan eksplisit: **akses edit hanya boleh oleh user yang
bersangkutan (pemilik entry) ATAU user yang punya akses "lihat semua
orang"** ("*Lihat semua task tim" -- match istilah UI di SettingsModal
dropdown `task_scope_visibility`, opsi "team").

## Keputusan

### Backend: `canEditDayEntry()` + gate di `UpdateDayEntry`
Fungsi murni baru di `dailytask/handler.go`:
```go
func canEditDayEntry(userID, picUserID string, isSuperUser bool, userScope string) bool {
    if userID == picUserID { return true }
    return isSuperUser || userScope == "team"
}
```
Dipanggil di awal `UpdateDayEntry` (sebelum decode body) -- query JOIN
`day_entries -> daily_tasks -> users` buat ambil `pic_user_id` (pemilik
entry) dan `task_scope_visibility` viewer. `super_user` selalu lolos
(short-circuit sebelum query, sama pola dengan `ListByBigTask`). Ditolak
403 dengan pesan "tidak punya akses untuk mengubah day entry ini".

**Ini mengubah behavior lama** (yang tidak punya gate sama sekali) --
sekarang user `task_scope_visibility='self'` yang BUKAN pemilik entry akan
ditolak PATCH, termasuk lewat `DailyTaskPanel`/`DayEntryEditModal` (bukan
cuma Team Today) -- KONSISTEN karena semuanya reuse endpoint yang sama.
Tapi user scope='self' pada dasarnya sudah tidak bisa MELIHAT daily task
orang lain di board (gate `ListByBigTask` sudah ada), jadi secara praktis
perubahan ini cuma benar-benar berdampak di Team Today (yang transparan,
semua orang bisa LIHAT semua orang, tapi tidak semua orang boleh EDIT).

### Frontend: sembunyikan tombol pensil kalau pasti ditolak
`team-today/+page.svelte` tambah `canEditEntry(ownerUserId)` -- cek client
side pakai `$auth.user.access_level`/`task_scope_visibility` (defense in
depth, backend tetap validasi ulang). Tombol pensil di suatu entry hanya
muncul kalau `ownerUserId === $auth.user.id` ATAU viewer
`access_level==='super_user'` atau `task_scope_visibility==='team'`.

### Bug tersembunyi ditemukan & diperbaiki: `GET /users/me` tidak return `task_scope_visibility`
Saat wiring pengecekan frontend di atas, ditemukan `user.loadMe()` query
SQL-nya TIDAK menyertakan kolom `task_scope_visibility` (kolom sudah ada
sejak migration `0026`, tapi `Scan()` di `loadMe()` melewatkannya) --
akibatnya field itu SELALU string kosong di response `/users/me`, walau
struct `User`/`Me` sudah mendeklarasikan field-nya. Ini bug independen
dari task ini (kemungkinan sudah ada sejak migration 0026), diperbaiki
sekalian karena Team Today butuh nilai field ini akurat di frontend untuk
gating tombol edit yang benar. `frontend/src/lib/stores/authStore.ts`
`UserSummary` juga ditambah field `task_scope_visibility` (sebelumnya tidak
ada sama sekali di tipe frontend).

## Alasan

- Extract ke fungsi murni `canEditDayEntry()` supaya bisa di-unit-test
  tanpa DB (wajib per CLAUDE.md untuk logic non-trivial) -- pola yang sama
  dengan `computeVerdict`/`computeExpectedPct` di package `bigtask`.
- Gate di backend (bukan cuma UI) karena ini aturan otorisasi sungguhan,
  bukan preferensi tampilan -- kalau cuma disembunyikan di frontend, user
  masih bisa PATCH langsung lewat curl/devtools.
- Gate di frontend JUGA ditambahkan (bukan cuma andalkan 403 backend)
  supaya UX-nya bersih -- tombol yang pasti gagal kalau diklik seharusnya
  tidak ditampilkan sama sekali, bukan menampilkan error setelah user
  klik.
- Pemilik entry SELALU boleh edit terlepas dari scope-nya sendiri --
  scope='self' itu soal "bisa lihat siapa", bukan "bisa edit punya
  sendiri" -- kalau pemilik entry tidak bisa edit entry-nya sendiri itu
  akan jadi bug UX yang jelas-jelas salah.

## Dampak / File Terpengaruh

- `backend/internal/dailytask/handler.go` -- fungsi baru
  `canEditDayEntry()`, gate permission di `UpdateDayEntry()` (query JOIN
  baru buat ambil pic_user_id + task_scope_visibility viewer).
- `backend/internal/dailytask/handler_test.go` -- `TestCanEditDayEntry`
  (7 test case: pemilik dengan berbagai scope, non-pemilik dengan
  berbagai kombinasi scope/super_user).
- `backend/internal/user/handler.go` -- fix bug `loadMe()` tidak query
  `task_scope_visibility` (independen dari task ini, ditemukan saat
  verifikasi).
- `frontend/src/lib/stores/authStore.ts` -- `UserSummary.task_scope_visibility`
  field baru.
- `frontend/src/routes/team-today/+page.svelte` -- `canSeeAllPeople`,
  `canEditEntry()`, gate tombol pensil di template.

## Verifikasi

- `go build ./...` -- sukses.
- `go test ./...` -- semua paket yang disentuh lulus (paket `board` tetap
  gagal karena bug pre-existing tidak terkait, sudah di-flag di
  decision-log-team-today-menu-20260901.md, belum diperbaiki, di luar
  scope task ini).
- `npm run check` -- 0 errors. `npm run test` -- 129/129 passed.
- Manual end-to-end via curl (3 skenario, semua sesuai ekspektasi):
  1. User scope='self', BUKAN pemilik entry -> PATCH /day-entries/{id} ->
     **403 Forbidden**, pesan "tidak punya akses untuk mengubah day entry
     ini".
  2. User scope='team', BUKAN pemilik entry -> PATCH -> **200 OK**, entry
     ter-update.
  3. User pemilik entry, scope='self' (kasus paling ketat) -> PATCH ->
     tetap **200 OK** (pemilik selalu boleh, terlepas dari scope-nya
     sendiri).
- `curl GET /users/me` sesudah fix `loadMe()` -- `task_scope_visibility`
  sekarang muncul benar di response (sebelumnya selalu kosong).
- Local Docker rebuild + container recreate untuk backend + nginx -- OK.

## Alternatif yang Ditolak

- **Gate hanya di frontend (sembunyikan tombol saja)**: ditolak -- bukan
  otorisasi sungguhan, gampang di-bypass lewat request langsung ke API.
- **Gate hanya di backend (tanpa sembunyikan tombol di UI)**: ditolak --
  UX buruk, user klik lalu dapat error 403 tanpa peringatan sebelumnya.
