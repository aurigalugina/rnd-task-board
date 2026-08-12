# claude-chat-service

Bridge antara Claude Code dan client aplikasi (chat UI) — fondasi untuk mekanisme
evolusi produk di `01-vision-product.md` §6, tapi scope-nya general-purpose.
Sub-project TERPISAH dari `backend`/`frontend`/`myagenda-service` rnd-task-board
(Node.js/TypeScript, `@anthropic-ai/claude-agent-sdk`).

**Baca dulu**: `docs/decision-log/decision-log-claude-chat-service-20260811.md`
di root repo — keputusan arsitektur & alasan lengkap (auth sengaja belum ada,
kenapa cwd wajib eksplisit, dst). Jangan implementasi menyimpang dari itu tanpa
diskusi ulang.

## TIDAK ADA AUTH/RBAC DI SERVICE INI (sengaja, sementara)

Service ini percaya penuh siapapun yang bisa reach dia — termasuk permintaan
`bypassPermissions` (setara akses eksekusi kode di `cwd` manapun yang dipilih).
Otorisasi "siapa boleh apa" adalah tanggung jawab consumer (`rnd-task-board`
backend). **Jangan expose service ini ke jaringan yang tidak dipercaya** sampai
auth peer-to-peer benar-benar dikerjakan.

## Menjalankan (dev)

```bash
npm install
npm run dev   # tsx watch src/index.ts, port 8090 (env PORT)
```

Butuh Claude Code CLI global (`npm install -g @anthropic-ai/claude-code`) untuk
`/auth/*` — `@anthropic-ai/claude-agent-sdk` sendiri sudah bundle binary-nya
sendiri buat sesi chat (`POST /sessions`), tidak butuh CLI global untuk itu.

```bash
npm run build && npm start   # compile TS ke dist/, jalanin hasil build
npm test                     # vitest — cuma logic murni (cwdValidation) yang ditest
```

## Provisioning akun Claude (sekali di awal, ~1x/tahun)

Chat session TIDAK butuh login per-session — service reuse satu token OAuth
tahan lama (`CLAUDE_CODE_OAUTH_TOKEN`, hasil `claude setup-token`, valid ~1
tahun). Provisioning:

```bash
curl -N -X POST http://localhost:8090/auth/setup-token
```

Respons SSE — event `output` berisi chunk stdout mentah (termasuk
URL/instruksi login OAuth, buka manual di browser & approve), event `done`
menandakan selesai (`success: true` + `capturedAt`, token otomatis tersimpan
ke `TOKEN_STORE_PATH` dan langsung dipakai sesi berikutnya).

> **Belum divalidasi terhadap output CLI asli** (lihat catatan di
> `src/auth/setupTokenBridge.ts`) — proses ini sengaja tidak dijalankan otomatis
> selama development karena punya efek nyata (generate token OAuth baru terikat
> akun Claude asli). Verifikasi/sesuaikan `extractToken()` pas dites langsung
> pertama kali.

Cek status: `GET /auth/status` (live dari `claude auth status --json`, digabung
metadata kapan token terakhir kita provision).

## API

### REST

| Endpoint | Fungsi |
|---|---|
| `GET /fs/browse?path=` | List subdirektori (buat UI "pointing dir"). Default root: `BROWSE_DEFAULT_ROOT`/`$HOME`. |
| `POST /sessions` | Bikin chat session baru. Body: `{cwd, model?, permissionMode?, resume?}`. |
| `GET /sessions` | List semua session aktif di memori proses ini. |
| `GET /sessions/:id` | Detail satu session (termasuk akumulasi `totalCostUsd`). |
| `DELETE /sessions/:id` | Tutup session (abort query, hapus dari memori). |
| `POST /auth/setup-token` | SSE — provisioning token (lihat di atas). |
| `GET /auth/status` | Status auth live. |

`permissionMode`: `default` \| `acceptEdits` \| `bypassPermissions` \| `plan` \|
`dontAsk` \| `auto`. `resume` = `claude_session_id` dari session sebelumnya
(BUKAN `id` internal kita) — resume selalu bikin session (internal) baru yang
membungkus riwayat percakapan lama.

### WebSocket — `/ws/sessions/:id`

Client → server:
```jsonc
{"type": "prompt", "text": "..."}
{"type": "set_mode", "mode": "bypassPermissions"}
{"type": "set_model", "model": "sonnet"}
{"type": "interrupt"}
```

Server → client:
```jsonc
{"type": "sdk_message", "message": { /* SDKMessage mentah dari Agent SDK */ }}
{"type": "session_updated", "session": { /* SessionSummary */ }}
{"type": "error", "message": "..."}
```

`message` di event `sdk_message` diteruskan APA ADANYA dari
`@anthropic-ai/claude-agent-sdk` (`SDKMessage` union — assistant text delta,
tool_use, result dengan `total_cost_usd`, dst) — client yang parse sesuai
kebutuhan render-nya, service ini tidak menyaring/mengubah bentuknya.

## Catatan Docker & filesystem

Kalau dijalankan via Docker, service HANYA bisa "pointing dir" ke path yang
memang di-mount ke dalam container (lihat komentar di `docker-compose.yml`).
Untuk kebutuhan bridging ke sembarang folder project di host, menjalankan
service ini LANGSUNG di host (`npm start`, bukan Docker) kemungkinan lebih
praktis — pertimbangkan sesuai environment deployment yang sebenarnya.

## Sengaja belum diputuskan/dikerjakan

Lihat bagian akhir `decision-log-claude-chat-service-20260811.md` — auth
peer-to-peer ke `rnd-task-board`, skema pesan WS yang lebih presis/stabil,
tempat penyimpanan cost selain in-memory, cara `rnd-task-board` benar-benar
konek ke service ini.
