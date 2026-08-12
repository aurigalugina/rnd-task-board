# Decision Log — Integrasi Change Request §6 (Backend Proxy + Alur Percakapan)

**Tanggal:** 2026-08-11
**Konteks:** change-request-integration

## Konteks/Masalah

`01-vision-product.md` §6 ("Mekanisme Evolusi Produk") mendeskripsikan jalur bagi SELURUH anggota tim (bukan cuma SPV) untuk mengusulkan perubahan/penambahan fitur lewat **percakapan bebas** yang dibantu agentic coding assistant, lalu disusun jadi `change_request.md` terstruktur, untuk ditriase oleh SPV & System Analyst (`sa`) dan dijadwalkan ke siklus berikutnya. Fondasi service-nya (`claude-chat-service`) sudah diimplementasi & smoke-tested (lihat `decision-log-claude-chat-service-20260811.md`), dan tabel `change_requests` sudah disiapkan sejak migration `0006`. Yang BELUM diputuskan (ditandai eksplisit sebagai "sengaja ditunda" di decision log service): **bagaimana `rnd-task-board` konek ke service, dan bentuk alur change-request di aplikasi.** Itu yang diputuskan di sini.

## Keputusan

### 1. Koneksi: Backend Proxy (Opsi B dari `integrations_guide.md`)

`rnd-task-board` backend (Go/chi) mem-**reverse-proxy** `/api/v1/chat/*` ke `claude-chat-service` (`CLAUDE_CHAT_SERVICE_URL`, default `http://localhost:8090`). Frontend cuma tahu satu base URL (`/api/v1`), tidak pernah connect langsung ke `:8090`.

- **REST** (`/api/v1/chat/sessions`, `/fs/browse`, dll) diproksikan dengan strip prefix `/api/v1/chat` → diteruskan ke root service.
- **WebSocket** (`/api/v1/chat/ws/sessions/:id`) diproksikan lewat mekanisme `httputil.ReverseProxy` yang sama (Go menangani `Connection: Upgrade` sebagai tunnel sejak 1.12).
- **Auth** diberlakukan DI PROXY (bukan di service — service memang no-auth by design). REST pakai header `Authorization: Bearer` seperti endpoint lain. WS pakai **query param `?access_token=`** karena browser `WebSocket` tidak bisa menyetel header custom (token disimpan in-memory, bukan cookie — lihat gotcha download-file di CLAUDE.md, masalah serupa). Modul proxy melakukan validasi JWT sendiri untuk kedua jalur via `auth.ParseAccessToken`.
- Endpoint intercept lokal `GET /api/v1/chat/local/config` (TIDAK diteruskan ke service) mengembalikan `{default_cwd}` dari env `CHAT_DEFAULT_CWD` supaya frontend tidak meng-hardcode path repo.

**Alasan pilih Opsi B**: otorisasi "siapa boleh buka sesi / pakai bypass" adalah tanggung jawab `rnd-task-board` (didelegasikan eksplisit di decision log service). Proxy = satu gerbang tempat `RequireAuth`/role check bisa ditempel, reuse JWT yang sudah ada, dan frontend tidak perlu tahu origin kedua (tidak perlu CORS lintas-origin ke `:8090`). Trade-off: perlu nulis WS proxy di Go — diterima.

### 2. Permission mode default: `plan` (read-only)

Sesi chat change-request dibuat dengan `permissionMode: "plan"` — assistant boleh BACA repo (biar usulannya kontekstual & realistis) tapi TIDAK menulis/eksekusi apa pun. Draft `change_request.md` dihasilkan sebagai **teks di dalam percakapan**, lalu disimpan aplikasi ke baris `change_requests` (bukan assistant nulis file langsung).

**Alasan**: (a) service masih no-auth (peer-to-peer auth ditunda) — batasi blast radius, jangan kasih jalur `bypassPermissions` dulu dari fitur yang dipakai semua anggota tim; (b) §6 tujuannya menyusun DOKUMEN usulan untuk ditriase manusia, bukan langsung ngoding — read-only sudah cukup; (c) konsisten dengan prinsip produk #2 ("approval = atensi, bukan gerbang") dan filosofi tidak merusak apa pun tanpa keputusan manusia.

