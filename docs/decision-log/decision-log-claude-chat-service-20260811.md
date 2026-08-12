# Decision Log — Sub-Project `claude-chat-service` (Bridge Claude Code, Vision Product §6)

**Tanggal:** 2026-08-11
**Konteks:** claude-chat-service

## Konteks/Masalah

`01-vision-product.md` §6 ("Mekanisme Evolusi Produk") menyebutkan portal ini akan menyediakan jalur bagi seluruh anggota tim untuk mengusulkan perubahan lewat percakapan bebas, dibantu "agentic coding assistant yang ditanam di repository project", lalu disusun jadi `change_request.md` terstruktur. Belum ada infrastruktur apapun buat itu — user minta mulai bangun fondasinya: sebuah service yang menjembatani Claude Code (jalan di level project/repo) dengan client aplikasi (chat UI), scope-nya sengaja dibuat general-purpose (bukan cuma buat capture change request) supaya berguna juga untuk kebutuhan lain: pilih/point direktori kerja, buka sesi chat, toggle skip-permissions/bypass-permissions, resume/continue sesi, pilih model, cek estimasi pemakaian token, plan mode, dan chat-only mode.

## Keputusan

**Struktur & stack**: sub-project baru `claude-chat-service/` di root repo (sibling `backend/`/`frontend`/`myagenda-service/`), Node.js + TypeScript, pakai **Claude Agent SDK** (bukan shell-out ke CLI mentah) supaya dapat `canUseTool` callback & streaming event untuk di-bridge ke WebSocket. `docker-compose.yml`/`package.json`/`README.md` sendiri, terpisah total dari stack Go rnd-task-board — konsisten dengan precedent `myagenda-service` (pilih stack sesuai kebutuhan sub-project, bukan dipaksa ikut pola monolith Go).

**Deployment model**: server bersama (bukan instance per-developer lokal), **satu akun Claude subscription (Pro/Max, OAuth)** dipakai bersama oleh seluruh sesi/user — bukan API key billing per-token.

**Tanpa RBAC/auth di level service ini.** Service ini murni bridge — mempercayai penuh siapapun yang berhasil reach dia (termasuk permintaan `bypassPermissions`/skip-permissions). Otorisasi "siapa boleh apa" adalah tanggung jawab `rnd-task-board` backend (consumer/"third app") yang memanggil service ini. Auth service-to-service (peer-to-peer) antara `rnd-task-board` dan `claude-chat-service` **sengaja ditunda**, bukan bagian dari scope sekarang.

**Directory (cwd) wajib dipilih eksplisit oleh user sebelum chat session dimulai** — bukan default/opsional. Tidak ada flag native Claude Code untuk targeting directory arbitrary saat runtime; `cwd` di-set sendiri oleh service saat spawn Agent SDK session. Validasi cukup ringan (tolak path yang jelas terlalu luas, mis. `$HOME` murni atau root filesystem), **bukan** allowlist konfigurasi path yang di-maintain manual.

**Session manager internal**: key by `session_id` milik service sendiri (bukan langsung Claude session id), menyimpan `{cwd, model, permission_mode, claude_session_id, akumulasi total_cost_usd}`. Permukaan API: REST untuk control-plane (create/get/resume/delete session), WebSocket untuk streaming chat per session.

**Mapping fitur ke opsi Agent SDK**:
| Fitur | Opsi SDK |
|---|---|
| Skip-permissions / bypass mode | `permissionMode: "bypassPermissions"` |
| Plan mode | `permissionMode: "plan"` |
| Chat-only (tanpa akses tool/file) | `disallowedTools: ["*"]` |
| Model selection | `model: "sonnet"/"opus"/"haiku"` (alias) |
| Resume/continue | simpan `claude_session_id` dari result tiap turn, pass ke opsi `resume`/`continue` |

**Token/quota**: tidak ada endpoint resmi Claude Code untuk "sisa kuota". Pendekatan yang dipakai: akumulasi `total_cost_usd` (tersedia di tiap response turn) ke storage milik service sendiri, sebagai estimasi pemakaian — bukan quota real dari Anthropic.

**Login/provisioning kredensial BUKAN flow per-chat-session.** Menggunakan `claude setup-token` (command resmi untuk kasus headless/server dengan akun subscription): sekali dijalankan (perkiraan ~1x/tahun, token OAuth yang dihasilkan valid ~1 tahun), di-bridge lewat PTY (pseudo-terminal, karena command ini interaktif — menunggu approve browser lalu mencetak token ke stdout) lewat endpoint admin:
- `POST /auth/setup-token` — spawn `claude setup-token` via PTY, tangkap URL/instruksi login dari stdout, relay ke pemanggil (supaya ada manusia yang approve di browser), tangkap token akhir, simpan sebagai env `CLAUDE_CODE_OAUTH_TOKEN` yang dipakai proses service untuk semua session sesudahnya.
- `GET /auth/status` — hitung mundur ke estimasi expiry berdasarkan tanggal setup terakhir yang disimpan sendiri oleh service (Claude Code tidak expose expiry token secara terpisah).

