# CLAUDE.md

File ini jadi panduan untuk Claude Code (claude.ai/code) saat kerja di repo ini.

## Living Memory — WAJIB DIBACA

File ini bukan dokumen statis sekali-tulis, tapi **living memory** — harus terus diperbarui seiring project berjalan. Setiap sesi Claude Code berikutnya:

- **Update file ini** kalau menemukan keputusan arsitektur baru, aturan bisnis baru, atau asumsi lama di sini yang ternyata sudah berubah/salah.
- **Jangan biarkan basi.** Kalau kode sudah menyimpang dari yang tertulis di sini, perbarui bagian terkait saat itu juga — jangan tunggu diminta.
- **Tulis ringkas, bukan naratif.** Tambahkan fakta + alasan singkat (state final, bukan cerita proses debugging).
- File ini menyimpan **rules + state final saat ini**. Histori bug/gotcha/insight non-obvious (kronologis, dengan konteks penemuannya) ada di `docs/lessons-learned.md` — kalau ragu sesuatu itu "rule saat ini" atau "cerita masa lalu", state final masuk sini, narasinya masuk lessons-learned.md.

## Decision Log — wajib dicatat tiap final decision

Setiap kali sebuah keputusan sudah **final** (bukan lagi didiskusikan — sudah disepakati untuk dijalankan), catat dulu ke decision log **sebelum** lanjut implementasi:

- Lokasi: `docs/decision-log/decision-log-[context]-YYYYMMDD.md`
- `[context]` = slug singkat kebab-case yang menjelaskan topik keputusan, mis. `kanban-state-machine`, `role-based-auth`, `sqlc-migration`.
- `YYYYMMDD` = tanggal keputusan difinalisasi.
- Satu file per keputusan/topik diskusi (bukan satu file besar yang terus ditambah).
- Isi minimal: **konteks/masalah**, **keputusan yang diambil**, **alasan** (termasuk alternatif yang ditolak kalau relevan), dan **dampak/file yang terpengaruh**.
- Kalau keputusan itu juga mengubah pola/arsitektur jangka panjang, sinkronkan ringkasannya ke bagian relevan di CLAUDE.md ini juga.

## Testing — wajib sebelum task code dianggap selesai

Setiap task yang menyentuh logika kode (fitur baru, bug fix, refactor — bukan cuma dokumentasi/config) **wajib ada unit test-nya sebelum task itu dilaporkan selesai**, bukan ditunda ke sesi/fase testing terpisah nanti.

- **Backend (Go):** tambah/perbarui `_test.go` untuk logic yang disentuh — prioritaskan logika computed murni (`computeVerdict`, kalkulasi `actual_pct`/`expected_pct`, rollup mingguan, dsb) karena itu yang paling gampang salah diam-diam dan paling sering disebut sebagai prinsip tidak bisa ditawar di BRD. Jalankan `go test ./...` sampai hijau sebelum lapor selesai.
- **Frontend:** minimal `npm run check` tiap ubah `.svelte`/`.ts`. Kalau ada logic non-trivial yang ditambahkan (bukan cuma markup/styling), tulis unit test-nya juga. Test runner: **vitest** (`npm run test`) — logic murni yang layak ditest diekstrak ke file `.ts` biasa di `lib/` (bukan ditinggal inline di `<script>` komponen), supaya bisa di-import & ditest tanpa render komponen. Contoh: `lib/dateRange.ts`, `lib/comments.ts`, `lib/dashboardStats.ts`.
- Kalau nemu logic computed murni baru yang belum ada test-nya, jangan tunggu "fase testing" — langsung tambah saat itu juga.
- Task dengan perubahan yang murni tidak-testable via unit test (mis. wiring UI yang divalidasi manual di browser, config) boleh cukup divalidasi manual — tapi kalau ada logic yang bisa diuji di dalamnya, logic itu tetap wajib ditest.

## Project

R&D Ops — portal internal untuk tracking board tim, "Big Task" (level proyek), dan "Daily Task" (eksekusi harian) dengan mekanisme sign-off/verdict. Spek produk lengkap ada di `docs/`: `01-vision-product.md`, `02-brd.md`, `03-srs.md`, `04-architecture.md`, `05-api-contract.md`, `06-db-design.md`. Dokumen-dokumen ini adalah sumber kebenaran untuk aturan bisnis (BRD RULE-xx, SRS FR-xx) — baca bagian relevan sebelum implementasi endpoint baru atau mengubah logika computed-field.

**`myagenda-service/` adalah sub-project TERPISAH** (Go module, docker-compose, Makefile, README sendiri) — integrasi Push-to-HR (Fase 2), sengaja dipisah karena production-nya akan dideploy di server aplikasi HR sendiri, bukan bareng rnd-task-board. Lihat `myagenda-service/README.md`.

**`claude-chat-service/` adalah sub-project TERPISAH lainnya** (Node.js/TypeScript, docker-compose sendiri) — bridge antara Claude Code dan client aplikasi, fondasi mekanisme evolusi produk di `01-vision-product.md` §6 (change request via percakapan). Lihat bagian "Claude Chat Bridge" di bawah.

