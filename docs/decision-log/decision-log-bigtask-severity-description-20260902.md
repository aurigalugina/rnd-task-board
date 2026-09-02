# Big Task: Tambah Severity (Critical/High/Medium/Low) + Deskripsi

**Date:** 2026-09-02
**Status:** Implemented

## Konteks / Masalah

User minta dua tambahan pada Big Task:
1. Flag **severity** dengan 4 level tetap: critical, high, medium, low.
2. Field **deskripsi** bebas teks, ditampilkan **di bagian atas** panel
   detail Big Task (di bawah title bar, sebelum daftar anggota tim/daily
   task).

Tidak ada di BRD/SRS -- keputusan produk baru murni permintaan user,
bukan requirement lama yang terlewat.

## Keputusan

### Skema DB (migration 0028)
```sql
ALTER TABLE big_tasks ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE big_tasks ADD COLUMN severity TEXT NOT NULL DEFAULT 'medium'
    CHECK (severity IN ('critical', 'high', 'medium', 'low'));
```
- `description`: TEXT bebas, default string kosong (opsional, tidak wajib
  diisi saat create).
- `severity`: TEXT dengan CHECK constraint di level DB (bukan enum
  Postgres asli -- konsisten dengan pola `access_level`/
  `task_scope_visibility` yang sudah ada di tabel `users`, semuanya TEXT +
  CHECK, bukan `CREATE TYPE ... AS ENUM`). Default `'medium'` -- Big Task
  lama (sebelum migration ini) otomatis dianggap medium, bukan NULL/error.

### Backend (`bigtask/handler.go`)
- `isValidSeverity(s string) bool` -- fungsi murni validasi terhadap 4
  literal yang sama dengan CHECK constraint DB (defense in depth: kalau
  someday migration CHECK-nya lupa di-apply di server tertentu, validasi
  Go tetap menjaga integritas data).
- `BigTask` struct dapat field `Description`/`Severity`.
- `ListByBoard`/`loadBigTask` (dipakai `Create`/`SignOff`/`Update`/
  `SetMembers` supaya response konsisten) query kedua kolom baru.
- `Create` (POST): `severity` opsional, default `"medium"` kalau kosong;
  ditolak 400 kalau diisi tapi bukan salah satu dari 4 literal valid.
  `description` opsional, tidak divalidasi (bebas teks apa saja termasuk
  kosong).
- `Update` (PATCH, **super_user only** -- endpoint ini SUDAH gated
  super_user dari awal untuk semua field termasuk name/start_date/
  deadline, jadi description/severity otomatis ikut aturan yang sama,
  tidak butuh gate terpisah): `description`/`severity` sebagai
  `*string` (pointer, pola sama dengan field lain di endpoint ini) --
  `nil` = tidak diubah, `""` (empty string via pointer) = boleh dikosongkan
  eksplisit untuk description, severity tetap divalidasi kalau diisi.

### Frontend
- `types.ts`: `Severity` type union baru, `BigTask.description`/`severity`.
- `BigTaskList.svelte`:
  - Form create & edit Big Task dapat 2 field baru: textarea Deskripsi,
    dropdown Severity (4 opsi, default Medium).
  - **Deskripsi ditampilkan di bagian PALING ATAS** panel detail (`<p
    class="bigtask-description">`), tepat di bawah title bar (nama +
    verdict badge + tombol aksi), SEBELUM baris "Anggota tim" -- sesuai
    permintaan eksplisit user "deskripsinya munculkan di bagian atas".
    Hanya dirender kalau tidak kosong (`{#if activeBt.description}`).
  - Badge severity ditampilkan di title bar detail (sebelah verdict
    badge), plus dot kecil berwarna di daftar Big Task kiri (HANYA untuk
    critical/high -- medium/low tidak perlu penanda visual tambahan
    karena itu level "normal", menghindari noise visual kalau semua Big
    Task punya dot).
  - Warna severity: critical = merah (`--win-red`), high = amber
    (`--win-amber`), medium = biru (`--win-blue`, warna default/netral),
    low = abu-abu (`--text-muted`) -- pola konsisten dengan skema warna
    badge lain yang sudah ada (`.badge-good`/`.badge-bad`/dst), otomatis
    ikut 8 tema yang ada karena pakai CSS custom properties.

