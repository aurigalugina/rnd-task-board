# claude-chat-service — API Contract

Versi dokumen ini mencerminkan state implementasi di `src/` (bukan draft/rencana).
Update dokumen ini tiap ada perubahan interface.

---

## Base URL

```
http://<host>:8090
```

Default port `8090`, override via env `PORT`.

---

## REST Endpoints

### `GET /healthz`

Liveness check.

**Response 200**
```json
{ "ok": true }
```

---

### `GET /fs/browse`

List subdirektori non-hidden di `path`. Dipakai UI "pointing dir" sebelum buat session — **sengaja tidak membatasi home/root di sini**, karena user perlu navigasi masuk ke subfolder project dari home root. Validasi ketat hanya berlaku saat `POST /sessions`.

**Query params**

| Param | Tipe | Default | Keterangan |
|---|---|---|---|
| `path` | `string` | `BROWSE_DEFAULT_ROOT` / `$HOME` | Absolute path direktori yang mau di-list |

**Response 200**
```json
{
  "path": "/home/lugidev/rnd-task-board",
  "parent": "/home/lugidev",
  "directories": [
    { "name": "backend",          "path": "/home/lugidev/rnd-task-board/backend" },
    { "name": "claude-chat-service", "path": "/home/lugidev/rnd-task-board/claude-chat-service" },
    { "name": "frontend",         "path": "/home/lugidev/rnd-task-board/frontend" }
  ]
}
```

**Error 400** — path bukan absolute, atau direktori tidak bisa dibaca:
```json
{ "error": "path harus berupa absolute path" }
{ "error": "Gagal baca direktori: ENOENT: no such file or directory, scandir '...'" }
```

---

### `POST /sessions`

Buat chat session baru. Session langsung aktif setelah dibuat — Agent SDK `query()` sudah berjalan, tinggal connect WebSocket dan kirim prompt pertama.

**Request body**

```json
{
  "cwd": "/home/lugidev/rnd-task-board",
  "model": "claude-sonnet-4-5",
  "permissionMode": "plan",
  "resume": "e6d34daf-bdcf-42bf-94c2-3da287d3dd21"
}
```

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `cwd` | `string` | ✅ | Absolute path direktori project. Tidak boleh: `/`, `$HOME`, path dari `CWD_FORBIDDEN_ROOTS`. Harus direktori yang benar-benar ada. |
| `model` | `string` | — | ID model Claude (`claude-opus-4-8`, `claude-sonnet-4-6`, dll). Default: SDK default. |
| `permissionMode` | `PermissionMode` | — | Lihat tabel Permission Modes. Default: `"default"`. |
| `resume` | `string` | — | `claude_session_id` dari session sebelumnya (bukan `id` internal kita). Resume selalu bikin session internal baru yang membungkus riwayat Claude lama. |

**Permission Modes**

| Mode | Perilaku |
|---|---|
| `default` | Permission prompt per tool call (tidak terpakai tanpa UI interaktif) |
| `acceptEdits` | Otomatis approve edit file, prompt untuk hal lain |
| `bypassPermissions` | Lewati semua permission check — Claude bisa edit/eksekusi apa saja di `cwd` |
| `plan` | Read-only: analisis & rencana saja, tidak ada write/eksekusi |
| `dontAsk` | Tidak prompt sama sekali, lanjutkan dengan tool calls |
| `auto` | SDK pilih otomatis |

> `bypassPermissions` otomatis set `allowDangerouslySkipPermissions: true` di SDK. Pastikan consumer sudah otorisasi user sebelum bolehkan mode ini.

