# Decision Log — Docker Compose Terpisah untuk Dev (Hot-Reload)

**Tanggal:** 2026-08-10
**Konteks:** docker-dev-compose

## Konteks/Masalah

`docker-compose.yml` yang ada cuma versi build/prod: multi-stage Dockerfile (`go build` sekali jadi binary statis, `npm run build` jadi bundle statis disajikan nginx). Cocok buat simulasi deploy, tapi setiap ubah kode harus `docker compose up --build` ulang dari nol — lambat buat kerja sehari-hari, dan berbeda dari workflow biasa (`go run ./cmd/api` + `npm run dev` langsung di host) yang butuh Go/Node terinstall lokal. User minta versi docker-compose khusus dev yang hot-reload.

## Keputusan

Buat file **terpisah** `docker-compose.dev.yml` (bukan menambah profile/override ke file yang sama), dengan `name: rndops-dev` (project name beda) supaya volume/network-nya tidak bertabrakan dengan stack build:

- **postgres**: sama seperti versi build.
- **backend**: `backend/Dockerfile.dev` — base `golang:1.22-alpine` + [`air`](https://github.com/air-verse/air) buat watch `.go` dan auto rebuild+restart. Source di-bind-mount (`./backend:/app`); Go module cache (`/go/pkg/mod`) dan build output air (`/app/tmp`) pakai named volume terpisah (bukan host dir langsung) — biar cache-nya persist antar restart tapi gak nyampah root-owned file ke host (lihat gotcha WSL2+Docker root-owned files yang sudah pernah kejadian di project lain).
- **frontend**: image `node:20-alpine` polos (gak perlu Dockerfile sendiri), jalanin `npm install && npm run dev -- --host --port 5173` langsung — vite dev server asli, HMR beneran, bukan build statis. `node_modules` pakai named volume (isolasi dari host, hindari mismatch native binding & install ulang tiap restart).
- **Tidak ada nginx** di versi dev — vite dev server sendiri yang serve frontend + proxy `/api` ke backend (`server.proxy` di `vite.config.ts`), sesuai workflow SvelteKit dev standar.
- `vite.config.ts` diubah supaya target proxy `/api` bisa di-override lewat env var `VITE_API_PROXY_TARGET` (default tetap `http://localhost:8080`, TIDAK mengubah behavior dev-lokal-tanpa-docker yang sudah ada) — perlu karena di dalam container frontend, `localhost:8080` merujuk ke dirinya sendiri, bukan container backend; compose set env ini ke `http://backend:8080`.

## Alasan

- **File terpisah, bukan profile di file yang sama**: dev dan build punya shape container yang beda banget (single-stage+bind-mount vs multi-stage+static-output) — nyampur di satu file pakai `profiles:` bikin file itu susah dibaca dan gampang salah pas cuma mau jalanin salah satu. Konvensi umum juga `docker-compose.<env>.yml` per environment.
- **`name: rndops-dev` beda dari default**: biar `docker compose -f docker-compose.dev.yml` dan `docker compose` (pakai `docker-compose.yml`) dianggap project terpisah oleh Docker — kalau nama project sama (default: nama folder), volume Postgres dkk bisa ke-mix atau konflik nama network/container antara dua stack.
- **`air` dipilih ketimbang cuma `go run` polos**: `go run` gak auto-restart pas file berubah, jadi "dev" experience-nya sama aja kayak versi build (harus restart container manual). `air` kasih hot-reload beneran, itungannya proporsional buat effort tambahan (satu Dockerfile.dev + satu `.air.toml`).
- **Port disamakan dengan versi build (5432/8080)**, bukan di-offset: supaya `make migrate-up`/`make migrate-down` yang sudah ada (hit `localhost:5432` langsung, gak lewat compose) tetap kepakai apa adanya buat stack manapun yang lagi nyala, tanpa perlu bikin target migrate terpisah per environment. Trade-off: dua stack gak bisa nyala bersamaan (harus `make down` dulu) — dianggap acceptable karena jarang ada kebutuhan nyalain build DAN dev bersamaan.

## Dampak/File Terpengaruh

- `docker-compose.dev.yml` (baru).
- `backend/Dockerfile.dev`, `backend/.air.toml` (baru).
- `frontend/vite.config.ts` — proxy target jadi `process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080'`, tambah `server.host = true`.
- `Makefile` — target baru `up-dev`, `down-dev`, `seed-dev`.
- `README.md` — section baru "Versi Dev (hot-reload)".
- `.gitignore` — tambah `backend/tmp/` (mountpoint kosong yang kebuat di host tiap air jalan lewat named volume).
- `CLAUDE.md` — section baru "Dev Environment (Docker)" mencatat gotcha `GOTOOLCHAIN=auto` (air@latest butuh Go lebih baru dari base image) dan rule proxy env var.