### 3. Otorisasi change_requests

- **Submit (`POST /change-requests`)** & **List (`GET /change-requests`)**: semua user terautentikasi. §6 eksplisit "seluruh anggota tim — bukan hanya SPV". List dibuka ke semua demi transparansi (tim kecil), `submitted_by` selalu dari JWT.
- **Triase (`PATCH /change-requests/{id}`)**: hanya `spv` + `sa` (RequireRole), sesuai §6 "ditriase oleh SPV dan System Analyst". Mengubah `status` (pending→approved/rejected/scheduled) + set `reviewed_by` (dari JWT) & `reviewed_at`.

### 4. Transisi status

`pending` adalah state awal. Triase boleh pindah ke `approved` / `rejected` / `scheduled`. Boleh balik ke `pending` (batal triase). Semua target harus salah satu dari 4 enum di CHECK constraint migration `0006`. Validasi transisi diekstrak ke fungsi murni `isValidStatusTransition` (+ test) — bukan sekadar cek enum, tapi juga cegah transisi tak masuk akal kalau ada (saat ini semua transisi antar 4 state diizinkan; fungsi disiapkan supaya mudah diperketat nanti).

### 5. cwd dari konfigurasi backend, BUKAN dipilih user

Direktori project sesi = `CHAT_DEFAULT_CWD` (env backend, didokumentasikan di `backend/.env.example`; di dev = path repo `rnd-task-board`). **Diketahui dari awal — user TIDAK memilih direktori di UI.** Frontend ambil nilai ini via `GET /api/v1/chat/local/config` dan menampilkannya read-only, langsung ke tombol "Mulai percakapan". Dir picker (`/fs/browse` yang diproksikan) cuma FALLBACK kalau `CHAT_DEFAULT_CWD` belum di-set (config balikin string kosong) — supaya fitur tetap bisa jalan sambil nunggu admin set env, bukan buat dipakai rutin.

**Alasan (revisi dari draft awal yang sempat menampilkan input path + picker sebagai jalur utama):** target repo change request HAMPIR SELALU produk ini sendiri; memaksa user mengetik/menavigasi path tiap kali adalah friksi tak perlu dan rawan salah. Path adalah keputusan deployment (di mana `claude-chat-service` membaca repo), bukan pilihan per-percakapan — jadi tempatnya di `.env` backend, sekali set.

## Dampak/File Terpengaruh

- `backend/internal/chatproxy/` — modul baru: reverse proxy REST+WS, auth header/query, intercept `/local/config`. Pure: `rewriteChatPath`, `tokenFromRequest` (+ test).
- `backend/internal/auth/auth.go` — ekstrak `ParseAccessToken(tokenString) (*AccessClaims, error)` (dipakai `RequireAuth` DAN WS proxy), + `AccessClaims` struct. Testable (security-critical).
- `backend/internal/changerequest/` — modul baru: List/Create/Update, `isValidStatusTransition` (+ test).
- `backend/cmd/api/main.go` — wire `/chat/*` (auth internal proxy) + `/change-requests` (protected; PATCH digate spv+sa).
- `frontend/src/lib/chatClient.ts` — helper REST + WS URL builder (+ inject access_token). `lib/chatMessages.ts` — ekstrak teks dari SDK message (pure, tested).
- `frontend/src/routes/change-requests/+page.svelte` — list + triase + tombol "Ajukan usulan".
- `frontend/src/lib/components/ChangeRequestChat.svelte` — panel chat (dir picker, streaming, simpan CR).
- `frontend/vite.config.ts` — proxy `/api` set `ws: true`.
- `docker-compose.dev.yml` — `CLAUDE_CHAT_SERVICE_URL` + `CHAT_DEFAULT_CWD` ke backend.
- `CLAUDE.md` — ringkasan integrasi + gotcha WS-auth-query-param.

## Update (2026-08-11) — Sesi chat persist lintas navigasi menu