**Response 201** — [`SessionSummary`](#sessionsummary)
```json
{
  "id": "a1b2c3d4-...",
  "cwd": "/home/lugidev/rnd-task-board",
  "model": "claude-sonnet-4-5",
  "permissionMode": "plan",
  "claudeSessionId": null,
  "totalCostUsd": 0,
  "status": "active",
  "createdAt": "2026-08-11T10:00:00.000Z"
}
```

`claudeSessionId` diisi setelah pesan pertama dari SDK masuk (async — polling via `GET /sessions/:id` atau pantau lewat WS event `session_updated`).

**Error 400** — validasi gagal:
```json
{ "error": "\"cwd\" terlalu luas — pilih subfolder project yang lebih spesifik" }
{ "error": "\"/not/exist\" bukan direktori yang valid" }
{ "error": { "fieldErrors": { "permissionMode": ["..."] } } }
```

---

### `GET /sessions`

List semua session aktif di memori proses ini. Session di-reset tiap service restart (storage in-memory, belum persist).

**Response 200** — array [`SessionSummary`](#sessionsummary)

---

### `GET /sessions/:id`

Detail satu session termasuk `totalCostUsd` terkini dan `claudeSessionId` (untuk keperluan resume nanti).

**Response 200** — [`SessionSummary`](#sessionsummary)

**Error 404**
```json
{ "error": "session tidak ditemukan" }
```

---

### `DELETE /sessions/:id`

Tutup session: abort `Query` SDK, close `AsyncQueue` input, hapus dari memori. WebSocket client yang sedang terhubung akan kehilangan koneksi.

**Response 204** — kosong

**Error 404**
```json
{ "error": "session tidak ditemukan" }
```

---

### `POST /auth/setup-token`

**SSE endpoint** — provisioning token OAuth Claude subscription (~1x/tahun, bukan per chat session). Spawn `claude setup-token` via PTY, stream stdout ke client supaya admin bisa lihat URL login dan approve manual di browser. **2026-08-21:** sesi PTY-nya keyed by `session_id` (dikirim sebagai event pertama) dan disimpan in-memory (`activeSetupSessions` di `routes/auth.ts`) selama proses masih jalan — dipasangkan sama `POST /auth/setup-token/{session_id}/input` di bawah buat kirim authorization code balik (environment headless/SSH biasanya minta approve browser DIIKUTI paste code manual ke terminal, bukan auto-detect via local callback).

Response `Content-Type: text/event-stream`.

**Events**

```
event: session
data: {"session_id": "b3f1..."}

event: output
data: {"chunk": "Opening browser for authentication...\nhttps://claude.ai/..."}

event: done
data: {"success": true, "capturedAt": "2026-08-11T10:00:00.000Z"}
```

```
event: done
data: {"success": false, "reason": "timeout atau parsing token gagal"}
```

Client disconnect (SSE ditutup dari sisi caller, mis. tombol "Batalkan") otomatis `kill()` proses PTY dan hapus sesi dari `activeSetupSessions` (lihat `req.on("close", ...)`).

> **Belum divalidasi terhadap output CLI asli** — `extractToken()` di `src/auth/setupTokenBridge.ts` perlu diverifikasi/disesuaikan pas dites langsung pertama kali (baca komentar di sana).

---

### `POST /auth/setup-token/{session_id}/input`

Kirim satu baris input (biasanya authorization code hasil approve browser) ke proses `claude setup-token` yang lagi jalan dan ditunjuk oleh `session_id` (didapat dari event `session` di atas). Enter/newline ditambahkan otomatis kalau caller belum menyertakannya.

**Request body**: `{ "input": "string" }`

**Response**: `204` kalau berhasil dikirim; `404` kalau `session_id` tidak ditemukan (sesi sudah selesai/timeout/salah id); `400` kalau `input` kosong.

---

### `GET /auth/status`

Status auth live dari `claude auth status --json`, digabung metadata token yang kita simpan.

**Response 200**
```json
{
  "loggedIn": true,
  "authMethod": "claude.ai",
  "subscriptionType": "pro",
  "email": "user@example.com",
  "tokenProvisioning": {
    "present": true,
    "capturedAt": "2026-08-11T10:00:00.000Z"
  }
}
```

```json
{
  "loggedIn": false,
  "error": "command not found: claude",
  "tokenProvisioning": { "present": false, "capturedAt": null }
}
```

---

## WebSocket — `/ws/sessions/:id`

Upgrade HTTP → WebSocket. Session harus `status === "active"`, kalau tidak: **404** lalu socket di-destroy.

Satu session bisa punya **beberapa koneksi WS sekaligus** (fan-out) — semua dapat event yang sama dari SDK.

### Client → Server

Semua pesan JSON.

**`prompt`** — kirim teks user ke Claude (opsional dengan gambar/screenshot)
```json
{ "type": "prompt", "text": "Jelaskan struktur folder backend di repo ini." }
```

Dengan lampiran gambar (Claude multimodal — mis. screenshot). `images[].data` = base64 **mentah** (tanpa prefix `data:...;base64,`). Boleh kirim gambar tanpa `text` (string kosong). `media_type` yang didukung: `image/png`, `image/jpeg`, `image/gif`, `image/webp`.
```json
{
  "type": "prompt",
  "text": "Warna tombol ini kurang kontras, tolong usulkan perbaikan.",
  "images": [
    { "media_type": "image/png", "data": "iVBORw0KGgoAAAANSUhEUg..." }
  ]
}
```
Diteruskan ke SDK sebagai content block array (`{type:"text"}` + `{type:"image", source:{type:"base64",...}}`). Kalau `images` kosong/absen, `content` tetap dikirim sebagai string biasa (backward-compatible).

**`set_mode`** — ubah permission mode live (berlaku di turn berikutnya)
```json
{ "type": "set_mode", "mode": "bypassPermissions" }
```
Sukses → server balas `session_updated`.

**`set_model`** — ubah model live
```json
{ "type": "set_model", "model": "claude-opus-4-8" }
```
Kosongkan `model` untuk kembali ke default SDK:
```json
{ "type": "set_model" }
```
Sukses → server balas `session_updated`.

**`interrupt`** — interrupt turn yang sedang berjalan
```json
{ "type": "interrupt" }
```

---

### Server → Client

**`sdk_message`** — event mentah dari `@anthropic-ai/claude-agent-sdk`, diteruskan apa adanya. Client yang bertanggung jawab parse sesuai kebutuhan render.

```json
{ "type": "sdk_message", "message": { /* SDKMessage */ } }
```

Lihat [SDK Message Types](#sdk-message-types) di bawah.

**`session_updated`** — dikirim setelah `set_mode` / `set_model` sukses
```json
{ "type": "session_updated", "session": { /* SessionSummary */ } }
```

**`error`** — error di level service (bukan error dari Claude)
```json
{ "type": "error", "message": "payload bukan JSON valid" }
```

---

## SDK Message Types

Pesan dalam `sdk_message.message` adalah union `SDKMessage` dari `@anthropic-ai/claude-agent-sdk`. Field-field utama yang relevan untuk rendering:

### `system`
Pesan inisialisasi sesi. Subtype `init` berisi info tools yang tersedia.
```json
{
  "type": "system",
  "subtype": "init",
  "session_id": "e6d34daf-...",
  "tools": [ /* list tool definitions */ ]
}
```

### `assistant`
Respons teks / tool call dari Claude.
```json
{
  "type": "assistant",
  "session_id": "e6d34daf-...",
  "message": {
    "role": "assistant",
    "content": [
      { "type": "text", "text": "Struktur folder backend adalah..." }
    ]
  }
}
```
Content bisa juga berisi `tool_use` block kalau Claude memanggil tool.

### `user`
Echo pesan user (termasuk `tool_result` dari tool calls).
```json
{
  "type": "user",
  "session_id": "e6d34daf-...",
  "message": {
    "role": "user",
    "content": [ { "type": "tool_result", ... } ]
  }
}
```

### `result`
Akhir dari satu turn. **Field penting**: `session_id` (simpan untuk resume!), `total_cost_usd`, `subtype`.
```json
{
  "type": "result",
  "subtype": "success",
  "session_id": "e6d34daf-bdcf-42bf-94c2-3da287d3dd21",
  "total_cost_usd": 0.30794,
  "stop_reason": "end_turn",
  "modelUsage": {
    "claude-sonnet-4-6": {
      "input_tokens": 1000,
      "output_tokens": 200,
      "cache_read_input_tokens": 48000,
      "cache_creation_input_tokens": 51212
    }
  }
}
```

`subtype` bisa: `success` | `error_max_turns` | `error_during_execution` | `interrupted`.

### `rate_limit_event`
SDK sedang menunggu rate limit. Biasanya tidak perlu dirender ke user, cukup ditampilkan sebagai "sedang menunggu...".

### Stream events (partial)
Karena `includePartialMessages: true`, ada delta events untuk streaming teks real-time. Bentuknya wrapper `stream_event` atau langsung delta — parse `message.type` untuk bedain dari pesan lengkap.

---

## Data Types

### `SessionSummary`

```typescript
{
  id: string;              // UUID internal kita — dipakai untuk REST & WS path
  cwd: string;             // Absolute path yang diset saat create
  model?: string;          // Model yang dipakai (undefined = SDK default)
  permissionMode: PermissionMode;
  claudeSessionId?: string; // claude_session_id dari SDK — simpan untuk resume
  totalCostUsd: number;    // Akumulasi biaya selama session (in-memory, reset tiap restart)
  status: "active" | "closed";
  createdAt: string;       // ISO 8601
}
```

---

## Error Response Umum

Semua error REST punya shape:
```json
{ "error": "pesan error" }
```

atau untuk validation error zod:
```json
{ "error": { "fieldErrors": { "field": ["..."] }, "formErrors": [] } }
```

| HTTP | Kondisi |
|---|---|
| 400 | Input tidak valid (cwd, schema, path) |
| 404 | Session / resource tidak ditemukan |
| 204 | Delete berhasil (no body) |

WebSocket: koneksi ke session non-existent / closed → HTTP 404 di upgrade handshake, socket di-destroy.
