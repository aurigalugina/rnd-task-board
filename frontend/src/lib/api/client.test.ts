// Test untuk alur refresh-and-retry di request() (client.ts) -- lihat catatan
// session lifecycle di authStore.ts. Fokus: 401 memicu SATU kali refresh lalu
// retry, refresh gagal -> onExpired() dipanggil, endpoint auth (/auth/login,
// /auth/refresh, /auth/logout) TIDAK memicu alur ini sama sekali, dan
// beberapa 401 bersamaan cuma memicu SATU panggilan refresh (dedup).
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api, setAccessToken, setSessionHandlers } from './client';

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' }
  });
}

describe('api client — refresh-and-retry on 401', () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    setAccessToken('expired-token');
    fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    setAccessToken(null);
    // Reset handlers supaya test lain (atau import authStore.ts di test lain)
    // tidak diam-diam kepasang handler stale dari sini.
    setSessionHandlers({
      refresh: () => Promise.reject(new Error('no handler registered')),
      onExpired: () => {}
    });
  });

  it('401 lalu refresh sukses -> request asli di-retry otomatis dengan token baru', async () => {
    const refresh = vi.fn().mockResolvedValue('new-token');
    const onExpired = vi.fn();
    setSessionHandlers({ refresh, onExpired });

    fetchMock
      .mockResolvedValueOnce(jsonResponse({ error: { message: 'expired' } }, 401))
      .mockResolvedValueOnce(jsonResponse({ ok: true }, 200));

    const result = await api.get<{ ok: boolean }>('/boards');

    expect(result).toEqual({ ok: true });
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(onExpired).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    // Retry harus pakai token BARU, bukan token lama yang sudah expired.
    const retryHeaders = fetchMock.mock.calls[1][1].headers as Record<string, string>;
    expect(retryHeaders.Authorization).toBe('Bearer new-token');
  });

  it('401 lalu refresh gagal -> onExpired dipanggil, error jelas dilempar ke caller', async () => {
    const refresh = vi.fn().mockRejectedValue(new Error('refresh token expired'));
    const onExpired = vi.fn();
    setSessionHandlers({ refresh, onExpired });

    fetchMock.mockResolvedValueOnce(jsonResponse({ error: { message: 'expired' } }, 401));

    await expect(api.get('/boards')).rejects.toThrow('Sesi habis, silakan login ulang.');
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(onExpired).toHaveBeenCalledTimes(1);
    // Tidak ada retry kedua ke /boards -- cuma 1 fetch (yang 401 awal).
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('request diulang tepat SEKALI -- 401 lagi setelah retry tidak memicu loop refresh', async () => {
    const refresh = vi.fn().mockResolvedValue('new-token-but-still-401');
    const onExpired = vi.fn();
    setSessionHandlers({ refresh, onExpired });

    fetchMock
      .mockResolvedValueOnce(jsonResponse({ error: { message: 'expired' } }, 401))
      .mockResolvedValueOnce(jsonResponse({ error: { message: 'still unauthorized' } }, 401));

    await expect(api.get('/boards')).rejects.toThrow('still unauthorized');
    expect(refresh).toHaveBeenCalledTimes(1); // bukan berkali-kali/infinite loop
    expect(fetchMock).toHaveBeenCalledTimes(2); // 1 awal + 1 retry, berhenti di situ
  });

  it('/auth/login TIDAK memicu refresh-and-retry -- 401 di situ berarti kredensial salah', async () => {
    const refresh = vi.fn();
    const onExpired = vi.fn();
    setSessionHandlers({ refresh, onExpired });

    fetchMock.mockResolvedValueOnce(jsonResponse({ error: { message: 'Email atau password salah' } }, 401));

    await expect(api.post('/auth/login', { email: 'a@b.com', password: 'x' })).rejects.toThrow(
      'Email atau password salah'
    );
    expect(refresh).not.toHaveBeenCalled();
    expect(onExpired).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('/auth/refresh TIDAK memicu refresh-and-retry -- mencegah loop refresh-of-refresh', async () => {
    const refresh = vi.fn();
    const onExpired = vi.fn();
    setSessionHandlers({ refresh, onExpired });

    fetchMock.mockResolvedValueOnce(jsonResponse({ error: { message: 'invalid refresh token' } }, 401));

    await expect(api.post('/auth/refresh')).rejects.toThrow('invalid refresh token');
    expect(refresh).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('beberapa request 401 bersamaan -- refresh cuma dipanggil SEKALI (dedup)', async () => {
    const refresh = vi.fn().mockResolvedValue('new-token');
    setSessionHandlers({ refresh, onExpired: vi.fn() });

    fetchMock
      .mockResolvedValueOnce(jsonResponse({ error: { message: 'expired' } }, 401)) // req A awal
      .mockResolvedValueOnce(jsonResponse({ error: { message: 'expired' } }, 401)) // req B awal
      .mockResolvedValueOnce(jsonResponse({ a: 1 }, 200)) // req A retry
      .mockResolvedValueOnce(jsonResponse({ b: 1 }, 200)); // req B retry

    const [a, b] = await Promise.all([api.get('/boards'), api.get('/users/me')]);

    expect(a).toEqual({ a: 1 });
    expect(b).toEqual({ b: 1 });
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it('respons non-401 yang gagal tetap melempar error seperti biasa (bukan alur refresh)', async () => {
    const refresh = vi.fn();
    setSessionHandlers({ refresh, onExpired: vi.fn() });

    fetchMock.mockResolvedValueOnce(jsonResponse({ error: { message: 'validasi gagal' } }, 400));

    await expect(api.post('/boards', { name: '' })).rejects.toThrow('validasi gagal');
    expect(refresh).not.toHaveBeenCalled();
  });
});
