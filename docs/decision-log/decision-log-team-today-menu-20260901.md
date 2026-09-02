# Menu "Team Today" + Kolom Realisasi di Day Entry

**Date:** 2026-09-01
**Status:** Implemented

## Konteks / Masalah

User (Lugi) ingin memonitor tim dengan POV "hari" -- "hari ini tim gue lagi
ngapain aja". Halaman yang ada tidak cocok untuk kebutuhan ini:
- **Dashboard**: POV per board/project (agregasi Big Task), bukan per orang.
- **Weekly Plan**: POV per daily task per MINGGU (buat push ke HR), bukan
  per hari, dan default-nya cuma diri sendiri (super_user bisa override
  `?as_user_id=` tapi satu orang per kali lihat, bukan semua orang
  sekaligus).

Terpisah tapi terkait: user juga minta tambahan kolom di Day Entry buat
mencatat REALISASI (apa yang benar-benar dikerjakan), terpisah dari kolom
"Rencana" (`planned_text`) yang sudah ada -- field ini jadi salah satu info
utama yang ditampilkan di menu Team Today baru.

## Keputusan

### 1. Kolom `realisasi_text` baru di `day_entries`
Migration `0027_day_entries_realisasi_text` -- `TEXT NOT NULL DEFAULT ''`,
sejajar dengan `planned_text` (Rencana) yang sudah ada. Bisa diedit lewat
`PATCH /day-entries/{id}` (field baru di `updateDayEntryRequest`, opsional
sama seperti field lain). Ditampilkan di:
- `DayEntryEditModal.svelte` -- textarea baru di bawah Rencana, HANYA saat
  edit entry existing (`entry !== null`) -- entry baru dibuat kosong dulu,
  realisasi diisi belakangan.
- `DailyTaskPanel.svelte` -- kolom baru di tabel Day Entry per Daily Task.

### 2. Menu baru "Team Today" (`/team-today`)
Endpoint `GET /team-today?date=YYYY-MM-DD` (default hari ini) di
`dailytask` package (bukan package baru -- scope query-nya emang soal Day
Entry lintas board/user, paling natural di situ). Response: array per user
(`TeamTodayUser`), masing-masing berisi `entries: TeamTodayEntry[]` (Day
Entry milik user itu di tanggal terpilih, sudah include board/big
task/daily task name).

**Keputusan eksplisit dari 3 pertanyaan clarify ke user:**
1. **Scope:** Build langsung (bukan cuma rekomendasi).
2. **Akses:** SEMUA user bisa lihat SEMUA orang -- transparan, sama seperti
   filosofi Dashboard. TIDAK di-scope `task_scope_visibility` seperti
   endpoint daily-task lain (`ListByBigTask`) -- keputusan sadar, beda dari
   pola existing.
3. **Pengelompokan:** Per ORANG (list nama, di bawahnya task-task hari itu)
   -- bukan per board/project.

User TANPA entry di tanggal terpilih tetap muncul di daftar (`entries: []`)
supaya jelas siapa yang belum update, bukan diam-diam hilang.

Frontend: halaman baru `frontend/src/routes/team-today/+page.svelte` --
navigasi hari (prev/next + "kembali ke hari ini", pola sama seperti
week-nav di Weekly Plan tapi per hari), satu card per user berisi
avatar+nama+jumlah task, lalu list entry (board, big task, daily task,
Rencana, Realisasi, Blocker kalau ada, badge status). Menu ditambahkan ke
nav utama (`+layout.svelte`) di antara Boards dan My Weekly Plan.

## Alasan

- Endpoint terpisah (bukan reuse `/weekly-plan` dengan date range 1 hari)
  karena granularitas & filosofi akses beda total: Weekly Plan itu "punya
  saya, buat push ke HR", Team Today itu "punya semua orang, buat
  monitoring transparan" -- maksa reuse endpoint yang sama bakal
  memperumit permission logic yang sudah ada.
- POV per orang (bukan per board) dipilih USER secara eksplisit -- cocok
  untuk pertanyaan "si A lagi ngapain" yang jadi motivasi awal.
- Tidak di-scope `task_scope_visibility` karena user secara eksplisit minta
  transparan buat semua orang, beda dari pola akses per-Big-Task yang sudah
  ada (yang memang didesain restriktif untuk kasus lain).

## Dampak / File Terpengaruh

- `backend/db/migrations/0027_day_entries_realisasi_text.{up,down}.sql`
- `backend/internal/dailytask/handler.go` -- `DayEntry.RealisasiText`,
  `loadDays()`, `UpdateDayEntry()`/`updateDayEntryRequest`, `AddDayEntry()`
  (RETURNING clause), handler baru `TeamToday()` + tipe
  `TeamTodayEntry`/`TeamTodayUser`.
- `backend/cmd/api/main.go` -- route baru `GET /team-today`.
- `frontend/src/lib/types.ts` -- `DayEntry.realisasi_text`,
  `TeamTodayEntry`, `TeamTodayUser`.
- `frontend/src/lib/components/DayEntryEditModal.svelte` -- field Realisasi.
- `frontend/src/lib/components/DailyTaskPanel.svelte` -- kolom Realisasi di
  tabel Day Entry.
- `frontend/src/routes/team-today/+page.svelte` -- halaman baru.
- `frontend/src/routes/+layout.svelte` -- nav link baru.

## Verifikasi

- `go build ./...`, `go test ./...` -- semua paket yang disentuh lulus
  (catatan: paket `board` gagal `go vet` karena bug pre-existing di
  `handler.go:152`, TIDAK terkait perubahan ini, tidak diperbaiki di sesi
  ini -- di luar scope, sudah di-flag ke user).
- `npm run check` -- 0 errors (perlu `npx svelte-kit sync` sekali untuk
  regenerate route types setelah nambah folder route baru).
- `npm run test` -- 129/129 passed.
- Manual: migration applied ke DB lokal (kolom `realisasi_text` ada),
  `GET /team-today` return 200 dengan bentuk JSON yang benar,
  `PATCH /day-entries/{id}` dengan `realisasi_text` tersimpan & ke-return
  dengan benar.
- Build & restart Docker lokal (backend + nginx recreate, bukan cuma
  restart -- image baru butuh container baru) -- OK.

## Catatan Lain (Ditemukan, Bukan Diperbaiki)

`backend/internal/board/handler.go:152` punya `fmt.Sprintf` dengan argumen
mismatch (`go vet` fail) -- pre-existing, tidak disentuh perubahan ini,
gagal saat `go test ./...` di package `board` (build failed karena vet).
Perlu diperbaiki di sesi terpisah.
