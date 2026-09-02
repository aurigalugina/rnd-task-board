// Klien API tipis untuk backend Go. Lihat 05-api-contract.md untuk kontrak lengkap.

const BASE_URL = '/api/v1';

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
  accessToken = token;
}

// Dibutuhkan buat menyusun URL WebSocket ke chat proxy — browser WebSocket tak
// bisa nyetel header Authorization, jadi token dikirim via query ?access_token=
// (divalidasi backend chatproxy). Lihat lib/chatClient.ts.
export function getAccessToken(): string | null {
  return accessToken;
}

// Session lifecycle (refresh access token / force logout) SENGAJA didaftarkan
// dari authStore.ts, bukan diimplementasikan di sini -- client.ts cuma urusan
// HTTP mechanics, authStore.ts yang punya state sesi (writable store, cache
// sessionStorage, dst). Menghindari circular import (authStore.ts sudah
// import `api` dari file ini) sekaligus pemisahan tanggung jawab yang jelas.
type SessionHandlers = {
  // refresh() harus resolve access token BARU (dan sudah memanggil
  // setAccessToken sendiri), atau reject kalau refresh token juga sudah mati.
  refresh: () => Promise<string>;
  // onExpired() dipanggil SEKALI ketika refresh gagal -- authStore.ts pakai
  // ini buat bersihkan state & set status 'unauthenticated' (redirect ke
  // /login diurus +layout.svelte yang react ke status itu).
  onExpired: () => void;
};
let sessionHandlers: SessionHandlers | null = null;
export function setSessionHandlers(handlers: SessionHandlers) {
  sessionHandlers = handlers;
}

// Endpoint yang TIDAK BOLEH memicu alur refresh-and-retry di bawah -- 401 di
// /auth/login berarti kredensial salah (bukan token expired), 401 di
// /auth/refresh atau /auth/logout berarti refresh token ITU SENDIRI yang
// sudah mati (retry lewat refresh() lagi cuma bikin loop).
const AUTH_BYPASS_PATHS = ['/auth/login', '/auth/refresh', '/auth/logout'];

// Dedup refresh -- kalau beberapa request nembak bersamaan pas access token
// baru saja expired, semua 401 itu HARUS menunggu satu panggilan
// /auth/refresh yang sama (bukan masing-masing manggil refresh sendiri-sendiri,
// yang berpotensi race/redundant network call).
let refreshPromise: Promise<string> | null = null;

async function request<T>(path: string, options: RequestInit = {}, isRetry = false): Promise<T> {
  // FormData (upload multipart) harus TANPA Content-Type manual — browser yang
  // set header itu sendiri termasuk boundary-nya, kalau di-override jadi rusak.
  const isFormData = options.body instanceof FormData;

  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    // refresh token dikirim sebagai httpOnly cookie (05-api-contract.md §2) — wajib
    // 'include' supaya /auth/refresh dan /auth/logout ikut mengirim/menerima cookie itu.
    credentials: 'include',
    headers: {
      ...(isFormData ? {} : { 'Content-Type': 'application/json' }),
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      ...(options.headers ?? {})
    }
  });

  // Access token expired di tengah sesi (umur 2 jam, lihat auth.go
  // accessTokenTTL) -- coba refresh SEKALI pakai refresh token cookie, lalu
  // retry request asli dengan token baru. Kalau refresh-nya sendiri gagal
  // (refresh token juga expired/dicabut), forceLogout via onExpired() supaya
  // user diarahkan ke /login alih-alih macet dengan error samar di halaman
  // yang sama. isRetry mencegah retry berulang (max 1x per request).
  if (res.status === 401 && !isRetry && sessionHandlers && !AUTH_BYPASS_PATHS.includes(path)) {
    try {
      if (!refreshPromise) refreshPromise = sessionHandlers.refresh();
      const newToken = await refreshPromise;
      refreshPromise = null;
      accessToken = newToken;
      return request<T>(path, options, true);
    } catch {
      refreshPromise = null;
      sessionHandlers.onExpired();
      throw new Error('Sesi habis, silakan login ulang.');
    }
  }

  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error?.message ?? `Request gagal: ${res.status}`);
  }

  // Beberapa endpoint 200 bisa saja tanpa body (deviasi dari kontrak) — res.json()
  // pada body kosong throw "Unexpected end of JSON input" dan bikin caller salah
  // kira request-nya gagal padahal sukses. Baca sebagai teks dulu baru parse.
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PATCH', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PUT', body: body ? JSON.stringify(body) : undefined }),
  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  upload: <T>(path: string, formData: FormData) => request<T>(path, { method: 'POST', body: formData })
};

// Dipakai khusus buat unduh file terproteksi (mis. GET /uploads/{filename}) —
// `<a href>` biasa TIDAK bisa bawa Bearer token (browser cuma otomatis
// ngirim cookie di navigasi biasa, bukan header custom), jadi endpoint yang
// butuh Authorization header harus di-fetch manual lalu di-download via Blob.
export async function downloadBlob(path: string): Promise<Blob> {
  const res = await fetch(`${BASE_URL}${path}`, {
    credentials: 'include',
    headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
  });
  if (!res.ok) throw new Error(`Gagal mengunduh: ${res.status}`);
  return res.blob();
}
