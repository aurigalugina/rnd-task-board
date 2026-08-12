import pty from "node-pty";
import os from "node:os";

export type SetupTokenResult = { success: true; token: string } | { success: false; reason: string };

/**
 * Bridge `claude setup-token` lewat pseudo-terminal (command ini interaktif —
 * nunggu approve browser lalu ngeprint token OAuth ~1 tahun ke stdout, TIDAK
 * disimpan sendiri oleh Claude Code, lihat decision log). `onOutput` dipanggil
 * per chunk stdout mentah (buat di-relay live ke caller — berisi URL/instruksi
 * login yang perlu dibuka manual oleh admin).
 *
 * CATATAN: pola ekstraksi token di bawah (`extractToken`) BELUM divalidasi
 * terhadap output asli `claude setup-token` sesudah approve browser beneran —
 * proses ini sengaja tidak dijalankan otomatis selama development (punya efek
 * nyata: generate token OAuth baru terikat akun Claude asli). WAJIB
 * diverifikasi/disesuaikan pas dites langsung pertama kali.
 */
export function runSetupToken(onOutput: (chunk: string) => void, timeoutMs = 5 * 60 * 1000): Promise<SetupTokenResult> {
  return new Promise((resolve) => {
    let settled = false;
    let buffer = "";

    const proc = pty.spawn("claude", ["setup-token"], {
      name: "xterm-256color",
      cols: 120,
      rows: 30,
      cwd: os.tmpdir(),
      env: process.env as Record<string, string>,
    });

    const timer = setTimeout(() => {
      if (settled) return;
      settled = true;
      proc.kill();
      resolve({ success: false, reason: "Timeout menunggu login OAuth selesai" });
    }, timeoutMs);

    proc.onData((chunk) => {
      buffer += chunk;
      onOutput(chunk);
    });

    proc.onExit(({ exitCode }) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);

      if (exitCode !== 0) {
        resolve({ success: false, reason: `claude setup-token keluar dengan code ${exitCode}` });
        return;
      }

      const token = extractToken(buffer);
      if (!token) {
        resolve({ success: false, reason: "Tidak berhasil menemukan token di output — cek log mentah" });
        return;
      }

      resolve({ success: true, token });
    });
  });
}

function extractToken(output: string): string | undefined {
  const prefixed = output.match(/\bsk-ant-oat[0-9A-Za-z_-]{10,}\b/);
  if (prefixed) return prefixed[0];

  const lines = output
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  const last = lines.at(-1);
  if (last && /^[A-Za-z0-9_.\-]{20,}$/.test(last)) return last;

  return undefined;
}
