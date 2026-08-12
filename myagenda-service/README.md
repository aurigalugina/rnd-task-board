# myagenda-service

Sub-project **terpisah** dari `rnd-task-board` — service kecil buat nerima push mingguan ("Push to HR", Fase 2 di `docs/01-vision-product.md`) dan upsert ke tabel `my_agenda`, skema asli sistem MyAgenda-HR (DDL di `db/migrations/0001_create_my_agenda.up.sql` **dipakai persis**, tidak dimodifikasi).

Untuk sekarang jalan pakai MySQL lokal (dummy, buat development/testing alur). Production-nya rencananya dipasang langsung di server aplikasi HR (di luar repo ini) — makanya sub-project ini punya `go.mod`, `docker-compose.yml`, dan `Makefile` sendiri, terpisah total dari `rnd-task-board`.

Detail keputusan & alasan lengkap: `../docs/decision-log/decision-log-myagenda-hr-service-20260810.md`.

## Menjalankan

```bash
cd myagenda-service
make up            # docker compose up --build (mysql + api, hot-reload via air)
make migrate-up    # sekali di awal
```

Service jalan di `http://localhost:8090`, MySQL di `localhost:3307` (user/pass/db: `myagenda`/`myagenda`/`myagenda`).

## API

### `POST /my-agenda`
Upsert satu baris `my_agenda`, identitas natural `(user_id, judul_task, tgl_rencana)` — DDL-nya tidak punya unique constraint di luar PK auto-increment, jadi upsert dilakukan di level aplikasi (SELECT dulu, lalu UPDATE atau INSERT).

Request:
```json
{
  "user_id": 123,
  "judul_task": "API Gateway Refactor",
  "tgl_rencana": "2026-08-10",
  "uraian_task": "IBS Gen 3 — Core Banking / API Gateway Refactor",
  "due_date": "2026-08-29",
  "target": 100,
  "capaian": 30,
  "is_percentage": true
}
```
`prosentase_capaian` dihitung server-side (`round(capaian)`), tidak perlu dikirim.

Response 201 (baris baru) / 200 (baris sudah ada, di-update):
```json
{ "my_agenda_id": 1, "action": "created" }
```

### `GET /healthz`
`{"status":"ok"}`.

## ⚠️ Belum siap production — hal yang WAJIB diselesaikan sebelum dipasang beneran

- **`user_id` di request ini MASIH PLACEHOLDER** (di-derive CRC32 dari UUID user `rnd-task-board`, bukan id pegawai HR asli — lihat `backend/internal/weeklyplan/handler.go` di repo utama). Belum ada mapping `users.id` (UUID kita) → employee id sistem HR. Ini keputusan terpisah yang butuh input tim HR/System Analyst.
- Tidak ada auth di endpoint `/my-agenda` — aman untuk sekarang karena dipanggil server-ke-server secara internal (bukan dari browser), TAPI kalau network deployment-nya berubah (mis. tidak lagi internal-only), WAJIB ditambah otentikasi (API key/mTLS/dst) sebelum expose ke luar.
- Belum ada retry/queue kalau service ini down saat `rnd-task-board` push — sifatnya best-effort (lihat decision log), push lokal (`weekly_push_log`) tetap jadi sumber kebenaran utama.
