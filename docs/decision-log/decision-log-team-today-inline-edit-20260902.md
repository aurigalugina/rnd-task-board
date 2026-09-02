# Team Today: Inline Edit + Filter Per User

**Date:** 2026-09-02
**Status:** Implemented

## Konteks / Masalah

Menu Team Today (lihat `decision-log-team-today-menu-20260901.md`) awalnya
READ-ONLY -- cuma menampilkan Rencana/Realisasi/Progress/Blocker per orang
per tanggal, tanpa cara update dari situ. User harus pindah ke
board/big-task terkait lalu buka `DailyTaskPanel` buat update entry-nya.

User minta dua tambahan:
1. **Update Day Entry langsung dari Team Today** (Rencana, Realisasi,
   Progress %, Blocker) -- alasan eksplisit: kemudahan UX update, bukan
   kebutuhan data/permission baru.
2. **Filter berdasarkan user**, dengan **default per-user (diri sendiri)**
   -- bukan default "semua orang" seperti sebelumnya.

## Keputusan

### Inline edit
Reuse endpoint yang SAMA dengan `DailyTaskPanel.svelte`/
`DayEntryEditModal.svelte`: `PATCH /day-entries/{day_entry_id}` (backend
`dailytask.UpdateDayEntry`, TIDAK diubah -- endpoint ini sudah tidak
punya permission check spesifik-user, siapapun authenticated boleh PATCH
entry manapun, konsisten dengan behavior yang sudah ada). TIDAK ada
endpoint atau field backend baru -- ini murni UX shortcut di frontend,
bukan kemampuan baru di sistem.

UI: klik ikon pensil di satu entry -> entry itu berubah jadi form inline
(textarea Rencana, textarea Realisasi, `DayProgressStatus` component yang
sama dipakai `DailyTaskPanel` buat Progress, textarea Blocker kalau belum
100%) -- tombol Simpan/Batal di bawahnya. Cuma SATU entry bisa diedit
dalam satu waktu (state `editingId` tunggal) -- klik pensil di entry lain
otomatis membatalkan draft yang sedang jalan, mencegah kebingungan "mana
yang belum disimpan" kalau banyak entry dibuka sekaligus.

Setelah simpan sukses, entry di-update IN-PLACE (bukan re-fetch seluruh
`/team-today`) -- menghindari flicker dan reset scroll position yang akan
terjadi kalau reload seluruh daftar cuma buat satu entry.

### Filter per user
Dropdown baru di toolbar atas: **default = diri sendiri** (`$auth.user.id`),
opsi lain "Semua orang" (`''`, balik ke behavior lama) dan tiap nama
individual (dari `/users/assignable`, sumber sama dengan picker PIC/reviewer
di tempat lain). Filtering dilakukan CLIENT-SIDE (`users.filter(u =>
u.user_id === selectedUserId)`) terhadap response `/team-today` yang sudah
di-fetch -- TIDAK ada query param baru di backend, karena payload untuk
satu hari itu kecil (jumlah user x beberapa entry) dan tidak ada urgensi
performa untuk filter di server.

## Alasan

- Reuse endpoint PATCH yang sama menjaga SATU sumber kebenaran untuk
  logic update Day Entry (validasi progress_pct 0-100, kosongkan
  blocker_text otomatis saat 100%, dst) -- kalau dibikin endpoint
  terpisah, dua tempat itu bisa divergen seiring waktu.
- Default filter ke diri sendiri (bukan "semua orang" seperti awal)
  adalah permintaan eksplisit user -- Team Today sekarang punya DUA use
  case: (a) "update progress saya sendiri, cepat" (default) dan (b) "lihat
  semua orang lagi ngapain" (switch ke "Semua orang" atau pilih nama lain).
  Default ke (a) karena itu yang paling sering dipakai (update harian
  personal), (b) tetap 1 klik away.
- Single-entry-editing (bukan multi-edit) mengurangi risiko "typo lalu
  lupa disimpan di entry lain" -- pola yang sama dipakai `BigTaskList`/
  `DailyTaskPanel` (satu modal terbuka dalam satu waktu, bukan inline edit
  di banyak baris sekaligus).

## Dampak / File Terpengaruh

- `frontend/src/routes/team-today/+page.svelte` -- filter dropdown, state
  edit inline (`editingId`/draft fields), `startEdit()`/`cancelEdit()`/
  `saveEdit()`, reuse `DayProgressStatus` component.
- Tidak ada perubahan backend -- `PATCH /day-entries/{id}` dan
  `GET /team-today` dipakai apa adanya, tidak diubah.
- Tidak ada perubahan tipe (`TeamTodayEntry`/`TeamTodayUser` sudah cukup).

## Verifikasi

- `npm run check` -- 0 errors.
- `npm run test` -- 129/129 passed (tidak ada logic murni baru yang perlu
  test terpisah -- semua computed logic yang dipakai, seperti
  `progressBadge`/`DayProgressStatus`, sudah ada & sudah tertest
  sebelumnya di file lain).
- Manual: `curl` `PATCH /day-entries/{id}` (endpoint yang sama dipakai
  inline edit) dengan `planned_text`/`realisasi_text`/`progress_pct` --
  HTTP 200, response berisi nilai yang baru di-update dengan benar.
- Local Docker rebuild + container recreate untuk nginx -- OK, halaman
  serve normal.

## Alternatif yang Ditolak

- **Endpoint PATCH baru khusus Team Today**: tidak ada alasan bikin
  duplikat logic update yang sudah ada dan sudah benar di
  `dailytask.UpdateDayEntry` -- reuse lebih aman.
- **Filter default "Semua orang"**: bertentangan dengan permintaan
  eksplisit user (default per-user).
- **Multi-entry edit bersamaan** (semua entry langsung jadi form tanpa
  perlu klik pensil): lebih ramai secara visual dan berisiko "lupa
  disimpan" di banyak tempat sekaligus -- klik-untuk-edit satu per satu
  lebih aman & konsisten dengan pola modal-tunggal yang sudah dipakai di
  seluruh aplikasi.
