# claude-chat-service — Integration Guide

Panduan untuk consumer yang mau konek ke service ini — baik frontend langsung
(browser WebSocket) maupun via backend proxy. Baca `api_contract.md` dulu untuk
referensi lengkap tiap endpoint.

---

## Pilihan arsitektur koneksi

Ada dua pola, keduanya valid — pilih sesuai kebutuhan:

### Opsi A — Frontend langsung ke service

Browser connect langsung ke `claude-chat-service` (REST + WebSocket). Backend
consumer (`rnd-task-board`) tidak terlibat di jalur chat.

```
Browser  ──REST/WS──►  claude-chat-service :8090
Browser  ──REST/WS──►  rnd-task-board :8080   (untuk fitur lain: task, board, dll)
```

**Cocok kalau:** deployment simpel, satu server, tidak butuh middleware di depan chat.
**Perlu:** CORS di-set di `claude-chat-service` (belum ada di implementasi sekarang — tambah header `Access-Control-Allow-Origin` kalau frontend beda origin).

### Opsi B — Backend proxy

`rnd-task-board` backend forward `/api/v1/chat/*` ke `claude-chat-service`.
Frontend hanya tahu satu base URL.

```
Browser  ──REST/WS──►  rnd-task-board :8080  ──forward──►  claude-chat-service :8090
```

**Cocok kalau:** mau tambah middleware di depan (rate-limit per user, log, auth
peer-to-peer nanti), atau frontend sudah pakai proxy SvelteKit / nginx ke satu host.

> Pola yang dipilih sebaiknya dicatat di decision log sebelum diimplementasi.
> Belum ada keputusan final untuk `rnd-task-board` — lihat catatan di
> `decision-log-claude-chat-service-20260811.md`.

---

## Alur integrasi end-to-end

### 1 — Browse direktori (UI "pointing dir")

Sebelum session dibuat, user harus pilih direktori project secara eksplisit.

```
GET /fs/browse?path=/home/lugidev
→ { path, parent, directories: [{name, path}, ...] }
```

Render sebagai file picker sederhana: klik folder → fetch `?path=<folder.path>` lagi
sampai user confirm direktori target. Simpan `path` yang dipilih sebagai `cwd` untuk
langkah berikutnya.

```typescript
async function browse(dirPath: string) {
  const res = await fetch(`http://localhost:8090/fs/browse?path=${encodeURIComponent(dirPath)}`);
  return res.json() as Promise<{ path: string; parent: string; directories: {name: string; path: string}[] }>;
}
```

---

### 2 — Buat session

```typescript
const res = await fetch("http://localhost:8090/sessions", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    cwd: "/home/lugidev/rnd-task-board",
    permissionMode: "plan",          // atau "bypassPermissions" kalau mau eksekusi
    // model: "claude-opus-4-8",     // opsional — SDK pilih default kalau dikosongi
    // resume: savedClaudeSessionId, // opsional — lihat bagian Resume
  }),
});
const session = await res.json();
// session.id   → dipakai untuk WS path & REST
// session.claudeSessionId → awalnya null, diisi async
```

Simpan `session.id` (internal) dan pantau `session.claudeSessionId` yang muncul setelah
turn pertama — butuh ini untuk resume.

---

### 3 — Connect WebSocket

```typescript
const ws = new WebSocket(`ws://localhost:8090/ws/sessions/${session.id}`);

ws.addEventListener("open", () => {
  console.log("WS connected, siap kirim prompt");
});

ws.addEventListener("message", (event) => {
  const msg = JSON.parse(event.data);
  handleServerMessage(msg);
});

ws.addEventListener("close", () => {
  console.log("WS closed");
});
```

---

### 4 — Kirim prompt

```typescript
function sendPrompt(text: string) {
  ws.send(JSON.stringify({ type: "prompt", text }));
}

sendPrompt("Jelaskan struktur folder backend di repo ini.");
```

---

### 5 — Handle server messages

```typescript
function handleServerMessage(msg: ServerToClientMessage) {
  switch (msg.type) {
    case "sdk_message":
      renderSdkMessage(msg.message);
      break;
    case "session_updated":
      updateSessionState(msg.session);
      break;
    case "error":
      showErrorBanner(msg.message);
      break;
  }
}

function renderSdkMessage(sdkMsg: unknown) {
  const m = sdkMsg as Record<string, unknown>;

  switch (m.type) {
    case "assistant": {
      // Respons teks / tool call dari Claude
      const content = (m.message as any).content as Array<{type: string; text?: string}>;
      const textBlocks = content.filter(b => b.type === "text");
      appendToChat("assistant", textBlocks.map(b => b.text).join(""));
      break;
    }
    case "result": {
      // Akhir turn — simpan claudeSessionId untuk resume
      const result = m as { session_id: string; total_cost_usd: number; subtype: string };
      saveClaudeSessionId(result.session_id);
      updateCostDisplay(result.total_cost_usd);
      setTurnComplete(result.subtype === "success");
      break;
    }
    case "system":
      // init event — bisa diabaikan atau log untuk debug
      break;
    case "rate_limit_event":
      showStatus("Claude sedang rate-limited, menunggu...");
      break;
    // stream delta events — parse sesuai kebutuhan kalau mau streaming real-time
    default:
      console.debug("sdk event:", m.type, m);
  }
}
```

---

### 6 — Multi-turn conversation

Setelah `result` event diterima (turn selesai), kirim prompt berikutnya ke WS yang
**sama** — session tetap hidup, riwayat percakapan di-maintain oleh SDK secara internal.

```typescript
// Turn 1
sendPrompt("Apa itu Big Task di proyek ini?");
// … tunggu result event …