## Alasan

- CHECK constraint di DB (bukan cuma validasi aplikasi) supaya data tetap
  konsisten walau ada jalur insert lain di masa depan (migrasi data,
  script admin, dst) yang mungkin lupa validasi di layer Go.
- Default `medium` (bukan NULL atau wajib diisi) supaya Big Task existing
  tidak perlu migrasi data manual dan form create tidak MEMAKSA user pilih
  severity kalau belum tahu -- medium adalah pilihan paling aman/netral.
- Dot indicator cuma untuk critical/high (bukan render badge penuh di list
  kiri) -- daftar Big Task sudah cukup padat (nama + verdict badge + hold
  badge), badge severity penuh di situ akan berlebihan; tujuannya cuma
  "tarik perhatian ke yang urgent", bukan menampilkan detail penuh (detail
  lengkap ada begitu diklik).
- Deskripsi taruh di ATAS (bukan di bawah/collapsed) sesuai instruksi
  eksplisit user -- ini konteks penting yang harus langsung terlihat
  begitu Big Task dibuka, bukan informasi sekunder.

## Dampak / File Terpengaruh

- `backend/db/migrations/0028_big_task_description_severity.{up,down}.sql`
- `backend/internal/bigtask/handler.go` -- `BigTask` struct,
  `isValidSeverity()`, `validSeverities`, `ListByBoard`/`loadBigTask`
  (query+scan), `createBigTaskRequest`/`Create`, `updateBigTaskRequest`/
  `Update` (SQL UPDATE statement).
- `backend/internal/bigtask/handler_test.go` -- `TestIsValidSeverity`
  (7 kasus: 4 valid, empty string, case-sensitivity, nilai sembarang).
- `frontend/src/lib/types.ts` -- `Severity` type, `BigTask.description`/
  `severity`.
- `frontend/src/lib/components/BigTaskList.svelte` -- state form create
  (`description`/`severity`) & edit (`editDescription`/`editSeverity`),
  `severityBadge()` helper, template create/edit form + tampilan
  deskripsi & badge severity di detail panel + dot di list.
- `frontend/src/app.css` -- `.severity-critical/high/medium/low`,
  `.severity-dot`/`.severity-dot-critical/high`, `.bigtask-description`.
- `frontend/src/lib/dashboardStats.test.ts` -- fixture `fakeBigTask()`
  diupdate menyertakan field baru (wajib, `BigTask` tidak lagi optional
  untuk field ini).

## Verifikasi

- `go build ./...` -- sukses. `go test ./...` -- semua paket yang
  disentuh lulus (paket `board` tetap gagal karena bug pre-existing tidak
  terkait, di-flag sebelumnya, di luar scope task ini).
- `npm run check` -- 0 errors. `npm run test` -- 129/129 passed.
- Manual end-to-end via curl:
  1. `POST /boards/{id}/big-tasks` dengan `severity:"critical"` +
     `description` -- 201, `GET` balikin kedua field dengan benar.
  2. `POST` dengan `severity:"urgent"` (invalid) -- **400**, pesan jelas.
  3. `PATCH /big-tasks/{id}` update `severity`+`description` -- 200,
     nilai baru ter-refleksi di response.
- Migration `0028` applied ke DB lokal -- kolom + CHECK constraint
  terverifikasi via `\d big_tasks`.
- Local Docker rebuild + container recreate untuk backend + nginx -- OK.

## Alternatif yang Ditolak

- **Postgres native ENUM type** untuk severity: ditolak, tidak konsisten
  dengan pola TEXT+CHECK yang sudah dipakai kolom serupa lain di skema
  ini (`access_level`, `task_scope_visibility`) -- native ENUM lebih sulit
  di-ALTER (tambah/hapus value) dibanding CHECK constraint biasa.
- **Severity wajib diisi tanpa default**: ditolak -- akan memaksa migrasi
  data manual untuk semua Big Task existing dan menambah friksi form
  create tanpa manfaat jelas.