## Commands

```bash
make up             # docker compose up --build (postgres + backend + nginx, full stack)
make down
make migrate-up     # jalankan migration via image docker golang-migrate
make migrate-down   # rollback satu migration
make seed           # seed dev user (spv@rndops.local / password123)

# Versi dev (hot-reload) -- docker-compose.dev.yml, port sama (5432/8080) jadi
# jangan bareng-bareng sama `make up`. Frontend HMR di :5173 (bukan :8080, gak ada nginx).
make up-dev         # air (backend) + vite dev server asli (frontend), source di-bind-mount dari host
make down-dev
make seed-dev       # sama seperti `make seed` tapi target ke stack -f docker-compose.dev.yml
```

Backend (Go), dari `backend/`:
```bash
go mod tidy
go build ./...
go vet ./...
go run ./cmd/api        # butuh DATABASE_URL, fallback ke kredensial dev localhost:5432 kalau tidak diset
go test ./...           # wajib hijau sebelum task code dianggap selesai
```

Frontend (SvelteKit), dari `frontend/`:
```bash
npm install
npm run dev             # vite dev server, expect backend bisa diakses di /api/v1 (via proxy)
npm run build
npm run check           # svelte-check type checking — jalankan ini setelah ubah file .svelte/.ts
npm run test            # vitest — unit test untuk logic non-trivial
```

Aplikasi tersaji di `http://localhost:8080` kalau dijalankan via `make up` (nginx proxy `/api/*` ke backend Go, dan menyajikan hasil build statis SvelteKit untuk selebihnya).

## Dev Environment (Docker)

`docker-compose.dev.yml` — versi hot-reload dari `docker-compose.yml` (yang itu murni buat build/prod). Detail lengkap: `docs/decision-log/decision-log-docker-dev-compose-20260810.md`.

