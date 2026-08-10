# Arsitektur Sistem — R&D Ops

**Status:** Draft v1
**Referensi:** `01-vision-product.md`, `02-brd.md`, `03-srs.md`

---

## 1. Gaya Arsitektur

Monolith modular. Mengingat skala tim kecil (satuan digit pengguna aktif) dan kebutuhan untuk dapat dijalankan penuh di mesin pengembangan lokal, arsitektur microservices tidak memberi nilai tambah pada iterasi ini — justru menambah biaya operasional (orkestrasi, observability lintas service) yang tidak proporsional dengan skala pemakaian.

Backend disusun sebagai satu binari Go dengan pemisahan modul secara logis per domain (board, big task, daily task, comment, weekly plan, user), bukan pemisahan proses.

## 2. Komponen Sistem

```
┌─────────────────────────────────────────────────────────────┐
│                         Nginx (reverse proxy)                 │
│   - TLS termination                                            │
│   - Serve static build SvelteKit (/)                          │
│   - Reverse proxy /api/* → Go backend                         │
└───────────────┬─────────────────────────────┬─────────────────┘
                │                             │
     ┌──────────▼──────────┐       ┌──────────▼──────────┐
     │   SvelteKit (SPA)    │       │   Go Backend (API)   │
     │   adapter-static      │       │   - chi router        │
     │   build output        │       │   - sqlc (typed SQL)  │
     └───────────────────────┘       │   - JWT auth           │
                                     └──────────┬────────────┘
                                                │
                                     ┌──────────▼────────────┐
                                     │     PostgreSQL          │
                                     │  - schema inti           │
                                     │  - migration: golang-    │
                                     │    migrate                │
                                     └─────────────────────────┘

     ┌──────────────────────────────────────────────────────┐
     │   (Fase 2 — belum dibangun) Konektor MyAgenda-HR       │
     │   dipicu manual dari Weekly Plan → HTTP call ke         │
     │   endpoint HR eksternal dengan payload upsert            │
     └──────────────────────────────────────────────────────┘
```

## 3. Deployment Topology (Lokal & Lanjutan)

**Tahap pengembangan (laptop lokal):**
- `docker compose up` menjalankan 4 container: `postgres`, `backend`, `frontend-build` (build step, hasil disalin ke volume nginx), `nginx`.
- Seluruh service pada satu Docker network internal; hanya Nginx yang mem-publish port ke host.

**Tahap operasional lanjutan (di luar cakupan dokumen ini, dicatat sebagai arah):**
- Deployment ke salah satu server internal yang sudah ada dalam mesh Tailscale/Headscale milik tim, mengikuti pola infrastruktur yang sudah berjalan untuk proyek lain (registry Docker privat, multi-server berbasis container).

## 4. Struktur Direktori Proyek (Rencana)

```
rnd-ops-portal/
├── docs/                     # dokumen ini
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── board/
│   │   ├── bigtask/
│   │   ├── dailytask/
│   │   ├── comment/
│   │   ├── cheatsheet/
│   │   ├── weeklyplan/
│   │   ├── user/
│   │   └── auth/
│   ├── db/
│   │   ├── migrations/
│   │   └── queries/          # sqlc source
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── routes/
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   └── api/
│   │   └── app.html
│   └── svelte.config.js
├── nginx/
│   └── nginx.conf
└── docker-compose.yml
```

## 5. Alur Data Kunci

### 5.1 Perhitungan Verdict (dihitung saat baca, bukan disimpan)
`expected_pct` dan `verdict` Big Task **tidak disimpan** sebagai kolom statis di basis data — keduanya bergantung pada `TODAY` (waktu saat permintaan dibuat) dan akan berubah setiap hari meskipun tidak ada aksi pengguna. Backend menghitung nilai ini secara real-time pada setiap permintaan baca (query time), berdasarkan `start_date`, `deadline`, dan agregasi `actual_pct` dari Daily Task terkait.

Ini adalah keputusan arsitektural penting: menyimpan `verdict` sebagai kolom statis akan memerlukan job terjadwal (cron) untuk memperbaruinya tiap hari, menambah kompleksitas tanpa manfaat sepadan pada skala data ini.

