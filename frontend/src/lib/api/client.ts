// Klien API tipis untuk backend Go. Lihat 05-api-contract.md untuk kontrak lengkap.

const BASE_URL = '/api/v1';

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
  accessToken = token;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
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
