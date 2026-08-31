// Auth store — session di-restore saat app dimuat via refresh token cookie
// (httpOnly, tidak diakses langsung dari sini). Access token hanya disimpan
// di memori (module-level state di client.ts), tidak pernah di localStorage.
import { writable } from 'svelte/store';
import { api, setAccessToken } from '$lib/api/client';
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
};

type AuthState = {
  status: 'idle' | 'loading' | 'authenticated' | 'unauthenticated';
  user: UserSummary | null;
};

// Cache non-sensitive display info saja (bukan token) -- CUMA ditulis di sini
// (dibaca lagi kalau nanti dibutuhkan), TIDAK dipakai buat skip fetch di
// init() lagi (lihat gotcha di bawah).
const STORAGE_KEY = 'np_user';

function cacheUser(user: UserSummary | null) {
  if (user) sessionStorage.setItem(STORAGE_KEY, JSON.stringify(user));
  else sessionStorage.removeItem(STORAGE_KEY);
}

const { subscribe, set } = writable<AuthState>({ status: 'idle', user: null });

// Gotcha ketemu 2026-08-10: init() dulu skip fetch /users/me kalau ada cache
// sessionStorage ("np_user") dari login sebelumnya -- tapi +layout.svelte
// TIDAK render apapun dari cache itu selama status 'loading' (cuma teks
// "Memuat sesi..."), jadi cache-nya nggak buat mencegah flicker apapun,
// cuma nyimpen data BASI kalau bentuk User berubah di backend (kejadian nyata:
// access_level ditambah, user yang sesinya udah lama gak bisa lihat fitur
// super_user sampai logout manual, walau DB & JWT-nya udah benar). Fix:
// SELALU fetch fresh /users/me di init(), jangan trust cache lagi.
async function init() {
  set({ status: 'loading', user: null });
  try {
    const res = await api.post<{ access_token: string }>('/auth/refresh');
    setAccessToken(res.access_token);

    const user = await api.get<UserSummary>('/users/me');
    cacheUser(user);
    set({ status: 'authenticated', user });
  } catch {
    setAccessToken(null);
    cacheUser(null);
    set({ status: 'unauthenticated', user: null });
  }
}

async function login(email: string, password: string) {
  console.log('[auth] Attempting login with email:', email);
  const res = await api.post<{ access_token: string; user: UserSummary }>('/auth/login', {
    email,
    password
  });
  console.log('[auth] Login response received:', { hasToken: !!res.access_token, user: res.user?.display_name });
  setAccessToken(res.access_token);
  cacheUser(res.user);
  set({ status: 'authenticated', user: res.user });
  console.log('[auth] Login complete, store updated');
}

async function logout() {
  try {
    await api.post('/auth/logout');
  } finally {
    // Pastikan sesi chat Change Request tidak menggantung setelah logout
    // (state-nya hidup di level modul, bukan komponen — lihat chatSessionStore).
    resetAll();
    setAccessToken(null);
    cacheUser(null);
    set({ status: 'unauthenticated', user: null });
  }
}

// refreshUser dipanggil setelah PATCH /users/me sukses, supaya store & cache
// ikut ter-update tanpa perlu logout/login ulang.
async function refreshUser() {
  const user = await api.get<UserSummary>('/users/me');
  cacheUser(user);
  set({ status: 'authenticated', user });
}

export const auth = { subscribe, init, login, logout, refreshUser };