### 5.2 Push Weekly Plan ke HR (Fase 2, disiapkan strukturnya)
1. Pengguna memicu aksi push dari tampilan Weekly Plan.
2. Backend menghitung ulang rollup mingguan pada saat itu juga (bukan dari cache).
3. Backend mencari `weekly_push_log` berdasarkan `(big_task_id, week_start)`. Jika belum ada, generate `callback_id` baru; jika sudah ada, gunakan `callback_id` yang tersimpan.
4. Backend menyimpan/memperbarui baris `weekly_push_log` dengan `pushed_at` terbaru.
5. *(Fase 2)* Backend melakukan HTTP call ke endpoint HR eksternal dengan payload yang menyertakan `callback_id` agar sisi penerima dapat melakukan upsert.

Pada iterasi Fase 1, langkah 5 disimulasikan (dicatat lokal saja) karena endpoint HR eksternal belum tersedia — namun struktur data dan alur di atas sudah final agar Fase 2 tinggal menyambungkan pemanggilan HTTP tanpa perubahan skema.

## 6. Keputusan Teknologi & Alasan

| Keputusan | Alasan |
|---|---|
| SvelteKit + `adapter-static` | Routing berbasis file untuk DX yang baik; hasil build statis cukup untuk kebutuhan internal tanpa SSR |
| Go + chi | Binari tunggal, mudah deploy, cocok untuk tim kecil; chi dipilih atas net/http murni untuk routing middleware yang lebih rapi tanpa overhead framework berat |
| sqlc | SQL eksplisit (bukan DSL ORM), selaras dengan kebiasaan tim menulis query langsung; menghasilkan kode Go typesafe dari SQL |
| golang-migrate | Migration terpisah dari kode aplikasi, dapat dijalankan independen saat deployment |
| PostgreSQL | Dukungan JSONB berguna untuk struktur fleksibel (mis. metadata tema/preferensi); transaksi ACID untuk konsistensi agregasi |
| Nginx | Titik terminasi TLS tunggal, reverse proxy, dan penyaji statis; selaras dengan pola infrastruktur lain yang sudah dipakai tim |
| JWT (stateless auth) | Menghindari kebutuhan session store terpisah pada skala pengguna kecil ini |

## 7. Keamanan (Ringkas)

- Autentikasi: JWT dengan masa berlaku terbatas, refresh token tersimpan sebagai HTTP-only cookie.
- Otorisasi: middleware role-check di layer backend untuk endpoint sensitif (sign-off, manajemen pengguna, push HR), bukan hanya disembunyikan di UI.
- Password disimpan sebagai hash (bcrypt/argon2), tidak pernah dalam bentuk plain text.
- Seluruh trafik antara Nginx dan klien wajib HTTPS saat berjalan di luar `localhost`.

## 8. Observability (Minimal, Sesuai Skala)

- Structured logging (JSON) pada backend Go untuk seluruh operasi tulis (create/update/sign-off/push).
- Tidak memerlukan stack observability penuh (mis. Prometheus/Grafana) pada iterasi awal — log terstruktur cukup untuk skala tim ini dan dapat ditingkatkan kelak tanpa perubahan arsitektur inti.

## 9. Ekstensibilitas — Change Request

Modul `change_request` disiapkan sebagai entitas data sejak awal (lihat `06-db-design.md`) namun tanpa antarmuka penuh pada Fase 1. Rencana integrasi ke depan: form input percakapan bebas pada frontend → dikirim ke endpoint backend yang menyimpan mentahnya → proses penyusunan `change_request.md` terstruktur dilakukan di luar aplikasi (oleh agentic coding assistant yang berjalan pada repository proyek) → hasilnya direferensikan kembali ke baris `change_requests` terkait untuk keperluan triase oleh SPV/SA.

Arsitektur ini sengaja tidak menyertakan pemanggilan API model bahasa dari dalam aplikasi pada Fase 1, untuk menghindari kompleksitas dan biaya operasional yang belum diperlukan pada tahap ini.
