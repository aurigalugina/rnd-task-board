import { randomUUID } from "node:crypto";
import { Router } from "express";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { runSetupToken, type SetupTokenHandle } from "../auth/setupTokenBridge.js";
import { getTokenMeta, saveToken } from "../auth/tokenStore.js";

const execFileAsync = promisify(execFile);

export const authRouter = Router();

// Sesi `claude setup-token` yang lagi jalan, keyed by sessionId -- perlu
// disimpan di memory karena SSE cuma satu arah (server -> client); input
// (authorization code hasil approve browser) masuk lewat request TERPISAH
// (POST .../input) yang butuh cara nemu proses PTY yang sama lagi nunggu.
// Single-instance in-memory sudah cukup (service ini sendiri single-instance,
// sama seperti sessionManager.ts buat chat).
const activeSetupSessions = new Map<string, SetupTokenHandle>();

/**
 * Provisioning kredensial akun Claude subscription — BUKAN dipanggil per
 * chat session, ini flow admin sekali-sekali (~1x/tahun, token OAuth hasil
 * `setup-token` valid lama). Streaming via SSE karena `claude setup-token`
 * interaktif: ngeprint URL login, TERKADANG (environment headless/SSH tanpa
 * local browser callback) juga minta authorization code hasil approve
 * browser di-paste balik ke terminal -- makanya endpoint ini dipasangkan
 * sama POST .../input di bawah, bukan cuma one-shot spawn+wait kayak versi
 * awal (lihat rndops CLAUDE.md "Login OAuth Claude di-handle service
 * sendiri", 2026-08-21).
 */
authRouter.post("/auth/setup-token", (req, res) => {
  const sessionId = randomUUID();

  res.writeHead(200, {
    "Content-Type": "text/event-stream",
    "Cache-Control": "no-cache",
    Connection: "keep-alive",
  });

  const sendEvent = (event: string, data: unknown) => {
    res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
  };

  const handle = runSetupToken(
    sessionId,
    (chunk) => sendEvent("output", { chunk }),
    (result) => {
      activeSetupSessions.delete(sessionId);
      if (result.success) {
        const record = saveToken(result.token);
        sendEvent("done", { success: true, capturedAt: record.capturedAt });
      } else {
        sendEvent("done", { success: false, reason: result.reason });
      }
      res.end();
    }
  );
  activeSetupSessions.set(sessionId, handle);

  // Dikirim SEBELUM output pertama supaya client pasti punya sessionId
  // walau `claude setup-token` belum ngeprint apa-apa (mis. delay startup).
  sendEvent("session", { session_id: sessionId });

  req.on("close", () => {
    activeSetupSessions.delete(sessionId);
    handle.kill();
    res.end();
  });
});

/**
 * Kirim input (authorization code hasil approve browser) ke sesi
 * `claude setup-token` yang lagi nunggu, dari POST /auth/setup-token yang
 * masih streaming di request terpisah (SSE satu arah, gak bisa terima input
 * lewat koneksi yang sama).
 */
authRouter.post("/auth/setup-token/:sessionId/input", (req, res) => {
  const handle = activeSetupSessions.get(req.params.sessionId);
  if (!handle) {
    res.status(404).json({ error: "Sesi setup-token tidak ditemukan (sudah selesai/timeout/salah id)" });
    return;
  }
  const input = typeof req.body?.input === "string" ? req.body.input : "";
  if (!input) {
    res.status(400).json({ error: "Field 'input' wajib diisi" });
    return;
  }
  handle.sendInput(input);
  res.status(204).end();
});

/**
 * Status auth LIVE dari `claude auth status --json` (ground truth langsung
 * dari CLI — bukan cuma tebak-tebakan expiry dari tanggal setup terakhir),
 * digabung dengan metadata kapan token terakhir kita provision.
 */
authRouter.get("/auth/status", async (_req, res) => {
  const tokenMeta = getTokenMeta();

  try {
    const { stdout } = await execFileAsync("claude", ["auth", "status", "--json"]);
    const cliStatus = JSON.parse(stdout);
    res.json({ ...cliStatus, tokenProvisioning: tokenMeta });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    res.status(200).json({ loggedIn: false, error: message, tokenProvisioning: tokenMeta });
  }
});
