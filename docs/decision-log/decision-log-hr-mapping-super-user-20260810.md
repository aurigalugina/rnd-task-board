# Decision Log — Mapping User HR, Referensi Tim, dan Role `access_level` (super_user)

**Tanggal:** 2026-08-10
**Konteks:** hr-mapping-super-user

## Konteks/Masalah

Integrasi HR (`myagenda-service`, lihat `decision-log-myagenda-hr-service-20260810.md`) masih pakai `user_id` PLACEHOLDER (derive CRC32 dari UUID kita) karena belum ada mapping ke id pegawai HR asli. User kasih data mapping asli (`user_id|email|nip|nama_lengkap`, ~70 baris) dan minta tiga hal terkait:

1. Tabel referensi `referensi_user_hr` dari data itu, dipakai buat mapping di Manajemen User (bukan placeholder lagi).
2. Tabel referensi Tim/Org (`referensi_tim`) — field "Tim/Org" yang saat ini free-text di form user jadi selection dari referensi ini.
3. `role_code` baru: `super_user` / `regular_user` — super_user bisa lihat SEMUA data, mulai dari My Weekly Plan (lihat weekly plan siapapun, bantu push atas nama orang lain, cek status push tim).

## Keputusan

**`users.access_level`** — kolom baru (`TEXT CHECK IN ('super_user','regular_user') DEFAULT 'regular_user'`), BUKAN lewat sistem `roles`/`user_roles` many-to-many yang sudah ada. Ini PENYIMPANGAN SADAR dari aturan "role selalu many-to-many, jangan kolom tunggal" yang sudah dipegang project ini — user diberi pilihan dan secara eksplisit pilih kolom terpisah (bukan rekomendasi default). Alasan konsep ini beda dari `roles` (spv/dev/qa/dst): `roles` = fungsi kerja yang BISA dirangkap (satu orang bisa spv+dev), `access_level` = tingkat visibilitas data yang SALING EKSKLUSIF (satu user cuma salah satu). Dimasukkan ke JWT claims (`access_level`) di samping `roles`, supaya endpoint & frontend bisa cek tanpa query DB ulang tiap request.

**`referensi_tim`** — tabel baru `(id uuid pk, name text unique)`, seed `'R&D'` (satu-satunya nilai yang sudah pernah dipakai). `users.org_team` TETAP kolom text apa adanya (TIDAK diubah jadi FK) — supaya tidak menyentuh bentuk JWT claims/response API yang sudah ada di banyak tempat (`auth.Login`, `user.List`, dst); validasi cukup di level aplikasi (`org_team` harus ada di `referensi_tim.name` saat create/update user). Frontend: dropdown sumbernya `GET /referensi-tim`, dengan opsi tambah nama tim baru langsung dari situ (create ringan, admin-only) — bukan CRUD penuh (belum butuh edit/hapus).

**`referensi_user_hr`** — tabel baru `(hr_user_id int pk, email text, nip text, nama_lengkap text)`, di-seed PERSIS dari data yang diberikan (migration, bukan dari UI — ini data MILIK sistem HR, bukan yang kita kelola). TIDAK ADA CRUD UI untuk tabel ini — kalau daftar pegawai HR berubah, perlu migration baru (di luar cakupan sekarang; dicatat sebagai limitasi).

**`users.hr_user_id`** — kolom baru, nullable, `UNIQUE REFERENCES referensi_user_hr(hr_user_id)` (satu user rnd ↔ satu pegawai HR, dan sebaliknya). Di-set lewat Manajemen User (dropdown cari dari `referensi_user_hr`, admin-only) — SEKARANG ADA JUGA kemampuan EDIT user yang SUDAH ADA (`PATCH /users/{id}`, sebelumnya tidak ada endpoint ini, cuma `POST /users` buat user baru). `pushToMyAgenda` (di `weeklyplan.Push`) SEKARANG pakai `hr_user_id` beneran kalau sudah di-mapping; fallback ke placeholder CRC32 LAMA (dengan log warning) kalau user belum di-mapping — supaya push tetap jalan (best-effort) sambil nunggu mapping lengkap.

