// Auth store — session di-restore saat app dimuat via refresh token cookie
// (httpOnly, tidak diakses langsung dari sini). Access token hanya disimpan
// di memori (module-level state di client.ts), tidak pernah di localStorage.
import { writable } from 'svelte/store';
import { api, setAccessToken } from '$lib/api/client';

export type UserSummary = {
  id: string;
  display_name: string;
  initials: string;
  roles: string[];
  org_team: string;
  theme_preference?: string;
};

type AuthState = {
  status: 'idle' | 'loading' | 'authenticated' | 'unauthenticated';
  user: UserSummary | null;
};

// Cache non-sensitive display info saja (bukan token) supaya nama/role tidak
// hilang saat reload sebelum silent-refresh selesai. Kalau cache kosong tapi
// refresh tetap berhasil (mis. cache di-clear manual, atau baru pertama kali
// login di tab ini setelah reload panjang), init() fallback ke GET /users/me
// (docs/decision-log/decision-log-users-api-gaps-20260809.md).
const STORAGE_KEY = 'np_user';

function loadCachedUser(): UserSummary | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function cacheUser(user: UserSummary | null) {
  if (user) sessionStorage.setItem(STORAGE_KEY, JSON.stringify(user));
  else sessionStorage.removeItem(STORAGE_KEY);
}

const { subscribe, set } = writable<AuthState>({ status: 'idle', user: null });

async function init() {
  set({ status: 'loading', user: null });
  try {
    const res = await api.post<{ access_token: string }>('/auth/refresh');
    setAccessToken(res.access_token);

    let user = loadCachedUser();
    if (!user) {
      user = await api.get<UserSummary>('/users/me');
      cacheUser(user);
    }
    set({ status: 'authenticated', user });
  } catch {
    setAccessToken(null);
    cacheUser(null);
    set({ status: 'unauthenticated', user: null });
  }
}

async function login(email: string, password: string) {
  const res = await api.post<{ access_token: string; user: UserSummary }>('/auth/login', {
    email,
    password
  });
  setAccessToken(res.access_token);
  cacheUser(res.user);
  set({ status: 'authenticated', user: res.user });
}

async function logout() {
  try {
    await api.post('/auth/logout');
  } finally {
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