// Turn 2 — session WS yang sama, riwayat tersimpan
sendPrompt("Bagaimana cara sign-off Big Task?");
```

---

### 7 — Ubah mode / model live

```typescript
// Switch ke bypass setelah user confirm
ws.send(JSON.stringify({ type: "set_mode", mode: "bypassPermissions" }));

// Ganti model
ws.send(JSON.stringify({ type: "set_model", model: "claude-opus-4-8" }));

// Kembali ke default
ws.send(JSON.stringify({ type: "set_model" }));
```

Server akan balas `session_updated` dengan state session terbaru.

---

### 8 — Interrupt

Kalau user klik "Stop" saat Claude sedang proses:

```typescript
ws.send(JSON.stringify({ type: "interrupt" }));
```

SDK akan menghentikan turn yang berjalan. Event `result` dengan `subtype: "interrupted"`
akan datang via WS.

---

### 9 — Tutup session

Saat user keluar dari chat atau menutup panel:

```typescript
ws.close();

// Kalau mau benar-benar hapus dari memori service:
await fetch(`http://localhost:8090/sessions/${session.id}`, { method: "DELETE" });
```

Tidak wajib `DELETE` — session juga akan otomatis closed kalau service restart
(karena in-memory). Tapi kalau banyak session dibuka dan tidak pernah di-close,
memori service akan terus tumbuh.

---

## Resume session Claude sebelumnya

`resume` membungkus riwayat percakapan Claude lama ke dalam session (internal) baru.
**Bukan** "lanjutkan session yang sama" — koneksi WS ke session lama tetap terputus
begitu di-close, yang di-resume adalah riwayat conversation-nya.

```typescript
// Simpan ini setelah dapat dari result event:
const claudeSessionId = "e6d34daf-bdcf-42bf-94c2-3da287d3dd21"; // dari result.session_id

// Nanti, saat user mau lanjut:
const res = await fetch("http://localhost:8090/sessions", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    cwd: "/home/lugidev/rnd-task-board",
    resume: claudeSessionId,
    // permissionMode & model bisa diset ulang — resume tidak lock ke nilai sebelumnya
  }),
});
const newSession = await res.json();
// Connect WS ke newSession.id, lanjutkan seperti biasa
```

> Cost `totalCostUsd` di session baru dimulai dari 0 (bukan kumulatif dari session lama).
> Turn pertama setelah resume akan lebih murah karena prompt cache dari session lama
> kemungkinan masih valid (TTL 5 menit di Anthropic).

---

## Provisioning token Claude (admin, ~1x/tahun)

Token OAuth disimpan oleh service di `TOKEN_STORE_PATH` dan di-load otomatis tiap
restart — chat session normal tidak butuh langkah ini selama token masih valid.

```bash
# Jalankan dari terminal, streaming SSE mentah:
curl -N -X POST http://localhost:8090/auth/setup-token
```

Atau dari JavaScript:

```typescript
const response = await fetch("http://localhost:8090/auth/setup-token", { method: "POST" });
const reader = response.body!.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  const text = decoder.decode(value);
  // Parse SSE lines: "event: output\ndata: {...}\n\n"
  for (const line of text.split("\n")) {
    if (line.startsWith("data: ")) {
      const data = JSON.parse(line.slice(6));
      if ("chunk" in data) {
        // Tampilkan ke admin — berisi URL login / instruksi
        process.stdout.write(data.chunk);
      } else if ("success" in data) {
        console.log("Token provisioned:", data);
      }
    }
  }
}
```

Cek status kapan saja:
```bash
curl http://localhost:8090/auth/status
```

---

## Catatan penting untuk consumer

### Tidak ada auth di service ini (sengaja)

Service percaya penuh siapapun yang bisa reach-nya. **Consumer wajib memastikan** hanya
user yang berwenang yang boleh buat session — terutama session dengan `bypassPermissions`
(setara akses langsung ke filesystem `cwd`). Kalau diekspos via backend proxy,
tambahkan auth middleware di sana.

### `claudeSessionId` baru tersedia setelah turn pertama

Simpan `claudeSessionId` dari event `result` di WS (bukan dari `POST /sessions` response
yang masih `null`). Kalau perlu persist untuk resume lintas sesi browser, simpan di
server-side storage — jangan andalkan `GET /sessions/:id` karena session hilang tiap
service restart.

### Biaya pertama kali mahal — wajar

Turn pertama di session baru akan membangun prompt cache (`cache_creation_input_tokens`
bisa 50k+). Turn berikutnya di session yang sama jauh lebih murah (cache read). Ini
perilaku normal Agent SDK / Anthropic, bukan bug.

### Docker: filesystem harus di-mount

Kalau service jalan di Docker dan `cwd` yang dipilih user ada di host filesystem,
path itu harus di-mount ke container. Opsi lain: jalankan service langsung di host
(`npm start`) — lebih simpel untuk development lokal.

### Cost storage in-memory

`totalCostUsd` di `SessionSummary` hanya akurat selama service tidak restart. Kalau
butuh tracking cost yang persist (per user, per proyek), consumer perlu ekstrak nilai
dari event `result` via WS dan simpan sendiri di storage-nya.