**Super User — My Weekly Plan**:
- `GET /weekly-plan?week_start=...&as_user_id=<uuid>` — param baru, CUMA dihormati kalau requesting user `access_level=super_user`; kalau `regular_user` yang kirim param ini, ditolak 403 (bukan diabaikan diam-diam — defensif, gampang dites, gak menyembunyikan percobaan akses gak sah). Filter tetap sama (`daily_tasks.pic_user_id`), cuma target-nya `as_user_id` bukan requester.
- `POST /weekly-plan/push` — sama, terima `as_user_id` opsional (super_user only). `weekly_push_log.pushed_by` TETAP diisi actor (super_user yang mengklik), BUKAN target user — buat audit "siapa yang benar-benar melakukan push". Payload ke `myagenda-service` pakai HR mapping milik TARGET user (`as_user_id`), karena laporan itu punya orang tersebut, bukan punya super_user-nya.
- `GET /weekly-plan/team-status?week_start=...` (endpoint baru, super_user only) — satu baris per user (SEMUA user, termasuk yang 0 Big Task minggu ini), `total_big_tasks`/`pushed_big_tasks`/`all_pushed` — buat kebutuhan "cek tim udah push atau belum" TANPA harus buka drill-down satu-satu.
- Frontend: dropdown "Lihat sebagai" di halaman Weekly Plan (cuma muncul kalau `access_level=super_user`) + tabel ringkasan status push tim di bawahnya.

## Alasan

- **`access_level` kolom tunggal (bukan role many-to-many)**: keputusan eksplisit user, didokumentasikan di sini sebagai pengecualian sadar terhadap konvensi arsitektur yang ada — supaya sesi berikutnya paham ini BUKAN kekhilafan, tapi keputusan yang sudah dipertimbangkan dan disetujui.
- **`org_team` tetap free-text + validasi aplikasi, bukan FK**: mengubah jadi FK akan mengubah bentuk JOIN di `auth.Login` (JWT claim) dan semua endpoint yang balikin `org_team` — risiko regresi luas untuk manfaat yang bisa dicapai lebih murah (validasi + dropdown sumber data, tanpa migrasi bentuk data).
- **`referensi_user_hr` read-only (seed-only, tanpa CRUD)**: data ini representasi sistem HR eksternal — kalau dikasih CRUD, ada risiko data kita "menyimpang" dari sumber HR asli tanpa sadar. Update resmi harus lewat proses re-export/migration baru dari HR, bukan diedit bebas dari UI kita.
- **`403` (bukan diabaikan diam-diam) kalau `regular_user` kirim `as_user_id`**: prinsip fail-loud untuk percobaan akses yang tidak sah — kalau diabaikan diam-diam, bug di frontend (misalnya kirim param itu gak sengaja) bisa lolos tanpa ketahuan, dan sulit ditest secara eksplisit.
- **`pushed_by` = actor, bukan target**: audit trail harus mencatat SIAPA yang benar-benar menekan tombol, bukan siapa yang "diwakili" — kalau ada push yang salah/perlu ditelusuri, harus jelas siapa yang melakukannya.

## Dampak/File Terpengaruh

- `backend/db/migrations/0012_create_referensi_tim.up/down.sql`, `0013_create_referensi_user_hr.up/down.sql`, `0014_add_users_hr_access_level.up/down.sql` (baru).
- `backend/internal/referensi/handler.go` (baru) — `GET/POST /referensi-tim`, `GET /referensi-user-hr`.
- `backend/internal/auth/auth.go` — `access_level` masuk JWT claims (Login/Refresh), `AccessLevelFromContext`.
- `backend/internal/user/handler.go` — `List`/`Create` include `access_level`/`hr_user_id`; handler baru `Update` (`PATCH /users/{id}`, admin-only) buat edit user existing.
- `backend/internal/weeklyplan/handler.go` — `List`/`Push` terima `as_user_id` (super_user only, 403 kalau bukan), handler baru `TeamStatus`; `pushToMyAgenda` pakai `hr_user_id` asli kalau ada.
- `backend/cmd/api/main.go` — route baru.
- `frontend/src/lib/types.ts`, `SettingsModal.svelte` (dropdown tim + HR mapping + access_level, edit user existing), `routes/weekly-plan/+page.svelte` (dropdown lihat-sebagai + tabel status tim).
- `docs/05-api-contract.md`, `docs/06-db-design.md` — dokumentasi endpoint & skema baru.
