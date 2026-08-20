# API Contract — R&D Ops

**Status:** Draft v1
**Referensi:** `03-srs.md`, `04-architecture.md`, `06-db-design.md`
**Base path:** `/api/v1`
**Format:** JSON, autentikasi via `Authorization: Bearer <JWT>` kecuali disebutkan lain.

---

## 1. Konvensi Umum

- Seluruh identitas entitas berbentuk UUID string.
- Tanggal berformat `YYYY-MM-DD`, timestamp berformat ISO 8601 (`YYYY-MM-DDTHH:mm:ssZ`).
- Field turunan (`actual_pct`, `expected_pct`, `verdict`, `is_weekend`, `completion_rate`) selalu dihitung server-side saat response dibentuk — tidak pernah diterima sebagai input dari klien.
- Response error mengikuti format:
```json
{ "error": { "code": "string", "message": "string" } }
```
- Paginasi (bila diperlukan pada daftar besar) memakai `?limit=&cursor=`, response menyertakan `next_cursor`.

## 2. Auth

### `POST /auth/login`
Request: `{ "email": "string", "password": "string" }`
Response 200: `{ "access_token": "string", "user": UserSummary }`
Refresh token dikirim sebagai HTTP-only cookie, tidak dalam body.

### `POST /auth/logout`
Menghapus refresh token cookie. Response 204.

### `POST /auth/refresh`
Menggunakan cookie refresh token. Response 200: `{ "access_token": "string" }`

`UserSummary`:
```json
{
  "id": "uuid",
  "display_name": "string",
  "initials": "string",
  "roles": ["dev", "spv"],
  "org_team": "R&D"
}
```

## 3. Boards

### `GET /boards`
Query param opsional: `?category=project|routine`, `?team_id=uuid` (**`team_id` cuma dihormati kalau caller `access_level=super_user`** — regular user diabaikan filternya karena sudah dibatasi otomatis, lihat catatan visibility di bawah).
Response 200: `[{ "id", "name", "description", "category": "project"|"routine"|null, "team_ids": ["uuid"] }]`. Board yang sudah di-archive TIDAK ikut muncul (`WHERE archived_at IS NULL`) — lihat §3.1.
**Visibility (2026-08-20):** regular user (`access_level=regular_user`) HANYA melihat board yang di-assign ke tim (`org_team`) dia sendiri via `board_teams` (join `referensi_tim.name = users.org_team`) — board bisa lintas tim, tapi user cuma lihat board yang timnya termasuk. `access_level=super_user` tidak dibatasi ini, boleh pakai `?team_id=` buat lihat tim manapun. Lihat `docs/decision-log/decision-log-boards-dashboard-enhancements-20260820.md`.

### `POST /boards`
Request: `{ "name": "string", "description": "string", "category": "project"|"routine"|null }` (`category` opsional, boleh diisi user manapun saat create).
Server otomatis nambah baris `board_teams` untuk `org_team` milik pembuat board (board "kepunyaan" tim pembuatnya secara default).
Response 201: Board object.

### `PATCH /boards/{board_id}` (super_user only, baru 2026-08-20)
Request (parsial): `{ "description": "string", "category": "project"|"routine"|null, "team_ids": ["uuid"] }` — kirim `team_ids` untuk REPLACE penuh set tim board itu (bukan append). Nama board TIDAK bisa diedit lewat endpoint ini.
Otorisasi: `access_level=super_user` (cek in-handler, bukan `RequireRole`) — 403 kalau bukan.
Response 200: Board object terbaru.

### `GET /boards/{board_id}/summary`
Response 200 — matriks dashboard per board:
```json
{
  "board_id": "uuid",
  "total_big_tasks": 8,
  "not_started": 1,
  "running": 5,
  "done": 2,
  "on_hold": 1,
  "won": 2,
  "lost": 1,
  "completion_rate": 62,
  "project_status": "in_progress"
}
```
`project_status` bernilai `"done"` hanya jika seluruh Big Task pada board tersebut ber-status signed (lihat SRS FR-BRD-07).

### 3.1 Board Archive (super_user only)

Lihat `docs/decision-log/decision-log-board-archive-20260812.md`. Board yang diarsipkan hilang dari `GET /boards` (Dashboard + tab Boards), tapi TETAP muncul apa adanya di `GET /weekly-plan` dan `GET /review-queue` (query di situ tidak difilter archived — laporan personal & antrean review tidak boleh "kehilangan" riwayat cuma karena project-nya diarsipkan).

