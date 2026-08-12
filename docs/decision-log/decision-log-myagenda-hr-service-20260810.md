# Decision Log — Sub-Project `myagenda-service` (Push to HR, Fase 2)

**Tanggal:** 2026-08-10
**Konteks:** myagenda-hr-service

## Konteks/Masalah

`Push to HR` (Weekly Plan, Fase 6) selama ini cuma dicatat lokal di `weekly_push_log` — TIDAK ada HTTP call ke sistem HR eksternal beneran, didokumentasikan sebagai "Fase 2, di luar cakupan" (`01-vision-product.md` §4: "Fase 2 — Integrasi MyAgenda-HR. Push otomatis rollup mingguan ke sistem HR melalui service yang ditanam di server aplikasi HR"). User minta mulai bangun potongan Fase 2 ini: service kecil TERPISAH yang nerima push dan upsert ke tabel `my_agenda` — skema PERSIS DDL yang diberikan user (tabel MySQL asli sistem HR). Untuk sekarang cukup dummy/lokal (MySQL lokal), production-nya nanti dipasang langsung di server aplikasi HR (di luar repo ini).

## Keputusan

**Struktur**: sub-project baru `myagenda-service/` di root repo (sibling dari `backend/`/`frontend/`), Go module TERPISAH (`myagenda-service/go.mod`), docker-compose SENDIRI (`myagenda-service/docker-compose.yml`, isi: MySQL + service-nya) — TIDAK digabung ke `docker-compose(.dev).yml` milik rnd-task-board, karena production-nya akan dideploy terpisah total (di server HR, bukan di infrastruktur rnd-task-board). Stack: Go + `chi` (router) + `database/sql` + `go-sql-driver/mysql` — konsisten sama pola `backend/` (tanpa ORM, query manual), tanpa dependency berat.

**Skema `my_agenda`**: DIPAKAI PERSIS DDL yang diberikan user (migration SQL biasa via `golang-migrate` + image docker `migrate/migrate`, sama pola kayak `backend/db/migrations/`, TIDAK diubah sedikit pun — ini tabel MILIK sistem HR, bukan skema yang kita desain sendiri).

**Endpoint**: `POST /my-agenda` — upsert APLIKASI-LEVEL (bukan `ON DUPLICATE KEY` SQL, karena DDL yang diberikan TIDAK punya unique constraint apapun selain PK auto-increment) berdasarkan kombinasi `(user_id, judul_task, tgl_rencana)` sebagai identitas natural: `SELECT` dulu baris yang cocok, kalau ada → `UPDATE` (capaian/prosentase_capaian/due_date/uraian_task/target), kalau tidak ada → `INSERT` baru.

**Mapping field dari Weekly Plan Row kita → `my_agenda`**:
| `my_agenda` | Sumber |
|---|---|
| `user_id` | **PLACEHOLDER** — di-derive dari UUID user kita (CRC32 mod 1.000.000), BUKAN id HR asli (lihat Alasan) |
| `judul_task` | `big_task_name` |
| `tgl_rencana` | `week_start` (Senin minggu yang di-push) |
| `uraian_task` | `"{board_name} / {big_task_name}"` |
| `due_date` | `big_tasks.deadline` |
| `target` | `expected_pct` (double) |
| `capaian` | `actual_pct` (double) |
| `is_percentage` | selalu `true` (1) — semua metrik kita persentase |
| `prosentase_capaian` | `ROUND(actual_pct)` (int, mirror dari `capaian` — kemungkinan dipakai buat index/filter cepat tanpa float compare, konsisten sama key index yang ada di kolom itu) |

**Wiring ke backend utama**: `weeklyplan.Push` (backend/internal/weeklyplan) SETELAH upsert lokal `weekly_push_log` sukses, BEST-EFFORT HTTP POST ke `MYAGENDA_SERVICE_URL` (env var, kosong = fitur nonaktif/skip — default aman, tidak breaking kalau service ini belum jalan). Gagal connect/timeout **TIDAK membatalkan** response sukses ke user (cuma log error di server) — push lokal (`weekly_push_log`, sumber kebenaran utama Callback ID FR-WKL-05) tetap jadi source of truth utama; sinkron ke HR bersifat tambahan/best-effort, BUKAN dependency keras buat fitur inti jalan.

**Otorisasi**: TIDAK ADA auth di endpoint `myagenda-service` untuk sekarang (internal service-to-service, dipanggil server-ke-server bukan dari browser) — production nanti berjalan DI DALAM server aplikasi HR sendiri (network-internal), bukan exposed publik.

## Alasan

- **Sub-project + docker-compose terpisah, bukan digabung**: sesuai instruksi user eksplisit ("bikin aja sub project terpisah") DAN konsisten sama rencana deploy jangka panjang ("nanti productionnya tinggal deploy di environment app HR nya") — kalau digabung ke compose rnd-task-board, bakal ada kerja ekstra misahin lagi pas deploy beneran nanti.
- **Upsert app-level, bukan `ON DUPLICATE KEY`**: DDL yang dikasih literally tidak punya unique constraint di luar PK auto-increment — mengubah DDL (nambah unique index) DILARANG karena ini skema MILIK sistem HR eksisting, bukan yang kita desain (kalau di production nanti field ini beda struktur, harus ngikutin apa adanya, bukan versi yang udah kita modifikasi).
- **`user_id` PLACEHOLDER (CRC32 dari UUID), bukan mapping asli**: kita TIDAK PUNYA data id user di sistem HR asli (tidak ada tabel mapping, tidak ada field itu di `users` kita). Dummy/lokal tujuannya cuma buat validasi ALUR (wiring, format payload, upsert logic) — bukan validasi identitas user beneran. WAJIB diganti sebelum production nyata: butuh mapping asli `users.id` (UUID kita) → employee id HR (kemungkinan lewat kolom baru `users.hr_employee_id` yang diisi manual/dari HRIS, TAPI itu keputusan terpisah yang butuh input dari tim HR/System Analyst, di luar cakupan kerjaan dummy ini).
- **Best-effort, tidak blocking**: push-to-HR gagal (mis. service belum jalan, network drop) TIDAK BOLEH bikin fitur "Push" inti yang sudah jalan (`weekly_push_log`, Callback ID) ikut gagal — dua concern yang independen, kegagalan integrasi eksternal tidak boleh menjalar ke fitur internal yang sudah stabil.

## Dampak/File Terpengaruh

- `myagenda-service/` (baru, sub-project lengkap): `go.mod`, `cmd/api/main.go`, `internal/myagenda/handler.go`, `db/migrations/0001_create_my_agenda.up.sql`/`.down.sql`, `Dockerfile`, `Dockerfile.dev`, `.air.toml`, `docker-compose.yml`, `Makefile`, `README.md`.
- `backend/internal/weeklyplan/handler.go` — `Push` nambah best-effort HTTP call ke `myagenda-service`.
- `backend/cmd/api/main.go` — baca env var `MYAGENDA_SERVICE_URL` (opsional), teruskan ke `weeklyplan.NewHandler`.
- `docs/05-api-contract.md` §8 — catatan singkat integrasi HR + link ke `myagenda-service/README.md` buat detail kontrak service terpisah ini (bukan bagian dari kontrak API rnd-task-board sendiri).
