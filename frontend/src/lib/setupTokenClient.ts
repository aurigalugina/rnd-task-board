// Klien buat provisioning ulang OAuth Claude (claude-chat-service, akun
// subscription BERSAMA) lewat backend proxy. Endpoint `/chat/auth/setup-token`
// itu SSE (server->client doang), jadi TIDAK bisa pakai `EventSource` (cuma
// dukung GET, gak bisa bawa header Authorization) -- dipakai `fetch` +
// `ReadableStream` manual. Input balik (authorization code hasil approve
// browser) dikirim lewat request TERPISAH (`sendSetupTokenInput`), lihat
// claude-chat-service/src/routes/auth.ts.
import { api, getAccessToken } from './api/client';

export type SetupTokenEvent =
  | { type: 'session'; sessionId: string }
  | { type: 'output'; chunk: string }
  | { type: 'done'; success: true; capturedAt: string }
  | { type: 'done'; success: false; reason: string };

export type AuthStatus = {
  loggedIn: boolean;
  error?: string;
  tokenProvisioning: { present: boolean; capturedAt?: string };
};

export const setupTokenApi = {
  status: () => api.get<AuthStatus>('/chat/auth/status'),
  sendInput: (sessionId: string, input: string) =>
    api.post<void>(`/chat/auth/setup-token/${sessionId}/input`, { input })
};

// parseSseMessage mengurai satu blok pesan SSE mentah (tanpa pemisah "\n\n"
// di akhir) jadi event terstruktur. Diekstrak dari loop streaming supaya bisa
// ditest tanpa perlu fetch/ReadableStream beneran.
export function parseSseMessage(raw: string): SetupTokenEvent | null {
  let event = 'message';
  let data = '';
  for (const line of raw.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim();
    else if (line.startsWith('data:')) data += line.slice(5).trim();
  }
  if (!data) return null;

  let parsed: any;
  try {
    parsed = JSON.parse(data);
  } catch {
    return null;
  }

  if (event === 'session' && typeof parsed.session_id === 'string') {
    return { type: 'session', sessionId: parsed.session_id };
  }
  if (event === 'output' && typeof parsed.chunk === 'string') {
    return { type: 'output', chunk: parsed.chunk };
  }
  if (event === 'done') {
    if (parsed.success) return { type: 'done', success: true, capturedAt: parsed.capturedAt ?? '' };
    return { type: 'done', success: false, reason: parsed.reason ?? 'Gagal (tidak ada alasan)' };
  }
  return null;
}

// extractErrorMessage mengurai body error respons non-2xx dari chatproxy.
// chatproxy balikin bentuk flat {"error":"pesan"} (beda dari
// {"error":{"message":...}} yang dipakai backend Go biasa) -- coba urai biar
// pesannya bersih bukan JSON mentah, fallback ke teks apa adanya kalau bukan
// JSON. Diekstrak jadi fungsi murni supaya bisa ditest tanpa fetch beneran.
export function extractErrorMessage(status: number, rawText: string): string {
  if (!rawText) return `Gagal memulai setup-token: ${status}`;
  try {
    const parsed = JSON.parse(rawText);
    if (typeof parsed?.error === 'string') return parsed.error;
  } catch {
    // bukan JSON, pakai teks mentah apa adanya
  }
  return rawText;
}

// stripAnsi membuang kode ANSI (escape sequence warna, cursor positioning,
// dst) yang dikirim `claude setup-token` -- CLI itu didesain buat terminal
// interaktif (PTY), bukan buat direndernya polos di web. Tanpa ini,
// authOutput penuh escape code mentah (\x1b[38;5;174m, \x1b[46G, dst) yang
// bikin log-nya sama sekali gak kebaca di UI. Regex ini cover CSI sequence
// (\x1b[...huruf) dan sequence 2-karakter lain (\x1b7, \x1b8, dst).
export function stripAnsi(input: string): string {
  // eslint-disable-next-line no-control-regex
  return input
    .replace(/\x1b\[[0-9;?]*[a-zA-Z]/g, '')
    .replace(/\x1b[()][A-Za-z0-9]/g, '')
    .replace(/\x1b./g, '')
    .replace(/\r/g, '');
}

// extractLoginUrl mencari URL OAuth login di output CLI yang sudah di-strip
// ANSI-nya -- CLI membungkus URL panjang jadi beberapa baris terpisah spasi
// kosong (line-wrap layar terminal), jadi baris "https://..." harus
// disambung dulu dengan baris lanjutannya sebelum bisa dipakai sebagai href
// yang valid (query string terputus di tengah kalau tidak).
export function extractLoginUrl(strippedOutput: string): string | null {
  const idx = strippedOutput.indexOf('https://');
  if (idx === -1) return null;
  // Ambil sampai whitespace ganda (baris kosong) atau baris "Paste code..."
  // -- indikator akhir blok URL di output asli `claude setup-token`.
  const rest = strippedOutput.slice(idx);
  const stopMatch = rest.match(/\n\s*\n|Paste\s*code/);
  const urlBlock = stopMatch ? rest.slice(0, stopMatch.index) : rest;
  const url = urlBlock.replace(/\s+/g, '').trim();
  return url.startsWith('https://') ? url : null;
}

// streamSetupToken membuka koneksi SSE dan memanggil onEvent per pesan yang
// berhasil diurai. Selesai (resolve) begitu stream server ditutup (event
// "done" sudah pasti dikirim sebelum itu oleh claude-chat-service, kecuali
// koneksi putus paksa).
export async function streamSetupToken(onEvent: (e: SetupTokenEvent) => void, signal?: AbortSignal): Promise<void> {
  const token = getAccessToken();
  const res = await fetch('/api/v1/chat/auth/setup-token', {
    method: 'POST',
    credentials: 'include',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    signal
  });
  if (!res.ok || !res.body) {
    const text = await res.text().catch(() => '');
    throw new Error(extractErrorMessage(res.status, text));
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    let sepIndex: number;
    while ((sepIndex = buffer.indexOf('\n\n')) !== -1) {
      const raw = buffer.slice(0, sepIndex);
      buffer = buffer.slice(sepIndex + 2);
      const evt = parseSseMessage(raw);
      if (evt) onEvent(evt);
    }
  }
}