Ketiga endpoint di bawah CUMA bisa diakses `access_level=super_user` — 403 kalau bukan (dicek in-handler, bukan lewat middleware role).

#### `GET /boards/archive`
Response 200: `[{ "id", "name", "description", "archived_at", "archived_by_name" }]`, urut `archived_at DESC`.

#### `PATCH /boards/{board_id}/archive`
Response 204. 409 kalau board tidak ditemukan atau sudah diarsipkan.

#### `PATCH /boards/{board_id}/unarchive`
Response 204. 409 kalau board tidak ditemukan atau belum diarsipkan.

## 4. Big Tasks

### `GET /boards/{board_id}/big-tasks`
Response 200:
```json
[
  {
    "id": "uuid",
    "board_id": "uuid",
    "name": "string",
    "start_date": "2026-08-01",
    "deadline": "2026-08-15",
    "default_pic_user_id": "uuid",
    "on_hold": false,
    "actual_pct": 62,
    "expected_pct": 70,
    "days_left": 4,
    "verdict": "on_progress",
    "signed": false,
    "signed_by": null,
    "signed_at": null,
    "reviewer_user_ids": ["uuid"]
  }
]
```
`verdict` ∈ `{ "on_progress", "win", "lose" }` — dihitung sesuai RULE-04/05/06 (BRD).
`reviewer_user_ids` — daftar user (bisa lebih dari satu, bisa kosong) yang jadi reviewer Daily Task di bawah Big Task ini lewat Review Queue (§9). Lihat `docs/decision-log/decision-log-bigtask-reviewer-assignment-20260810.md`.

### `POST /boards/{board_id}/big-tasks`
Request: `{ "name", "start_date", "deadline", "default_pic_user_id", "reviewer_user_ids": ["uuid"] }` (`reviewer_user_ids` opsional, default `[]`).
Response 201: `{ "id": "uuid" }`.

### `POST /big-tasks/{big_task_id}/sign-off`
Otorisasi: role `spv`. Ditolak (409) apabila `actual_pct` < 100.
Request (opsional, baru 2026-08-20): `{ "signed_at": "YYYY-MM-DD" }` — kalau diisi (backdate manual), CUMA boleh kalau caller juga `access_level=super_user` (403 kalau non-super_user isi field ini). Divalidasi 400 kalau: tanggal di masa depan, sebelum `start_date` Big Task, atau sebelum tanggal `day_entries` terakhir Big Task itu. Kalau `signed_at` diisi, `big_task_signoffs.signed_at_backdated_by` diisi user_id super_user itu (audit trail, dipakai frontend buat indikator "(backdated)").
Response 200: Big Task object dengan `signed: true`.

### `DELETE /big-tasks/{big_task_id}/sign-off`
Otorisasi: role `spv`. Undo sign-off.
Response 204.

### `PATCH /big-tasks/{big_task_id}` (super_user only, baru 2026-08-20)
Request (parsial): `{ "name": "string", "start_date": "YYYY-MM-DD", "deadline": "YYYY-MM-DD" }`.
Otorisasi: `access_level=super_user` (cek in-handler) — 403 kalau bukan. Set `big_tasks.updated_by`.
Response 200: Big Task object terbaru. Lihat `docs/decision-log/decision-log-verdict-backfill-signoff-20260820.md` — dipakai bareng backdate sign-off (mis. geser deadline dulu sebelum sign-off retroaktif).

**Catatan verdict (2026-08-20):** `verdict` untuk Big Task yang sudah `signed` dihitung dari `deadline` vs `signed_at` (BEKU sejak titik sign-off) — BUKAN dari `deadline` vs waktu request berjalan. Big Task yang di-sign-off tepat waktu tetap `"win"` selamanya walau dibaca ulang lama setelah deadline lewat.

## 5. Daily Tasks & Day Entries

### `GET /big-tasks/{big_task_id}/daily-tasks`
Response 200:
```json
[
  {
    "id": "uuid",
    "big_task_id": "uuid",
    "title": "string",
    "pic_user_id": "uuid",
    "start_date": "2026-08-05",
    "end_date": "2026-08-07",
    "actual_pct": 66,
    "days": [
      {
        "id": "uuid",
        "entry_date": "2026-08-05",
        "planned_text": "string",
        "progress_pct": 65,
        "blocker_text": "",
        "is_weekend": false
      }
    ]
  }
]
```
`progress_pct` (0-100) — `0` = Belum, `1-99` = On Progress, `100` = Selesai (turunan murni, bukan field tersimpan terpisah). `actual_pct` Daily Task = rata-rata `progress_pct` semua `days`-nya. Menggantikan `is_done` boolean lama — lihat `docs/decision-log/decision-log-day-entry-progress-pct-20260810.md`.