Semua chat session normal reuse token ini tanpa perlu login ulang — login/`/login` interaktif biasa (browser lokal + expiry pendek) TIDAK dipakai karena tidak cocok untuk server headless yang diakses remote.

## Alasan

- **Sub-project + stack Node terpisah**: Claude Agent SDK first-class di TypeScript/Python; WebSocket + pengelolaan lifecycle proses/sesi lebih idiomatik di Node dibanding shelling-out dari Go dan parsing stdout CLI manual. Precedent sudah ada di `myagenda-service` — pilih stack per kebutuhan sub-project, bukan dipaksa Go.
- **Tanpa RBAC/auth di service ini (sementara)**: instruksi eksplisit user — fokus dulu ke fungsionalitas bridge-nya, otorisasi didelegasikan penuh ke `rnd-task-board` backend. Risiko disadari secara eksplisit oleh user (siapapun yang bisa reach service ini dengan `bypassPermissions` + `cwd` bebas setara akses eksekusi kode di server) — **ini keputusan sementara, bukan permanen**, auth peer-to-peer direncanakan sebagai iterasi berikutnya.
- **cwd eksplisit + validasi ringan, bukan allowlist ketat**: berdasarkan pengalaman langsung user bahwa Claude Code aman dijalankan selama di dalam folder project (bukan folder home/global) — cukup sanity check dasar, tidak perlu daftar path yang di-maintain manual.
- **Akun subscription via `claude setup-token`, bukan API key**: user secara eksplisit memilih model subscription (Pro/Max) dibanding API key billing. OAuth login interaktif biasa (`/login`) mengandalkan browser lokal di mesin yang sama — tidak cocok untuk server headless yang diakses dari client remote. `claude setup-token` adalah jalan resmi Anthropic untuk kasus ini: token tahan lama (~1 tahun), digenerate sekali dengan approve browser, dipakai berulang tanpa perlu login ulang tiap restart service atau tiap buka sesi chat baru.
- **Tidak ada tracking quota resmi**: Claude Code tidak mengekspos API "sisa kuota/token" — `total_cost_usd` per-turn adalah data paling dekat yang tersedia, jadi diakumulasi sendiri sebagai pendekatan terbaik yang ada.

## Dampak/File Terpengaruh

- `claude-chat-service/` — sub-project baru, **belum diimplementasi** (baru tahap keputusan arsitektur/scoping). Akan berisi `package.json`, entrypoint server (REST + WS), modul session manager, integrasi `@anthropic-ai/claude-agent-sdk`, modul PTY-bridge untuk `claude setup-token`, `docker-compose.yml`, `Dockerfile`, `README.md`.
- `CLAUDE.md` — ditambahkan ringkasan di bagian Project & Arsitektur, merujuk ke decision log ini.
- **Belum menyentuh** `backend/`/`frontend/` rnd-task-board — integrasi konsumsi (siapa yang memanggil `claude-chat-service`, bentuk auth peer-to-peer, apakah frontend connect langsung ke WS atau lewat proxy backend) sengaja ditunda, akan jadi decision log terpisah saat benar-benar dikerjakan.

## Belum diputuskan / sengaja ditunda (jangan asumsikan sudah ada)

- Auth/keamanan antar service (`rnd-task-board` ↔ `claude-chat-service`) — saat ini TIDAK ADA sama sekali.
- Bagaimana client menemukan daftar direktori yang valid untuk di-point (browse filesystem? daftar dikonfigurasi? tidak dibatasi sama sekali selain validasi ringan?).
- Skema pesan WebSocket yang presis (bentuk event, penamaan).
- Tempat penyimpanan akumulasi `total_cost_usd` (in-memory vs DB/file) dan retensinya.
- Bagaimana `rnd-task-board` (frontend/backend) benar-benar terhubung ke service ini — proxy lewat backend Go, atau client connect langsung.

## Update (2026-08-11, sesudah implementasi)

Decision log ini mencatat keputusan SCOPING awal (saat itu service belum ada kodenya). Perkembangan sesudahnya:

- **Service sudah diimplementasi & smoke-tested** (kode ada di `claude-chat-service/src/`). "Dampak/File Terpengaruh" di atas yang bilang "belum diimplementasi" adalah state SAAT keputusan dibuat, bukan sekarang.
- **Skema pesan WS, cara point direktori, koneksi ke `rnd-task-board`** yang tadinya di "belum diputuskan" SUDAH diputuskan: WS schema lihat `claude-chat-service/docs/api_contract.md`; dir picker via `GET /fs/browse`; koneksi = **backend proxy (Opsi B)** lihat `decision-log-change-request-integration-20260811.md`.
- **Masih ditunda beneran:** auth peer-to-peer antar service, dan persist akumulasi `total_cost_usd`.
