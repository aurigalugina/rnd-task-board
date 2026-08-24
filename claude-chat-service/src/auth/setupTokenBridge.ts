import pty from "node-pty";
import os from "node:os";

export type SetupTokenResult = { success: true; token: string } | { success: false; reason: string };

/**
 * Handle live ke satu proses `claude setup-token` yang lagi jalan -- dipakai
 * caller (routes/auth.ts) buat ngirim input balik (mis. authorization code
 * hasil approve browser, yang WAJIB di-paste manual ke terminal untuk
 * environment headless/SSH tanpa local browser callback) dan buat kill paksa
 * (mis. client SSE disconnect di tengah jalan).
 */
export type SetupTokenHandle = {
  sessionId: string;
  sendInput: (input: string) => void;
  kill: () => void;
};

/**
 * Bridge `claude setup-token` lewat pseudo-terminal (command ini interaktif —
 * ngeprint URL login, nunggu approve browser (dan di environment headless,
 * nunggu authorization code di-paste balik ke terminal), lalu ngeprint token
 * OAuth ~1 tahun ke stdout -- TIDAK disimpan sendiri oleh Claude Code, lihat
 * decision log). `onOutput` dipanggil per chunk stdout mentah (di-relay live
 * ke caller lewat SSE), `onDone` dipanggil sekali begitu proses selesai
 * (sukses dapat token, gagal, atau timeout).
 *
 * CATATAN: pola ekstraksi token di bawah (`extractToken`) BELUM divalidasi
 * terhadap output asli `claude setup-token` sesudah approve browser beneran —
 * proses ini sengaja tidak dijalankan otomatis selama development (punya efek
 * nyata: generate token OAuth baru terikat akun Claude asli). WAJIB
 * diverifikasi/disesuaikan pas dites langsung pertama kali.
 */
export function runSetupToken(
  sessionId: string,
  onOutput: (chunk: string) => void,
  onDone: (result: SetupTokenResult) => void,
  timeoutMs = 5 * 60 * 1000
): SetupTokenHandle {
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
    finish({ success: false, reason: "Timeout menunggu login OAuth selesai" });
    proc.kill();
  }, timeoutMs);

  function finish(result: SetupTokenResult) {
    if (settled) return;
    settled = true;
    clearTimeout(timer);
    onDone(result);
  }

  proc.onData((chunk) => {
    buffer += chunk;
    onOutput(chunk);
  });

  proc.onExit(({ exitCode }) => {
    if (settled) return;

    if (exitCode !== 0) {
      finish({ success: false, reason: `claude setup-token keluar dengan code ${exitCode}` });
      return;
    }

    const token = extractToken(buffer);
    if (!token) {
      finish({ success: false, reason: "Tidak berhasil menemukan token di output — cek log mentah" });
      return;
    }

    finish({ success: true, token });
  });

  return {
    sessionId,
    sendInput: (input: string) => {
      if (settled) return;
      // Terminal PTY butuh Enter eksplisit buat commit baris input -- kalau
      // caller belum nyertain newline, tambahin \r (bukan \n -- PTY raw mode
      // ngarepin carriage return buat Enter, konsisten sama xterm).
      proc.write(/[\r\n]$/.test(input) ? input : input + "\r");
    },
    kill: () => {
      finish({ success: false, reason: "Dibatalkan" });
      proc.kill();
    },
  };
}

export function extractToken(output: string): string | undefined {
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