### `POST /big-tasks/{big_task_id}/daily-tasks`
Request: `{ "title", "pic_user_id", "start_date", "end_date" }`
Server menghasilkan satu `day_entries` per tanggal kalender dalam rentang (FR-DLY-02).
Response 201: Daily Task object (bentuk sama seperti GET, `days` terisi entri kosong).

### `POST /daily-tasks/{daily_task_id}/clone-review`
Request: `{ "role_tag": "SPV" | "QA", "start_date", "end_date" }`
Server membuat Daily Task baru pada Big Task yang sama dengan judul `"[Review {role_tag}] {judul asal}"` dan `pic_user_id` default = user PERTAMA (urut `display_name` ASC) yang punya role `spv`/`qa` sesuai `role_tag` (FR-DLY-07). Ditolak 400 kalau tidak ada user dengan role tersebut. Lihat `docs/decision-log/decision-log-clone-review-20260809.md` untuk detail & alasan (termasuk kenapa ini endpoint sungguhan, bukan cuma prefill form klien seperti di mockup demo).
Response 201: Daily Task object baru (bentuk sama seperti `POST /big-tasks/{big_task_id}/daily-tasks`).

### `PATCH /day-entries/{day_entry_id}`
Request (parsial, kirim field yang berubah saja):
```json
{ "planned_text": "string", "progress_pct": 65, "blocker_text": "string" }
```
`progress_pct` ditolak 400 kalau di luar rentang 0-100.
Response 200: Day Entry object terbaru (termasuk `id` — dibutuhkan klien karena inilah satu-satunya cara mendapat `day_entry_id` untuk PATCH berikutnya, lihat `id` di response `GET .../daily-tasks` §5).
Catatan: saat `progress_pct` diset `100` (Selesai), klien direkomendasikan mengirim `blocker_text: ""` dalam payload yang sama (selaras perilaku mockup: menandai selesai mengosongkan blocker).

### `POST /daily-tasks/{daily_task_id}/day-entries`
Request: `{ "entry_date": "2026-08-10", "planned_text": "string" }` (`planned_text` opsional, default `""`).
Response 201: Day Entry object baru (`progress_pct: 0`, `blocker_text: ""`).
Nambah SATU baris day_entries manual di luar generate otomatis — dipakai kalau 1 tanggal mau di-breakdown jadi lebih dari satu task/rencana (`day_entries` TIDAK LAGI dibatasi satu baris per tanggal, lihat `docs/decision-log/decision-log-day-entry-add-delete-20260810.md`). `entry_date` tidak divalidasi terhadap rentang `start_date`/`end_date` Daily Task (sengaja, lihat decision log).

### `DELETE /day-entries/{day_entry_id}`
Hapus permanen satu baris day_entries (mis. hari weekend yang ke-generate otomatis tapi PIC tidak mau lembur). `actual_pct` Daily Task otomatis ikut menyesuaikan (dihitung dari SEMUA `day_entries` yang tersisa, bukan disimpan terpisah).
Response 204.

## 6. Comments

### `GET /big-tasks/{big_task_id}/comments?scope=all|general|{daily_task_id}`
Response 200:
```json
[
  {
    "id": "uuid",
    "big_task_id": "uuid",
    "daily_task_id": "uuid|null",
    "author_id": "uuid",
    "body": "string",
    "created_at": "timestamp"
  }
]
```

### `POST /big-tasks/{big_task_id}/comments`
Request: `{ "daily_task_id": "uuid|null", "body": "string" }`
`author_id` diambil dari token JWT, bukan dari body request (FR-CMT-05).
Response 201: Comment object.

## 7. Cheat Sheet / Referensi

### `GET /boards/{board_id}/cheat-sheet`
Response 200: `[{ "id", "board_id", "type", "title", "value", "author_id", "created_at" }]`

### `POST /boards/{board_id}/cheat-sheet`
Request: `{ "type": "file"|"url"|"note", "title": "string", "value": "string" }`
Untuk `type: "file"`, `value` berisi referensi berkas hasil unggah (lihat §10 Upload).
Response 201: Cheat Sheet item.

