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
Response 200: `[{ "id", "name", "tag" }]`

### `POST /boards`
Request: `{ "name": "string", "tag": "string" }`
Response 201: Board object.

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
    "signed_at": null
  }
]
```
`verdict` ∈ `{ "on_progress", "win", "lose" }` — dihitung sesuai RULE-04/05/06 (BRD).

### `POST /boards/{board_id}/big-tasks`
Request: `{ "name", "start_date", "deadline", "default_pic_user_id" }`
Response 201: Big Task object (bentuk sama seperti di atas, `actual_pct` = 0 karena belum ada Daily Task).

### `POST /big-tasks/{big_task_id}/sign-off`
Otorisasi: role `spv`. Ditolak (409) apabila `actual_pct` < 100.
Response 200: Big Task object dengan `signed: true`.

### `DELETE /big-tasks/{big_task_id}/sign-off`
Otorisasi: role `spv`. Undo sign-off.
Response 204.

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
        "is_done": true,
        "blocker_text": "",
        "is_weekend": false
      }
    ]
  }
]
```

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
{ "planned_text": "string", "is_done": true, "blocker_text": "string" }
```
Response 200: Day Entry object terbaru (termasuk `id` — dibutuhkan klien karena inilah satu-satunya cara mendapat `day_entry_id` untuk PATCH berikutnya, lihat `id` di response `GET .../daily-tasks` §5).
Catatan: saat `is_done` diset `true`, klien direkomendasikan mengirim `blocker_text: ""` dalam payload yang sama (selaras perilaku mockup: menandai selesai mengosongkan blocker).

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

### `POST /weekly-plan/push`
Request: `{ "big_task_id": "uuid", "week_start": "2026-08-03" }`
Server melakukan upsert pada `weekly_push_log` berdasarkan `(big_task_id, week_start)` — generate `callback_id` baru hanya jika belum ada (FR-WKL-05).
Response 200: `{ "callback_id": "string", "pushed_at": "timestamp" }`

## 9. Review Queue & Notifikasi

Otorisasi: role `spv` saja untuk kedua endpoint di bawah — lihat `docs/decision-log/decision-log-review-queue-scope-20260809.md` untuk alasan (BRD BR-11.1 cuma menyebut kebutuhan ini untuk SPV, role lain tidak punya use case yang terdokumentasi).

### `GET /review-queue`
Mendaftar SELURUH Daily Task (lintas semua board) yang **belum** ditinjau (item_type cuma `daily_task` pada iterasi ini).
Response 200: `[{ "type": "daily_task", "id": "uuid", "title": "string", "reviewed": false, "big_task_id": "uuid", "big_task_name": "string", "board_id": "uuid", "board_name": "string" }]`
`big_task_id`/`big_task_name`/`board_id`/`board_name` tambahan di luar draft awal kontrak, buat konteks tampilan + link ke halaman board terkait (bukan breaking change, cuma field ekstra).

### `POST /review-queue/{item_type}/{item_id}/mark-reviewed`
`item_type` yang didukung saat ini cuma `daily_task`. Ditolak 404 kalau `item_id` tidak ditemukan.
Response 200: `{ "reviewed": true, "reviewed_by": "uuid", "reviewed_at": "timestamp" }`
Tidak mengubah field progress/status entitas terkait (FR-NTF-02). Idempoten — dipanggil ulang pada item yang sama cuma update `reviewed_by`/`reviewed_at`.

## 10. Users, Roles, & Pengaturan

### `GET /users` — daftar pengguna beserta role (otorisasi: admin/spv)
Response 200: `[{ "id", "display_name", "initials", "email", "org_team", "roles": ["dev","qa"] }]`

### `POST /users` — buat pengguna baru (otorisasi: admin/spv)
Request: `{ "display_name", "email", "password", "initials", "org_team", "role_codes": ["dev","qa"] }`
`password` wajib — admin/SPV menentukan password awal user baru dan menyampaikannya out-of-band (lihat `docs/decision-log/decision-log-users-api-gaps-20260809.md`). Di-hash bcrypt sebelum disimpan, tidak pernah disimpan/dikembalikan plain text.
Response 201: `{ "id", "display_name", "initials", "email", "org_team", "roles" }`

### `GET /users/assignable` — daftar ringkas user untuk form assignment PIC (semua pengguna terautentikasi, FR-ASG-02)
Response 200: `[{ "id", "display_name", "initials", "org_team", "roles": ["dev","qa"] }]`
Tanpa `email` — berbeda dari `GET /users` yang dibatasi admin/spv (lihat `docs/decision-log/decision-log-pic-assignment-endpoint-20260809.md`).

### `GET /users/me` — profil milik user yang sedang login
Response 200: `{ "id", "display_name", "initials", "email", "org_team", "roles", "theme_preference" }`

### `PATCH /users/me` — update profil sendiri
Request (parsial): `{ "display_name", "initials", "current_password", "password", "theme_preference" }`
`current_password` **wajib** disertakan kalau field `password` ada di request (ganti password) — ditolak 401 kalau tidak cocok dengan password saat ini. Tidak dibutuhkan untuk update field lain.
Response 200: bentuk sama seperti `GET /users/me`.

### `GET /roles` — daftar role tersedia (untuk keperluan filter assignment, FR-ASG-02)
Response 200: `[{ "code", "label" }]`

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
