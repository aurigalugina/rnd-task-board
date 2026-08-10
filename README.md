# R&D Ops — Portal Internal Tim R&D

Dokumentasi produk lengkap ada di `docs/`:
`01-vision-product.md`, `02-brd.md`, `03-srs.md`, `04-architecture.md`, `05-api-contract.md`, `06-db-design.md`.

## Stack

| Layer | Teknologi |
|---|---|
| Frontend | SvelteKit (`adapter-static`) |
| Backend | Go (chi router, pgx) |
| Database | PostgreSQL 16 |
| Reverse proxy | Nginx |

## Menjalankan di Lokal

### 1. Siapkan dependency

Sandbox tempat scaffold ini dibuat tidak punya akses ke `proxy.golang.org`, jadi `go.sum` **belum** ter-generate. Jalankan ini di laptop Anda (butuh koneksi internet):

```bash
cd backend
go mod tidy
```

```bash
cd frontend
npm install
```

### 2. Jalankan seluruh stack

```bash
make up
# setara dengan: docker compose up --build
```

### 3. Jalankan migration (sekali di awal, dan setiap ada migration baru)

```bash
make migrate-up
```

### 4. (Opsional) Seed user untuk testing login

```bash
make seed
```

Login: `spv@rndops.local` / `password123`. Setelah insert, jalankan manual query di `backend/db/seed/0001_seed_dev_user.sql` bagian bawah untuk menautkan role ke user tersebut (lihat komentar di file itu) — sengaja dipisah agar `id` user bisa dicatat dulu.

Aplikasi dapat diakses di `http://localhost:8080`.

## Versi Dev (hot-reload)

`docker-compose.yml` di atas adalah versi **build**: multi-stage Dockerfile, frontend jadi static bundle, disajikan nginx — cocok buat coba stack lengkap, tapi setiap ubah kode harus `docker compose up --build` ulang.

Untuk kerja sehari-hari (ubah kode, langsung kepakai tanpa rebuild manual), pakai `docker-compose.dev.yml`:

```bash
make up-dev
# setara dengan: docker compose -f docker-compose.dev.yml up --build
```

- **Backend** (`backend/Dockerfile.dev`) pakai [`air`](https://github.com/air-verse/air) — watch file `.go`, auto rebuild+restart tiap save.
- **Frontend** jalanin `vite dev` asli (HMR) di container `node:20-alpine`, bukan build statis.
- Source code di-*bind mount* dari host, jadi editor/IDE biasa (bukan di dalam container) tetap kepakai.

Akses:
- Frontend (dev server, HMR): `http://localhost:5173`
- Backend API langsung: `http://localhost:8080/api/v1/...`

Port sama dengan versi build (`5432`/`8080`), jadi **jangan jalankan dua-duanya bersamaan** — `make down` dulu kalau stack build masih nyala, baru `make up-dev`.

Migration & seed tetap pakai target Makefile yang sudah ada (nyambung ke DB manapun yang sedang aktif di `localhost:5432`):

```bash
make migrate-up
make seed-dev   # kalau lagi pakai stack dev (make seed kalau stack build)
```

Matikan stack dev: `make down-dev`.

## Struktur Proyek

Lihat `docs/04-architecture.md` §4 untuk penjelasan lengkap; ringkasannya:

```
rnd-ops-portal/
├── docs/           dokumen produk (Vision, BRD, SRS, Architecture, API, DB)
├── backend/        Go API (chi + pgx)
├── frontend/       SvelteKit
├── nginx/          konfigurasi reverse proxy
├── docker-compose.yml
└── Makefile        shortcut perintah umum
```

## Status Implementasi Scaffold

Modul yang **sudah diimplementasikan penuh** (handler + routing, siap dikonsumsi frontend):

- **Board** — list, create, dan `summary` (matriks dashboard per board).
- **Big Task** — list dengan kalkulasi `verdict`/`expected_pct` real-time, create, sign-off (dengan validasi 100%), undo sign-off.
- **Daily Task** — create (otomatis generate Day Entry per tanggal kalender), update Day Entry inline.

Modul yang **didokumentasikan lengkap di API Contract namun handler-nya belum ditulis** (silakan diimplementasikan mengikuti pola yang sama seperti `internal/board`/`internal/bigtask`):

- Comments (§6), Cheat Sheet (§7), Weekly Plan (§8), Review Queue (§9), Users & Roles (§10).

Frontend baru berisi shell navigasi (empat halaman: Dashboard, Boards, My Weekly Plan, Review Queue) dan halaman Dashboard yang sudah terhubung ke API `GET /boards`. Halaman lain masih placeholder deskriptif.

## Keputusan yang Perlu Diketahui

- **`verdict` dan `expected_pct` tidak pernah disimpan di database** — selalu dihitung ulang di backend saat request masuk (lihat `docs/04-architecture.md` §5.1). Ini keputusan sadar untuk menghindari kebutuhan cron job harian.
- **sqlc belum diaktifkan** — handler saat ini query langsung via `pgx` agar scaffold ini bisa langsung dibangun tanpa langkah `sqlc generate` tambahan. `sqlc.yaml` dan pola query sudah didokumentasikan di `docs/06-db-design.md` §5 sebagai referensi bila ingin migrasi ke pendekatan generated code nanti.
- Kredensial database default (`rndops`/`rndops`) dan `JWT_SECRET` di `docker-compose.yml` **hanya untuk pengembangan lokal** — wajib diganti sebelum di-deploy ke server yang diakses bersama tim.
# rnd-task-board
# rnd-task-board