### `PATCH /boards/{board_id}/cheat-sheet/{item_id}` (super_user only, baru 2026-08-20)
Request (parsial): `{ "type", "title", "value" }` — semua field termasuk `type` bisa diganti (mis. note → file).
Otorisasi: `access_level=super_user` (cek in-handler) — 403 kalau bukan.
Response 200: Cheat Sheet item terbaru.

### `DELETE /boards/{board_id}/cheat-sheet/{item_id}` (super_user only, baru 2026-08-20)
Otorisasi: `access_level=super_user` — 403 kalau bukan.
Response 204.

## 8. Weekly Plan

### `GET /weekly-plan?week_start=2026-08-03`
Response 200:
```json
[
  {
    "big_task_id": "uuid",
    "big_task_name": "string",
    "board_id": "uuid",
    "board_name": "string",
    "actual_pct": 80,
    "expected_pct": 71,
    "last_push": { "callback_id": "string", "pushed_at": "timestamp" } 
  }
]
```
`last_push` bernilai `null` jika belum pernah di-push untuk minggu tersebut. Hanya Big Task dengan minimal satu Day Entry pada rentang minggu terpilih yang muncul (selaras FR-WKL-02).

**Scope-nya PER USER (PIC), bukan cross-user** — "My Weekly Plan", laporan pribadi ke HR (`04-architecture.md`, `01-vision-product.md` §5 poin 3), bukan rollup tim (itu peran Dashboard). Cuma Daily Task yang `pic_user_id`-nya requesting user sendiri yang dihitung; `actual_pct`/`expected_pct` juga cuma dari Daily Task dia, BUKAN agregat semua PIC di Big Task itu (kalau satu Big Task punya beberapa PIC, Big Task itu akan muncul di Weekly Plan MASING-MASING PIC dengan angka `actual_pct` yang beda-beda sesuai kerjaan masing-masing). `POST /weekly-plan/push` ikut aturan filter yang sama, biar angka yang di-push konsisten sama yang tampil di layar. Lihat `docs/decision-log/decision-log-weekly-plan-scope-per-user-20260810.md` (sempat jadi bug — awalnya gak difilter sama sekali, ketemu & diperbaiki 2026-08-10).

**Query param `as_user_id` (baru, 2026-08-10)**: `GET /weekly-plan?week_start=...&as_user_id=<uuid>` — CUMA dihormati kalau requesting user `access_level=super_user` (lihat §10), else ditolak 403 (bukan diabaikan diam-diam). Filter tetap sama (`pic_user_id`), cuma target-nya `as_user_id` bukan requester — buat super_user "lihat sebagai" user lain. Lihat `docs/decision-log/decision-log-hr-mapping-super-user-20260810.md`.

### `POST /weekly-plan/push`
Request: `{ "big_task_id": "uuid", "week_start": "2026-08-03", "as_user_id"?: "uuid" }`
Server melakukan upsert pada `weekly_push_log` berdasarkan `(big_task_id, week_start)` — generate `callback_id` baru hanya jika belum ada (FR-WKL-05). `weekly_push_log.pushed_by` SELALU actor yang benar-benar memanggil endpoint ini (buat audit), BUKAN `as_user_id` — datanya (actual_pct/expected_pct/payload HR) tetap milik target user (`as_user_id` kalau diisi, else diri sendiri).
`as_user_id` (opsional) — sama aturannya seperti `GET /weekly-plan`: CUMA dihormati buat `access_level=super_user`, else 403.
Response 200: `{ "callback_id": "string", "pushed_at": "timestamp" }`

**Integrasi HR (Fase 2, dummy/lokal, 2026-08-10)**: setelah upsert `weekly_push_log` sukses, server BEST-EFFORT kirim HTTP POST ke `myagenda-service` (sub-project terpisah, `myagenda-service/README.md`) — upsert ke tabel `my_agenda` (skema asli sistem HR). Diatur env var `MYAGENDA_SERVICE_URL` (kosong = fitur nonaktif, di-skip diam-diam). Gagal terhubung ke service ini TIDAK membatalkan response 200 di atas — `weekly_push_log` tetap sumber kebenaran utama, sinkron HR bersifat tambahan. `user_id` yang dikirim ke `my_agenda` pakai `hr_user_id` ASLI target user kalau sudah di-mapping (lihat §10), fallback ke placeholder CRC32 (dengan warning log) kalau belum — lihat `docs/decision-log/decision-log-myagenda-hr-service-20260810.md` dan `decision-log-hr-mapping-super-user-20260810.md`.