- Backend: `backend/Dockerfile.dev` pakai [`air`](https://github.com/air-verse/air), source di-bind-mount (`./backend:/app`), Go module cache & `tmp` build dir pakai NAMED VOLUME.
- Frontend: image `node:20-alpine` polos, `npm install && npm run dev -- --host --port 5173` tiap start, `node_modules` pakai named volume.
- **Base image backend `golang:1.25-alpine`** (both `Dockerfile` dan `Dockerfile.dev`) — `github.com/xuri/excelize/v2` (fitur Import/Export) butuh Go 1.25 lewat transitive dep. Jangan downgrade kembali ke 1.22.
- **`vite.config.ts` proxy `/api` di-parameterize via `process.env.VITE_API_PROXY_TARGET`** (default `http://localhost:8080` buat dev lokal tanpa Docker). Di `docker-compose.dev.yml`, frontend container set `VITE_API_PROXY_TARGET=http://backend:8080` — kalau tetap hardcode `localhost:8080`, proxy dari dalam container nyasar ke dirinya sendiri.
- Port sama dengan versi build (`5432`/`8080`) — TIDAK bisa jalan bersamaan dengan `make up`, harus `make down` dulu.

## Arsitektur

Monolith modular, bukan microservices — satu binari Go, dipisah per domain secara logis via internal package, satu instance Postgres. Keputusan skala yang disengaja (`04-architecture.md` §1); jangan usulkan pemisahan jadi banyak service.

**Alur request:** Nginx → build statis SvelteKit (adapter-static, tanpa SSR) untuk `/`, reverse-proxy ke backend Go/chi untuk `/api/*`.

**Pola modul backend** (`backend/internal/<domain>/handler.go`): tiap domain punya struct `Handler` yang membungkus `*pgxpool.Pool`, dibuat via `NewHandler(pool)`, method-nya langsung dipasang ke route chi di `cmd/api/main.go`. Tidak ada layer service/repository — handler query `pgx` langsung. Semua 10 modul (`05-api-contract.md`) sudah diimplementasi penuh — modul baru di luar kontrak yang ada mengikuti pola yang sama (struktur file, gaya inline-SQL, bentuk struct request/response), dan pasang route baru ke grup `protected` di `main.go`.

**sqlc belum aktif.** Handler pakai query `pgx` mentah. `06-db-design.md` §5 mendokumentasikan pola query sqlc yang dituju sebagai referensi untuk migrasi nanti — jangan aktifkan tooling sqlc tanpa didiskusikan dulu, jangan asumsikan `.sql` di `backend/db/queries/` sudah terpakai (folder itu masih kosong).

**Field computed-at-read — jangan pernah disimpan sebagai kolom:** `actual_pct`, `expected_pct`, `verdict` (big task), `completion_rate`/`project_status` (board summary), `is_weekend` (day entry). Nilai-nilai ini bergantung pada waktu berjalan (`TODAY`) atau agregasi baris anak, wajib dihitung ulang tiap kali dibaca (`04-architecture.md` §5.1, `06-db-design.md` §1) — keputusan sadar untuk menghindari cron job harian. Field turunan baru ikuti pola ini.

**Logika verdict** (`backend/internal/bigtask/handler.go: computeVerdict`): status `on_progress` netral berapa pun gap actual vs expected, sampai ada titik keputusan (sign-off, atau deadline lewat tanpa sign-off). Signature `computeVerdict(deadline, signedAt *time.Time, now)` — kalau `signedAt != nil`, verdict BEKU (bandingin `deadline vs signedAt`, bukan `now`, selamanya); kalau belum signed, `deadline vs now`. (history: `docs/lessons-learned.md#verdict--sign-off`)

**Sign-off** (`big_task_signoffs`): keberadaan baris = signed. Undo = hapus baris tersebut (bukan flag status). Sign-off ditolak 409 kalau `actual_pct < 100` (BRD RULE-07).

**Backdate sign-off + edit Big Task (super_user only).** `PATCH /big-tasks/{id}/sign-off` terima body opsional `{signed_at: "YYYY-MM-DD"}` — hanya `access_level=super_user`, divalidasi: tidak boleh masa depan, tidak boleh sebelum `start_date`, tidak boleh sebelum tanggal `day_entries` terakhir. `big_task_signoffs.signed_at_backdated_by` diisi kalau backdated (indikator "(backdated)" di UI, JANGAN dihapus — audit trail). `PATCH /big-tasks/{id}` (super_user only) untuk edit `name`/`start_date`/`deadline`. Pola audit "siapa override": nullable FK `*_backdated_by`/`updated_by` ke `users`, BUKAN tabel audit log terpisah. (history: `docs/lessons-learned.md#verdict--sign-off`)

**Generate Daily Task → Day Entry**: insert satu baris `day_entries` per tanggal kalender dalam `[start_date, end_date]` inklusif, satu transaksi (SRS FR-DLY-01/02). `is_weekend` dihitung saat baca, tidak pernah disimpan.

**Day Entry — tidak dibatasi satu baris per tanggal.** `POST /daily-tasks/{id}/day-entries` (body `{entry_date, planned_text?}`) nambah baris baru di tanggal manapun; `DELETE /day-entries/{id}` hapus permanen. `actual_pct` dihitung dari SEMUA `day_entries` yang ADA saat dibaca. List di-`ORDER BY entry_date, created_at` (SENGAJA bukan `updated_at`). (history: `docs/lessons-learned.md#day-entry--progress`)

**Day Entry status 3-state granular.** `progress_pct` (smallint 0-100), BUKAN `is_done` boolean. State (Belum/On Progress/Selesai) diturunkan MURNI dari `progress_pct` (`0`→Belum, `1-99`→On Progress, `100`→Selesai), tidak disimpan kolom terpisah. `actual_pct` = `AVG(progress_pct)` di semua level. Logic turunan di `lib/dayProgress.ts` (+ test). (history: `docs/lessons-learned.md#day-entry--progress`)

**Auth**: JWT stateless access token (HS256, env `JWT_SECRET`, expiry 2 jam) via `Authorization: Bearer *** Refresh token JUGA JWT stateless terpisah (claim `typ:"refresh"`, expiry 7 hari), dikirim via cookie httpOnly (`Path=/api/v1/auth`, `SameSite=Lax`, `Secure` aktif kalau `APP_ENV=production`) — TIDAK ada tabel session/refresh-token di DB (`04-architecture.md` §6).

`auth.RequireAuth` (`backend/internal/auth/auth.go`) validasi access token, sisipkan `userID` + `roles` (dari claim JWT, bukan query DB) ke context. `auth.RequireRole("spv", ...)` middleware terpisah, dipasang SETELAH `RequireAuth`. Role di access token statis selama masa berlaku token (maks 2 jam) — perubahan role user baru kepakai setelah token refresh.

Frontend: `frontend/src/lib/stores/authStore.ts` — access token in-memory saja (bukan localStorage), sesi di-restore via silent refresh (`POST /auth/refresh`, cookie httpOnly). Info user non-sensitif di-cache `sessionStorage` cuma buat tampilan, bukan buat auth — `authStore.init()` SELALU fetch fresh `GET /users/me`, tidak pernah skip-fetch pakai cache. (history soal kenapa cache-first dihapus: `docs/lessons-learned.md#auth--session-staleness`)

**Root layout gating (`+layout.svelte`):** `<slot/>` HARUS selalu dibungkus div yang sama, di posisi `{#if/else}` yang sama, baik untuk halaman login maupun sudah-login — cuma `class` pada div pembungkus dan sibling-nya yang boleh toggle via `{#if}` terpisah. JANGAN taruh `<slot/>` sebagai satu-satunya isi cabang `{#if isLoginPage}` yang berbeda dari cabang `{:else}`. Bentuk final aman: satu `<div class="app" class:login-mode={isLoginPage}>` yang selalu ada. (history/regresi 2x: `docs/lessons-learned.md#auth`)

**Role** many-to-many via `user_roles` (bukan kolom tunggal di `users`) — kode role: `spv`, `sa`, `dev`, `qa`, `admin`.

**Migration**: file SQL bernomor di `backend/db/migrations/` (`NNNN_deskripsi.up.sql` / `.down.sql`), dijalankan via image Docker `migrate/migrate`.

**Frontend — struktur halaman kerja:** `routes/+page.svelte` (Dashboard, matriks summary per board via `GET /boards/{id}/summary`). `routes/boards/+page.svelte` = halaman kerja penuh: tab pilih board (state di URL via `?board=<id>`) → `lib/components/BigTaskList.svelte` (list + create Big Task, sign-off/undo-signoff hanya role `spv`) → expand baris → `lib/components/DailyTaskPanel.svelte` (list + create Daily Task, tabel Day Entry inline-editable, PIC picker dibatasi ke anggota Big Task via prop `members`). `lib/types.ts` = bentuk response API (jangan hitung ulang field turunan di sini). `lib/dateRange.ts` (+ test) = preview murni rentang tanggal, cerminan logika inklusif backend.

**Daily Task collapse + filter "sudah selesai"** — state UI murni (`localStorage`, key `rndops_daily_task_expanded_ids`), BUKAN kolom backend. Filter checkbox default OFF, `isOngoing(dt)` = pure function (`actual_pct !== 100 && end_date >= today`).

## Design System

Tema retro "Windows 9x/XP chrome" (4 tema), source of truth: `docs/rnd-ops-mockup_3.jsx`. Semua halaman sudah di-re-skin — **jangan bangun halaman baru dengan styling ad-hoc, selalu pakai token & class di bawah.**

**"Retro" itu VISUAL doang, bukan wajib retro di semua interaksi.** Token warna/font/border/chrome (titlebar, tabs, badge) tetap ikut retro theme, TAPI kalau ada pola interaksi MODERN yang lebih jelas buat user (mis. toggle switch buat boolean), pakai itu. Kalau ragu "masih dalam batas retro-visual atau kejauhan", tanya user. (history/regresi: `docs/lessons-learned.md#frontendsvelte-gotchas`)

**Token & komponen dasar** (`frontend/src/app.css`): custom properties per tema via `:root[data-theme="retro-light|retro-dark|modern-light|modern-dark"]`. Named vars: `--face`, `--content-bg`, `--content-alt`, `--text-primary`, `--text-muted`, `--titlebar-a/b`, `--win-blue/-light`, `--win-green`, `--win-red`, `--win-amber`. Font: `Tahoma, 'MS Sans Serif', 'Segoe UI', Arial, sans-serif`, ukuran dasar 11px — SEMUA komponen baru wajib ikut ini.

**Tema disimpan di `users.theme_preference`**, state UI di `lib/stores/themeStore.ts`. Ganti tema = `SettingsModal` panggil `PATCH /users/me {theme_preference}` lalu `auth.refreshUser()`.

**Komponen reusable** (`lib/components/`): `Avatar.svelte`, `VerdictBadge.svelte`, `DualBar.svelte`, `StatCard.svelte`, `DonutChart.svelte`, `GroupedBarChart.svelte` — **chart custom SVG, BUKAN library** (tidak ada versi Svelte resmi untuk Recharts, kebutuhan chart simpel). Chart baru yang jauh lebih kompleks = alasan valid untuk evaluasi ulang.

**`.app` HARUS fluid, JANGAN dikasih `max-width` + `margin:auto`.** Root cause table/section sparse di-fix DI ELEMEN ITU SENDIRI (mis. `.weekplan-table { max-width: 900px; }`), bukan di container `.app`. (history: `docs/lessons-learned.md#frontendsvelte-gotchas`)

**My Profile & Settings = modal**, bukan route halaman (`lib/components/ProfileModal.svelte`, `lib/components/SettingsModal.svelte`), dipanggil dari dropdown avatar user di topbar. `SettingsModal` punya 2 tab: "Manajemen user" (admin/spv only) dan "Tema aplikasi" (semua role).

## Dashboard

**POV per BOARD (=project), bukan per Big Task.** Diagregasi di CLIENT dari `GET /boards` + `GET /boards/{id}/big-tasks` tiap board (`Promise.all`), TIDAK ada endpoint backend baru. Agregasi 2 tingkat di `lib/dashboardStats.ts`: `aggregateBoards()` (Big Task → `BoardAgg` per board) lalu `computeDashboardStats()` (`BoardAgg[]` → statistik portfolio). Rule status board: **all-or-nothing per bucket, default "running" kalau campuran** — `done` HANYA kalau SEMUA Big Task signed, `hold` HANYA kalau SEMUA on_hold, `not_started` HANYA kalau SEMUA 0%/belum punya Big Task. Rule verdict board: **asimetris** — `lose` kalau ADA MINIMAL SATU Big Task lose, `won` HANYA kalau status `done` DAN tidak ada lose sama sekali. Metrik dashboard baru ikuti pola agregasi-ke-level-board ini. (history: `docs/lessons-learned.md#dashboard-aggregation`)

**`activeBoards`** (dipakai di `progressChartData`, attention-list, `nearestDeadline`) = SEMUA board KECUALI `status==='done'`. Section dashboard baru yang nge-list board: pikirkan dulu status mana yang relevan, jangan default ke "cuma running". (history: `docs/lessons-learned.md#dashboard-aggregation`)

**Warna identitas board di chart**: `boardColor(boardId, dark)` di `dashboardStats.ts` — warna di-assign via HASH deterministik dari `boardId` (bukan index/urutan tampil), board yang sama SELALU dapat warna sama. Palette 8-hue categorical dari skill `dataviz` (light+dark variant terpisah). Chart baru yang butuh identitas warna per-entity: pakai pola hash-based ini, cek skill `dataviz` dulu. (history: `docs/lessons-learned.md#dashboard-aggregation`)

**"Tim" grid** cuma nampilin identitas+role dari `GET /users/assignable` — TIDAK ada hitungan "task aktif"/"belum direview" per orang (butuh cross-reference mahal + Review Queue) — jangan tambah balik tanpa endpoint agregat backend proper.

**Board description** ditampilkan sebagai banner (`.board-desc-banner`, `white-space: pre-wrap`) di atas `<BigTaskList>` kalau non-kosong. Input via `<textarea class="inline-input inline-textarea">`.

**Klik nama board = navigasi ke `/boards?board=<id>`** — dipakai di semua tempat nama board muncul di Dashboard.

## Comments, Cheat Sheet, Upload

**Comments (Fase 3):** `backend/internal/comment/handler.go` — `GET /big-tasks/{id}/comments?scope=all|general|{daily_task_id}`, `POST` (author_id selalu dari JWT). Render WAJIB via `lib/comments.ts` (`renderCommentHtml`), jangan `{@html}` body mentah — escape HTML dulu, baru mention (`@Nama`) dibungkus `<span class="mention-tag">` (urutan ini yang bikin aman dari XSS).

**Cheat Sheet & Upload (Fase 4):** `backend/internal/cheatsheet/` — CRUD referensi board (`file`/`url`/`note`), scope BOARD. `backend/internal/upload/` — `POST /uploads` (multipart, field `file`) simpan ke disk lokal (`UPLOAD_DIR`, default `/app/uploads`, **wajib docker volume** `uploads`), nama `<uuid>_<nama-asli-disanitasi>`. `GET /uploads/{filename}` untuk ambil lagi.

**Link download file TIDAK BOLEH `<a href>` biasa** — browser tidak menempelkan header custom ke navigasi `<a href>`, access token in-memory bukan cookie. Wajib pakai `lib/api/client.ts` `downloadBlob(path)` (fetch manual + header Authorization → `Blob` → object URL sementara + `<a>` sintetis via JS). Pola ini wajib untuk resource lain yang butuh auth + didownload langsung browser.

**Cheat Sheet Edit & Delete (super_user only).** `cheatsheet.Update`/`Delete` cek `auth.IsSuperUser(ctx)` in-handler (bukan `RequireRole`). Upload pakai `api.upload()` (`FormData`, JANGAN set `Content-Type` manual).

## Clone-as-review & Review Queue

**Clone-as-review:** `dailytask.CloneReview` — `POST /daily-tasks/{id}/clone-review` body `{reviewer_user_id, start_date, end_date}`, bikin Daily Task baru beneran. Reviewer WAJIB anggota Big Task (400 kalau bukan). Judul otomatis `"[Review {display_name}] {judul asal}"`, PIC = reviewer, menyimpan `review_of_daily_task_id`.

**Review Queue:** `backend/internal/reviewqueue/` — `GET /review-queue`, `POST /review-queue/{item_type}/{item_id}/mark-reviewed` (`item_type = "daily_task"` saja). Isi queue = task review (`review_of_daily_task_id != NULL`) yang PIC-nya = requesting user & belum ada di `item_reviews`. `MarkReviewed` otorisasi: requesting user harus PIC dari task review itu. `item_reviews`: keberadaan baris = sudah ditinjau, idempoten `ON CONFLICT (item_type, item_id)`.

**Anggota Big Task:** tabel `big_task_members (big_task_id, user_id)` = siapa saja yang terlibat (bukan khusus reviewer). `bigtask.BigTask.MemberUserIDs []string`. **WAJIB min. 2 anggota** saat create. Editable via `PUT /big-tasks/{id}/members` (replace-set, tetap min-2). **PIC Daily Task WAJIB anggota** (validasi di `Create` + `CloneReview`). Reviewer clone-review juga wajib anggota.

**Frontend Review Queue** pakai shared store `lib/stores/reviewQueueStore.ts` (dipakai bareng `+layout.svelte` badge notifikasi dan `routes/review-queue/+page.svelte`) — dipakai karena dua route sibling tidak bisa saling dengar event Svelte langsung.

## Weekly Plan & HR Integration

**Weekly Plan:** `backend/internal/weeklyplan/` — `GET /weekly-plan?week_start=YYYY-MM-DD` (rollup lintas board, INNER JOIN `day_entries` di kondisi JOIN bukan WHERE), `POST /weekly-plan/push` (upsert `weekly_push_log` via `ON CONFLICT (big_task_id, week_start)`). "My Weekly Plan" = laporan PERSONAL, `List`/`Push` di-JOIN `dt.pic_user_id = <user login>` — Big Task multi-PIC muncul beda-beda per orang, itu WAJAR. **Aturan umum: endpoint baru dengan konteks personal ("My ...") tidak otomatis ter-filter per-user cuma karena butuh auth token — query harus eksplisit `WHERE`/`JOIN ... = userID`.** (history: `docs/lessons-learned.md#weekly-plan`)

**Super User bisa lihat/push Weekly Plan siapapun**: `as_user_id` opsional, CUMA dihormati kalau `access_level=super_user`, else 403. `weekly_push_log.pushed_by` SELALU actor, BUKAN `as_user_id` target. `GET /weekly-plan/team-status` (super_user only).

**Integrasi HR — `myagenda-service`** (sub-project TERPISAH, dummy/lokal saat ini): skema `my_agenda` (MySQL) PERSIS DDL dari sistem HR asli, TIDAK BOLEH diubah. `POST /my-agenda` upsert APLIKASI-LEVEL berdasarkan `(user_id, judul_task, tgl_rencana)` (SELECT dulu, lalu INSERT/UPDATE — bukan `ON DUPLICATE KEY`). `weeklyplan.Push` manggil service ini SETELAH `weekly_push_log` sukses — BEST-EFFORT, gagal connect TIDAK membatalkan response 200. **`user_id` yang dikirim MASIH PLACEHOLDER** (CRC32 dari UUID kita, bukan id pegawai HR asli) sampai `users.hr_user_id` ter-mapping via Manajemen User, lalu pakai `resolveHRUserID` asli. Di dev, backend reach service via `host.docker.internal` (`extra_hosts: host-gateway`).

## Access Level & HR Mapping

`users.access_level` (`super_user`/`regular_user`, default `regular_user`) — SENGAJA kolom tunggal (exception dari aturan role many-to-many, lihat "Jangan lakukan"), masuk JWT claims. `referensi_tim` dan `referensi_user_hr` adalah tabel referensi (data MILIK sistem HR asli, TIDAK ADA CRUD UI). `users.hr_user_id` (nullable, unique FK) di-mapping lewat Manajemen User.

**`GET /referensi-tim` di grup `protected` biasa** (bukan `RequireRole`) — dibutuhkan SEMUA authenticated user untuk filter tim Dashboard; mutasi (POST/PATCH/DELETE) tetap admin-only. `access_level` adalah axis TERPISAH dari `roles`. (history: `docs/lessons-learned.md#board-category--team-scoping`)

## Import / Export & Board Archive

**Import/Export (super_user only):** `backend/internal/dataport/` — `GET /admin/export` (XLSX 4 sheet: Boards, BigTasks, DailyTasks, DayEntries), `POST /admin/import` (multipart, return JSON stats). Import resolusi user via email; PIC/anggota tidak ditemukan → skip/diabaikan + warning. Frontend `api.upload()` untuk multipart (bukan `api.post`). Library: `github.com/xuri/excelize/v2`.

**Board Archive (super_user only):** `boards.archived_at`/`archived_by` (existence-pattern). `board.List` filter `WHERE archived_at IS NULL` — Dashboard & tab Boards otomatis konsisten. **Weekly Plan & Review Queue SENGAJA TIDAK difilter** (JOIN langsung ke `boards`/`daily_tasks`) — laporan personal & antrean review adalah level TASK, tidak boleh "hilang" cuma karena project-nya diarsipkan.

**Board Category & Team Scoping:** `boards.category` (`project`/`routine`, nullable, tanpa default/backfill). Semua user set `category` saat create; hanya super_user ubah kategori board existing. `board_teams (board_id, team_id)` many-to-many. `board.Create` auto-insert baris `board_teams` untuk `org_team` pembuat. **Visibility rule**: regular user cuma lihat board dengan `board_teams` yang match `org_team`-nya sendiri; super_user tidak dibatasi, dapat `?team_id=` opsional. Keduanya terima `?category=`.

## Claude Chat Bridge — `claude-chat-service`

Sub-project di root repo, Node.js/TypeScript + **Claude Agent SDK** (bukan shell-out CLI mentah). Arsitektur lengkap: `docs/decision-log/decision-log-claude-chat-service-20260811.md`. Endpoint & WS schema: `claude-chat-service/docs/api_contract.md`. Panduan integrasi: `claude-chat-service/docs/integrations_guide.md`.

**Ringkasan keputusan kunci** (jangan implementasi menyimpang tanpa diskusi ulang):
- **Deployment**: server bersama, SATU akun Claude subscription (OAuth) dipakai bersama semua sesi.
- **TIDAK ADA RBAC/auth di service ini sendiri** (sengaja, sementara) — full trust ke pemanggil, otorisasi didelegasikan ke `rnd-task-board` backend.
- **`cwd` wajib dipilih eksplisit** sebelum chat mulai — validasi ketat (tolak `$HOME`/root/`CWD_FORBIDDEN_ROOTS`).
- **Session manager internal** — satu `Query` SDK hidup terus (via `AsyncQueue`), bukan spawn ulang tiap prompt.
- **Resume** = field `resume` di `POST /sessions` (bukan endpoint terpisah).
- **Login via `claude setup-token`** (PTY, session-based + input channel terpisah karena butuh authorization code hasil approve browser di-paste balik) — BUKAN `claude login` interaktif biasa, itu tidak dikonsumsi service untuk auth jangka panjang. Token OAuth ~1 tahun, auto-load tiap start dari `TOKEN_STORE_PATH`. (history: `docs/lessons-learned.md#claude-chat-bridge`)
- **`total_cost_usd` in-memory** — reset tiap service restart, belum ada persistent storage.

**Gotcha belum diverifikasi:** `extractToken()` di `src/auth/setupTokenBridge.ts` belum divalidasi terhadap output CLI asli — cek/sesuaikan regex parsing saat dites langsung pertama kali.

**UI trigger di `rnd-task-board`:** Settings → tab "Login Claude" (super_user only). Gating akses ditaruh di `backend/internal/chatproxy/handler.go` (`isSetupTokenPath` + cek `claims.AccessLevel != "super_user"` → 403), BUKAN cuma disembunyikan di UI — proxy Go adalah satu-satunya titik enforcement karena service sendiri no-auth. Sub-route sensitif baru: ikuti pola path-based gate di proxy ini.

**Koneksi = backend proxy.** `backend/internal/chatproxy/` reverse-proxy `/api/v1/chat/*` → `CLAUDE_CHAT_SERVICE_URL` (dev: `host.docker.internal:8091`, sengaja beda dari myagenda `:8090`). Frontend tidak pernah connect langsung ke service. `httputil.ReverseProxy` menangani REST DAN WebSocket sekaligus.

**Auth WS via query param, bukan header** — browser `WebSocket` tidak bisa set header `Authorization`; proxy terima token via `?access_token=` (di-strip sebelum diteruskan). REST tetap header `Authorization: *** Divalidasi via `auth.ParseAccessToken` (dipakai bareng, ada test). `vite.config.ts` proxy `/api` WAJIB `{ ws: true }`.

**Permission mode chat change-request = `plan` (read-only), SENGAJA.** Assistant boleh baca repo tapi tidak menulis/eksekusi — batasi blast radius karena service masih no-auth. JANGAN naikin ke `bypassPermissions` tanpa auth peer-to-peer dulu.

**`change_requests`:** `backend/internal/changerequest/` — `GET /change-requests` (semua user login), `POST /change-requests` (submit), `PATCH /change-requests/{id}` (triase, `RequireRole("spv","sa")`). `document_md` (markdown inline, terpisah dari `raw_conversation`) diisi otomatis saat assistant selesai turn "Susun change request". Render WAJIB via `renderMarkdown()` (marked + DOMPurify) — JANGAN `<pre>` plain text. (history: `docs/lessons-learned.md#change-request-document`)

**Sesi chat PERSIST lintas navigasi menu** — state di store level modul `lib/stores/chatSessionStore.ts`, BUKAN di komponen (`onDestroy` akan nutup WS + DELETE sesi tiap pindah menu kalau ditaruh di komponen). Sesi ditutup lewat aksi EKSPLISIT saja: tombol "Akhiri sesi" atau logout.

**`claude-chat-service` dijalankan DI HOST untuk dev lokal**, bukan via docker-compose-nya sendiri — service baca `CHAT_DEFAULT_CWD` langsung dari filesystem host, dan `claude` CLI di host sudah authenticated jadi Agent SDK pakai kredensial itu langsung. Cara jalanin: `cd claude-chat-service && PORT=8091 node dist/index.js`. (history: `docs/lessons-learned.md#claude-chat-bridge`)

**Lampiran gambar/screenshot di chat**: prompt WS bawa `{type:'prompt', text, images:[{media_type, data}]}` (`data` = base64 mentah tanpa prefix). Validasi tipe (png/jpeg/gif/webp) + guard 5MB/gambar.

## Documentation Maintenance Protocol

Saat mengubah kode, dokumen terkait wajib ikut di-update DI SESI YANG SAMA — jangan ditunda:

| Perubahan | Dokumen yang wajib diupdate |
|---|---|
| Endpoint/route baru atau berubah | `docs/05-api-contract.md` |
| Entity/schema/migration baru | `docs/06-db-design.md` |
| Keputusan arsitektur final | `docs/decision-log/decision-log-[slug]-YYYYMMDD.md` |
| Pola/konvensi baru yang berlaku ke depan | Bagian relevan di `CLAUDE.md` ini |
| Insight non-obvious ditemukan (gotcha, quirk data, status/enum meaning) | `docs/lessons-learned.md` |
| Blocker eksternal (butuh input manusia/sistem lain) | `docs/blocker-logs/{topik}-YYYYMMDD-blocker.md` |
| Akhir sesi kerja substansial | `docs/progress-log/progress-log-YYYYMMDD.md` |

## Blocker Log

Kalau ada blocker (butuh input eksternal, dependency belum ada, ambiguitas requirement yang tidak bisa diputuskan sendiri):

```
docs/blocker-logs/{topik}-YYYYMMDD-blocker.md
```

Isi minimal: **Deskripsi** (apa yang jadi blocker), **Dampak** (bagian mana yang ter-block), **Opsi Solusi** (1-2 opsi), **Yang Dibutuhkan** (apa yang perlu dari manusia/sistem lain). Setelah resolve, tambahkan section `## Resolution` — jangan hapus file-nya.

## Progress Log

Di akhir sesi kerja substansial, tulis ringkas ke:

```
docs/progress-log/progress-log-YYYYMMDD.md
```

Isi: apa yang selesai, deviasi dari plan awal (kalau ada), masalah yang ditemukan. Ringkas, bukan naratif panjang — detail teknis lengkap ada di decision-log/lessons-learned.md yang direferensikan.

## Definition of Done

Sebelum melaporkan task/fitur selesai, cek:

- [ ] Test relevan lulus (`go test ./...` dan/atau `npm run check`/`npm run test` sesuai bagian yang disentuh) — lihat bagian Testing.
- [ ] Dokumen terkait sudah di-update sesuai tabel di Documentation Maintenance Protocol.
- [ ] Kalau ada keputusan arsitektur baru → decision-log sudah ditulis.
- [ ] Tidak ada credential/secret yang ke-hardcode atau ter-commit.
- [ ] Kalau menyentuh dev-only bypass/shortcut baru → sudah ditandai eksplisit di bagian Security Notes.
- [ ] Kalau ada insight non-obvious yang ditemukan selama kerja → sudah masuk `docs/lessons-learned.md`.

## Security Notes

- **Jangan pernah commit `.env`/file kredensial.** Pastikan tetap di `.gitignore`.
- Kredensial dev (`spv@rndops.local / password123` — lihat Commands di atas) HANYA untuk seed data development, jangan pernah dipakai sebagai kredensial default di production.
- Access token disimpan in-memory di frontend (bukan localStorage) by design — jangan ubah ke penyimpanan persistent tanpa alasan kuat (lihat bagian Auth).
- `claude-chat-service` sengaja **TIDAK ADA RBAC/auth sendiri** (full-trust ke pemanggil) — kalau nambah sub-route sensitif baru di service itu, WAJIB digate di proxy Go (`chatproxy/handler.go`), bukan cuma disembunyikan di UI. Lihat bagian Claude Chat Bridge.
- Kalau nemu/menambahkan dev-only bypass baru (auth yang di-skip, hardcoded user, `permitAll()`-style shortcut) di proyek ini — dokumentasikan eksplisit di section ini, jangan biarkan tersembunyi diam-diam di kode.

## Jangan lakukan

- Jangan simpan `verdict`, `expected_pct`, `actual_pct`, `completion_rate`, atau `is_weekend` sebagai kolom tabel — wajib tetap computed-at-read.
- Jangan tambahkan kolom `role` tunggal di `users` — role itu many-to-many via `user_roles`. **Exception sadar**: `users.access_level` (`super_user`/`regular_user`) MEMANG kolom tunggal — keputusan eksplisit user, konsepnya beda dari `roles` (saling eksklusif, bukan bisa dirangkap). Jangan tiru pola ini buat konsep baru tanpa alasan setara — kalau ragu, tanya user dulu.
- Jangan ganti auth ke cookie/session — ini API stateless Bearer-JWT by design.
- Jangan tambahkan layer abstraksi service/repository di handler backend — modul yang sudah ada query `pgx` langsung, modul baru harus ikut pola itu.
- Jangan hapus `preprocess: vitePreprocess()` di `frontend/svelte.config.js` — tanpa itu `npm run build` gagal total begitu ada sintaks TypeScript di dalam `<script lang="ts">` manapun.
- Jangan taruh `<slot/>` root layout di cabang `{#if}` terpisah per state auth — lihat bagian Arsitektur.
- Jangan bikin halaman baru dengan styling ad-hoc — pakai token & class di `app.css` (lihat Design System).
- Jangan tambah library chart (Recharts/Chart.js/dst) — donut & grouped bar sudah di-cover custom SVG. Lihat decision log kalau kebutuhannya berubah jauh lebih kompleks.
- Jangan bikin ulang halaman/route buat "My Profile" atau "Settings" — itu sudah jadi modal, bukan route SvelteKit.