**Masalah:** implementasi awal menaruh state sesi (session id, WebSocket, messages, cost) di dalam komponen `ChangeRequestChat.svelte` dengan `onDestroy(() => cleanup())` yang close WS + `DELETE /sessions/:id`. Akibatnya begitu user pindah menu (komponen unmount), sesi langsung mati — padahal user cuma mau lihat menu lain sebentar.

**Keputusan:** angkat SELURUH state sesi ke store level modul `lib/stores/chatSessionStore.ts` (hidup di luar lifecycle komponen, mirip pola `reviewQueueStore`). Komponen jadi "view" murni yang subscribe ke store; `onDestroy` TIDAK lagi menutup apa pun. Sesi baru benar-benar ditutup lewat aksi EKSPLISIT:
- Tombol **"Akhiri sesi"** (`closeSession()`) — close WS + `DELETE` sesi di service + reset store ke `idle`.
- **Logout** (`authStore.logout` panggil `resetAll()`) — supaya sesi tidak menggantung setelah ganti user.

Visibility panel di halaman digerakkan oleh `$chatSession.step !== 'idle'` (bukan flag lokal halaman) — jadi navigasi keluar/masuk `/change-requests` otomatis menampilkan lagi panel + history. Indikator titik hijau (`.chat-live-dot`) di tab "Change request" nandain sesi masih hidup dari menu mana pun.

**Batasan sadar:** persist hanya lintas navigasi SPA (client-side). Full page reload tetap mereset store (WS mati) — dan sesi di service juga in-memory, jadi ini konsisten. `onWsClose` yang tak disengaja (service restart) TIDAK menghapus history — user tetap bisa baca & simpan transcript, cuma dikasih error banner.

**Logic murni yang ditest** (side-effect WS tidak ditest, tapi reduksinya iya): `appendChatEvent` (gabung teks assistant streaming, immutable) & `buildTranscript` di `lib/chatMessages.ts`.

**File terpengaruh tambahan:** `lib/stores/chatSessionStore.ts` (baru), `lib/components/ChangeRequestChat.svelte` (jadi thin view), `routes/change-requests/+page.svelte` (visibility dari store), `lib/stores/authStore.ts` (logout → resetAll), `routes/+layout.svelte` + `app.css` (indikator `.chat-live-dot`).

## Update (2026-08-11) — Lampiran gambar/screenshot

**Kebutuhan:** user mau melampirkan screenshot saat mengusulkan perubahan (mis. "tombol ini kurang kontras" + gambar). Claude multimodal, jadi feasible.

**Keputusan:** perluas protokol prompt WS jadi `{type:'prompt', text, images?:[{media_type, data}]}` (`data` = base64 mentah). `claude-chat-service.sendPrompt` menyusun content block array Anthropic (`text` + `image` base64) kalau ada gambar; tanpa gambar tetap kirim string (backward-compatible). Backend proxy tidak berubah (WS tunnel raw). Frontend: tombol 📎 (file picker) + paste clipboard (screenshot lazimnya di-paste), preview + hapus, render di bubble user; validasi tipe png/jpeg/gif/webp + guard 5MB. Draft yang disimpan (`buildTranscript`) menandai `_[N gambar dilampirkan]_` (gambar sendiri tidak dimasukkan ke `raw_conversation` — cuma teks + penanda, biar baris DB tidak membengkak base64; kalau nanti perlu simpan gambarnya, arahkan ke modul `upload` yang sudah ada).

**Diverifikasi end-to-end live:** kirim PNG merah via WS proxy → Claude jawab "Merah". Rebuild+restart `claude-chat-service` diperlukan karena menyentuh sub-project itu.

## Belum diputuskan / ditunda (konsisten dgn decision log service)

- Auth peer-to-peer `rnd-task-board` ↔ `claude-chat-service` (service masih full-trust; proxy jadi satu-satunya penjaga — cukup untuk sekarang karena service hanya di-reach via proxy di jaringan internal).
- Persist akumulasi `total_cost_usd` (masih in-memory di service).
- Membiarkan assistant MENULIS `change_request.md` langsung ke repo (butuh `bypassPermissions` + auth p2p dulu) — sekarang draft disimpan sebagai teks di DB.