### `GET /weekly-plan/team-status?week_start=2026-08-03` (super_user only, baru 2026-08-10)
Ditolak 403 kalau requesting user bukan `access_level=super_user`.
Response 200: `[{ "user_id", "display_name", "initials", "total_big_tasks", "pushed_big_tasks", "all_pushed": boolean }]`
Satu baris per user (SEMUA user, termasuk yang `total_big_tasks: 0` minggu ini) — buat kebutuhan "cek tim udah push atau belum" tanpa buka drill-down satu-satu lewat `as_user_id`. `all_pushed = total_big_tasks > 0 && pushed_big_tasks === total_big_tasks`.

## 9. Review Queue & Notifikasi

Otorisasi: SEMUA user login (bukan cuma role `spv` lagi — lihat `docs/decision-log/decision-log-bigtask-reviewer-assignment-20260810.md`, menggantikan sebagian keputusan lama di `decision-log-review-queue-scope-20260809.md`). Hasil `GET /review-queue` dan hak akses `mark-reviewed` di-filter PER USER: item tampil/bisa ditandai kalau requesting user ada di `reviewer_user_ids` Big Task terkait (§4), ATAU Big Task itu belum punya reviewer di-assign sama sekali DAN requesting user ber-role `spv` (fallback).

### `GET /review-queue`
Mendaftar Daily Task (lintas semua board) yang **belum** ditinjau DAN requesting user berwenang atasnya (lihat aturan otorisasi di atas). item_type cuma `daily_task` pada iterasi ini.
Response 200: `[{ "type": "daily_task", "id": "uuid", "title": "string", "reviewed": false, "big_task_id": "uuid", "big_task_name": "string", "board_id": "uuid", "board_name": "string" }]`
`big_task_id`/`big_task_name`/`board_id`/`board_name` tambahan di luar draft awal kontrak, buat konteks tampilan + link ke halaman board terkait (bukan breaking change, cuma field ekstra).

### `POST /review-queue/{item_type}/{item_id}/mark-reviewed`
`item_type` yang didukung saat ini cuma `daily_task`. Ditolak 404 kalau `item_id` tidak ditemukan, 403 kalau requesting user bukan reviewer yang berwenang atas item itu (dicek eksplisit, bukan cuma nge-trust filter di `GET`).
Response 200: `{ "reviewed": true, "reviewed_by": "uuid", "reviewed_at": "timestamp" }`
Tidak mengubah field progress/status entitas terkait (FR-NTF-02). Idempoten — dipanggil ulang pada item yang sama cuma update `reviewed_by`/`reviewed_at`.

## 10. Users, Roles, & Pengaturan

Semua endpoint di bawah SEKARANG juga mengembalikan/menerima `access_level` (`"super_user"` | `"regular_user"`, default `"regular_user"`) dan `hr_user_id` (int atau `null`) — lihat `docs/decision-log/decision-log-hr-mapping-super-user-20260810.md`. `access_level` **konsep terpisah dari `roles`** (many-to-many) — satu user cuma salah satu dari dua nilai itu, bukan bisa dirangkap kayak role biasa.

### `GET /users` — daftar pengguna beserta role (otorisasi: admin/spv)
Response 200: `[{ "id", "display_name", "initials", "email", "org_team", "roles": ["dev","qa"], "access_level", "hr_user_id" }]`

### `POST /users` — buat pengguna baru (otorisasi: admin/spv)
Request: `{ "display_name", "email", "password", "initials", "org_team", "role_codes": ["dev","qa"], "access_level"?, "hr_user_id"? }`
`password` wajib — admin/SPV menentukan password awal user baru dan menyampaikannya out-of-band (lihat `docs/decision-log/decision-log-users-api-gaps-20260809.md`). Di-hash bcrypt sebelum disimpan, tidak pernah disimpan/dikembalikan plain text. `org_team` divalidasi harus ada di `referensi_tim` (400 kalau tidak). `access_level`/`hr_user_id` opsional (default `regular_user`/`null`).
Response 201: `{ "id", "display_name", "initials", "email", "org_team", "roles", "access_level", "hr_user_id" }`

