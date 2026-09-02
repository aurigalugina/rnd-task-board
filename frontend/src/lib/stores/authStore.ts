// Auth store — session di-restore saat app dimuat via refresh token cookie
// (httpOnly, tidak diakses langsung dari sini). Access token hanya disimpan
// di memori (module-level state di client.ts), tidak pernah di localStorage.
//
// Session lifecycle (2026-09-01): access token umur 2 jam (backend
// auth.go accessTokenTTL), refresh token umur 7 hari (httpOnly cookie).
// Sebelumnya access token expired di tengah sesi cuma menghasilkan error
// generik di halaman yang sama (401 dari request mana pun), user gak tau
// harus login ulang. Sekarang dua lapis:
//   1. PROAKTIF: timer refresh access token setiap ~100 menit (di bawah TTL
//      2 jam) di background, selama status authenticated -- idealnya user
//      TIDAK PERNAH kena expired di tengah kerja.
//   2. REAKTIF (fallback): client.ts request() auto-refresh-and-retry saat
//      dapat 401 (lihat setSessionHandlers). Kalau refresh token JUGA sudah
//      mati (7 hari lewat / revoked), onExpired() di bawah set status
//      'unauthenticated' -- +layout.svelte react ke situ dan redirect ke
//      /login otomatis (lihat `$: if ($auth.status === 'unauthenticated' ...)`).
import { writable } from 'svelte/store';
import { api, setAccessToken, setSessionHandlers } from '$lib/api/client';
import { resetAll } from '$lib/stores/chatSessionStore';

export type UserSummary = {
  id: string;
  display_name: string;
  initials: string;
  roles: string[];
  org_team: string;
  theme_preference?: string;
  // access_level ('super_user'/'regular_user') -- konsep terpisah dari roles
  // (many-to-many), lihat docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
  access_level?: string;
  // task_scope_visibility ('self'/'team') -- 'team' = akses "lihat semua
  // orang" (dipakai gate edit Team Today, decision-log-team-today-edit-permission-20260902.md).
  task_scope_visibility?: string;
};

type AuthState = {
  status: 'idle' | 'loading' | 'authenticated' | 'unauthenticated';
  user: UserSummary | null;
  // Alasan terakhir kenapa jadi unauthenticated -- dipakai login page buat
  // nampilin pesan yang tepat ("sesi habis" vs baru buka app / logout manual).
  expiredReason: 'session_expired' | null;
};

// Cache non-sensitive display info saja (bukan token) -- CUMA ditulis di sini
// (dibaca lagi kalau nanti dibutuhkan), TIDAK dipakai buat skip fetch di
// init() lagi (lihat gotcha di bawah).
const STORAGE_KEY = 'np_user';

function cacheUser(user: UserSummary | null) {
  if (user) sessionStorage.setItem(STORAGE_KEY, JSON.stringify(user));
  else sessionStorage.removeItem(STORAGE_KEY);
}

const { subscribe, set, update } = writable<AuthState>({ status: 'idle', user: null, expiredReason: null });

// Timer proaktif refresh access token -- lihat catatan lapis 1 di atas.
// 100 menit dipilih supaya ada margin cukup sebelum TTL 2 jam habis (kalau
// refresh gagal karena network hiccup sesaat, masih ada ~20 menit sebelum
// token beneran expired dan fallback reaktif di client.ts turun tangan).
const PROACTIVE_REFRESH_MS = 100 * 60 * 1000;
let refreshTimer: ReturnType<typeof setInterval> | null = null;

function startProactiveRefresh() {
  stopProactiveRefresh();
  refreshTimer = setInterval(async () => {
    try {
      await doRefresh();
    } catch {
      // Refresh token juga sudah mati -- doRefresh() sendiri sudah panggil
      // forceExpire(), timer dihentikan di sana (lewat stopProactiveRefresh
      // yang dipanggil forceExpire). Tidak perlu apa-apa lagi di sini.
    }
  }, PROACTIVE_REFRESH_MS);
}

function stopProactiveRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
}

// doRefresh dipakai bersama oleh timer proaktif DAN fallback reaktif
// (setSessionHandlers.refresh di bawah) -- satu jalur, bukan dua
// implementasi terpisah yang bisa divergen.
async function doRefresh(): Promise<string> {
  const res = await api.post<{ access_token: string }>('/auth/refresh');
  setAccessToken(res.access_token);
  return res.access_token;
}

function forceExpire() {
  stopProactiveRefresh();
  setAccessToken(null);
  cacheUser(null);
  set({ status: 'unauthenticated', user: null, expiredReason: 'session_expired' });
}

setSessionHandlers({
  refresh: doRefresh,
  onExpired: forceExpire
});

// Gotcha ketemu 2026-08-10: init() dulu skip fetch /users/me kalau ada cache
// sessionStorage ("np_user") dari login sebelumnya -- tapi +layout.svelte
// TIDAK render apapun dari cache itu selama status 'loading' (cuma teks
// "Memuat sesi..."), jadi cache-nya nggak buat mencegah flicker apapun,
// cuma nyimpen data BASI kalau bentuk User berubah di backend (kejadian nyata:
// access_level ditambah, user yang sesinya udah lama gak bisa lihat fitur
// super_user sampai logout manual, walau DB & JWT-nya udah benar). Fix:
// SELALU fetch fresh /users/me di init(), jangan trust cache lagi.
async function init() {
  set({ status: 'loading', user: null, expiredReason: null });
  try {
    await doRefresh();
    const user = await api.get<UserSummary>('/users/me');
    cacheUser(user);
    set({ status: 'authenticated', user, expiredReason: null });
    startProactiveRefresh();
  } catch {
    setAccessToken(null);
    cacheUser(null);
    set({ status: 'unauthenticated', user: null, expiredReason: null });
  }
}

async function login(email: string, password: string) {
  const res = await api.post<{ access_token: string; user: UserSummary }>('/auth/login', {
    email,
    password
  });
  setAccessToken(res.access_token);
  cacheUser(res.user);
  set({ status: 'authenticated', user: res.user, expiredReason: null });
  startProactiveRefresh();
}

async function logout() {
  try {
    await api.post('/auth/logout');
  } finally {
    // Pastikan sesi chat Change Request tidak menggantung setelah logout
    // (state-nya hidup di level modul, bukan komponen — lihat chatSessionStore).
    resetAll();
    stopProactiveRefresh();
    setAccessToken(null);
    cacheUser(null);
    set({ status: 'unauthenticated', user: null, expiredReason: null });
  }
}

// refreshUser dipanggil setelah PATCH /users/me sukses, supaya store & cache
// ikut ter-update tanpa perlu logout/login ulang.
async function refreshUser() {
  const user = await api.get<UserSummary>('/users/me');
  cacheUser(user);
  update((s) => ({ ...s, user }));
}

export const auth = { subscribe, init, login, logout, refreshUser };
