import { Router } from "express";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { runSetupToken } from "../auth/setupTokenBridge.js";
import { getTokenMeta, saveToken } from "../auth/tokenStore.js";

const execFileAsync = promisify(execFile);

export const authRouter = Router();

/**
 * Provisioning kredensial akun Claude subscription — BUKAN dipanggil per
 * chat session, ini flow admin sekali-sekali (~1x/tahun). Streaming via SSE
 * karena `claude setup-token` interaktif (nunggu approve browser di tengah
 * jalan) — client harus render output mentah biar admin bisa lihat
 * URL/instruksi login yang dicetak lalu approve manual di browsernya sendiri.
 */
authRouter.post("/auth/setup-token", (_req, res) => {
  res.writeHead(200, {
    "Content-Type": "text/event-stream",
    "Cache-Control": "no-cache",
    Connection: "keep-alive",
  });

  const sendEvent = (event: string, data: unknown) => {
    res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
  };

  runSetupToken((chunk) => sendEvent("output", { chunk }))
    .then((result) => {
      if (result.success) {
        const record = saveToken(result.token);
        sendEvent("done", { success: true, capturedAt: record.capturedAt });
      } else {
        sendEvent("done", { success: false, reason: result.reason });
      }
      res.end();
    })
    .catch((err) => {
      sendEvent("done", { success: false, reason: String(err) });
      res.end();
    });

  _req.on("close", () => res.end());
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