### `PATCH /users/{id}` — edit pengguna yang SUDAH ADA (otorisasi: admin/spv, baru)
Request (parsial): `{ "org_team"?, "access_level"?, "hr_user_id"?, "clear_hr_user_id"?: boolean, "role_codes"?: ["dev","qa"] }`
Sebelum ini TIDAK ADA cara ubah user existing (cuma `POST /users` buat bikin baru) — ditambahkan supaya admin bisa map `hr_user_id`/set `access_level`/ubah `org_team`/role ke user lama. `role_codes`, kalau dikirim, MENGGANTI SELURUH assignment role user itu (bukan menambah — klien wajib kirim daftar lengkap). `hr_user_id: null` dengan `clear_hr_user_id: true` melepas mapping HR.
Response 200: bentuk sama seperti `GET /users` (satu object).

### `GET /users/assignable` — daftar ringkas user untuk form assignment PIC (semua pengguna terautentikasi, FR-ASG-02)
Response 200: `[{ "id", "display_name", "initials", "org_team", "roles": ["dev","qa"] }]`
Tanpa `email`/`access_level`/`hr_user_id` — berbeda dari `GET /users` yang dibatasi admin/spv (lihat `docs/decision-log/decision-log-pic-assignment-endpoint-20260809.md`).

### `GET /users/me` — profil milik user yang sedang login
Response 200: `{ "id", "display_name", "initials", "email", "org_team", "roles", "theme_preference", "access_level", "hr_user_id" }`

### `PATCH /users/me` — update profil sendiri
Request (parsial): `{ "display_name", "initials", "current_password", "password", "theme_preference" }`
`current_password` **wajib** disertakan kalau field `password` ada di request (ganti password) — ditolak 401 kalau tidak cocok dengan password saat ini. Tidak dibutuhkan untuk update field lain. TIDAK bisa ubah `access_level`/`hr_user_id` sendiri lewat endpoint ini (lihat `PATCH /users/{id}`, admin-only).
Response 200: bentuk sama seperti `GET /users/me`.

### `GET /roles` — daftar role tersedia (untuk keperluan filter assignment, FR-ASG-02)
Response 200: `[{ "code", "label" }]`

### `GET /referensi-tim` — daftar tim/org (otorisasi: admin/spv, baru)
Response 200: `[{ "id", "name" }]`
Sumber dropdown "Tim/Org" di form user — gantiin free-text (lihat decision log). `users.org_team` TETAP kolom text apa adanya, cuma divalidasi terhadap tabel ini.

### `POST /referensi-tim` — tambah nama tim baru (otorisasi: admin/spv, baru)
Request: `{ "name" }`
Response 201: `{ "id", "name" }`
Bukan CRUD penuh (belum ada update/delete).

### `GET /referensi-user-hr` — daftar pegawai sistem HR asli (otorisasi: admin/spv, baru)
Response 200: `[{ "hr_user_id", "email", "nip", "nama_lengkap" }]`
Data MILIK sistem HR eksternal, di-seed dari export yang diberikan — TIDAK ADA CRUD UI buat tabel ini (update resmi lewat migration baru). Dipakai buat mapping `users.hr_user_id` di Manajemen User.

### `POST /uploads` (multipart/form-data, field `file`)
Dipakai oleh Cheat Sheet tipe `file`. Response: `{ "value": "string (nama file tersimpan, dipakai sebagai value cheat-sheet)" }`
`value` yang dikembalikan berbeda dari nama file asli (di-prefix UUID untuk mencegah collision) — lihat `docs/decision-log/decision-log-file-upload-storage-20260809.md`.

### `GET /uploads/{filename}`
Mengambil kembali file yang sudah diupload (dipakai sebagai link unduh pada Cheat Sheet tipe `file`). Tidak didokumentasikan di draft kontrak awal, ditambahkan karena tanpa ini upload tidak bisa diambil kembali — lihat decision log di atas.

## 11. Ringkasan Kode Status HTTP

| Kode | Pemakaian |
|---|---|
| 200 | Sukses (GET, update, aksi non-create) |
| 201 | Entitas baru berhasil dibuat |
| 204 | Sukses tanpa body (delete, logout) |
| 400 | Input tidak valid |
| 401 | Tidak terautentikasi |
| 403 | Terautentikasi namun tidak berwenang (mis. non-SPV memanggil sign-off) |
| 404 | Entitas tidak ditemukan |
| 409 | Konflik aturan bisnis (mis. sign-off saat `actual_pct` < 100) |
