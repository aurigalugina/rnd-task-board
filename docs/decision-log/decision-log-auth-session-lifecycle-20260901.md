# Auth: Session Lifecycle -- Proactive + Reactive Access Token Refresh

**Date:** 2026-09-01
**Status:** Implemented

## Konteks / Masalah

User (Lugi) menanyakan: sistem auth pakai access token? Kalau access token
expired, apakah sudah force logout?

Audit kode menemukan:
- **Ya**, sistem pakai access token JWT (umur 2 jam, in-memory di frontend,
  TIDAK di localStorage) + refresh token JWT (umur 7 hari, httpOnly cookie).
- **Belum** ada force logout. Ketika access token expired di tengah sesi,
  `client.ts` `request()` cuma `throw new Error(...)` dari response 401 --
  tidak ada auto-refresh, tidak ada redirect ke `/login`. User tetap di
  halaman yang sama, setiap aksi (klik tombol, load data) gagal dengan
  pesan error generik tanpa penjelasan bahwa solusinya login ulang.

## Keputusan

Dua lapis penanganan session expiry, dipilih dari 3 opsi yang ditawarkan ke
user (auto-refresh reaktif saja / force-logout langsung / proactive +
reactive) -- user tidak merespons dalam batas waktu, jadi diputuskan opsi
**paling robust** (proactive + reactive) karena dampaknya murni peningkatan
UX tanpa risiko keamanan baru, dan sejalan dengan filosofi codebase yang
sudah ada (refresh token 7 hari sengaja panjang supaya user jarang perlu
login ulang manual).

### Lapis 1 -- PROAKTIF (authStore.ts)
Timer `setInterval` refresh access token setiap **100 menit** (di bawah TTL
2 jam) selama status `authenticated`. Idealnya user TIDAK PERNAH mengalami
401 karena token expired di tengah kerja -- token selalu di-refresh sebelum
sempat kedaluwarsa. Timer dimulai di `login()` dan `init()` (setelah
authenticated), dihentikan di `logout()` dan `forceExpire()`.

### Lapis 2 -- REAKTIF/fallback (client.ts)
`request()` sekarang menangkap 401, mencoba **satu kali** refresh access
token via `sessionHandlers.refresh()` (didaftarkan dari authStore.ts lewat
`setSessionHandlers()`, menghindari circular import), lalu retry request
asli dengan token baru. Kalau refresh JUGA gagal (refresh token 7 hari
sudah expired/dicabut), `sessionHandlers.onExpired()` dipanggil ->
authStore.ts set status `unauthenticated` dengan `expiredReason:
'session_expired'` -> `+layout.svelte` (sudah ada logic ini, tidak diubah)
otomatis redirect ke `/login` -> halaman login menampilkan pesan "Sesi Anda
sudah habis, silakan masuk kembali."

**Safety rails:**
- `AUTH_BYPASS_PATHS` (`/auth/login`, `/auth/refresh`, `/auth/logout`) TIDAK
  memicu alur refresh-and-retry -- 401 di `/auth/login` berarti kredensial
  salah (bukan token expired), dan retry di `/auth/refresh` sendiri akan
  jadi infinite loop.
- `isRetry` param mencegah retry berulang -- maksimal 1x retry per request,
  kalau retry-nya sendiri masih 401 (refresh token ternyata basi tapi
  refresh() call sukses dengan token yang somehow masih gak valid), error
  asli dilempar apa adanya, tidak loop.
- `refreshPromise` dedup -- kalau beberapa request nembak bersamaan pas
  token baru expired, semuanya menunggu SATU panggilan `/auth/refresh`
  yang sama, bukan masing-masing manggil refresh sendiri (race condition +
  redundant network call).

## Alasan

- Access token 2 jam adalah TTL pendek yang disengaja (mengurangi window
  serangan kalau token bocor) -- tapi tanpa mekanisme refresh yang proper,
  itu jadi UX buruk (user kerja lama bisa "putus" tanpa peringatan).
- Refresh token 7 hari via httpOnly cookie sudah didesain untuk
  memungkinkan sesi panjang tanpa re-login manual tiap hari -- fix ini
  akhirnya benar-benar memanfaatkan desain itu (sebelumnya cuma dipakai di
  `init()` saat app pertama dimuat, bukan selama sesi berjalan).
- Proactive (lapis 1) dipilih di atas reactive-saja supaya user nyaris
  tidak pernah "merasakan" refresh terjadi -- kegagalan request karena 401
  di tengah kerja itu tetap mengganggu meski auto-retry, mencegahnya lebih
  baik daripada menanganinya.
- Reactive (lapis 2) tetap dipertahankan sebagai fallback -- proactive
  timer bisa gagal (network down pas jadwal refresh, tab di-suspend
  browser, dst), jadi tidak boleh jadi satu-satunya mekanisme.

## Dampak / File Terpengaruh

- `frontend/src/lib/api/client.ts` -- `setSessionHandlers()`, alur
  refresh-and-retry di `request()`, `AUTH_BYPASS_PATHS`, dedup
  `refreshPromise`.
- `frontend/src/lib/stores/authStore.ts` -- `doRefresh()`, `forceExpire()`,
  proactive timer (`startProactiveRefresh`/`stopProactiveRefresh`),
  `AuthState.expiredReason`, register `setSessionHandlers` saat modul
  di-load. Juga dibersihkan: `console.log`/`console.error` debug yang
  ketinggalan dari sesi troubleshooting login sebelumnya.
- `frontend/src/routes/login/+page.svelte` -- notice "Sesi Anda sudah
  habis, silakan masuk kembali." saat `expiredReason === 'session_expired'`.
  Juga dibersihkan debug console.log yang sama.
- Tidak ada perubahan backend -- `accessTokenTTL`/`refreshTokenTTL` di
  `auth.go` tidak diubah, endpoint `/auth/refresh` sudah ada dan berfungsi
  benar, cuma belum dipakai proaktif dari frontend.

## Verifikasi

- `npm run check` -- 0 errors.
- `npm run test` -- 129/129 passed, termasuk test baru
  `frontend/src/lib/api/client.test.ts` (7 test kasus: retry sukses, retry
  gagal -> onExpired, no-loop pada retry gagal kedua, bypass /auth/login,
  bypass /auth/refresh, dedup refresh untuk request bersamaan, error
  non-401 tetap normal).
- Manual: `curl` login + `/auth/refresh` terhadap backend lokal -- HTTP 200
  keduanya, access token baru valid.
- Build & restart Docker lokal -- OK.

## Alternatif yang Ditolak

- **Reactive-only** (opsi 1 dari 3 yang ditawarkan): lebih simpel tapi user
  tetap merasakan jeda/retry setiap ~2 jam pas kerja panjang tanpa jeda.
- **Force-logout langsung tanpa refresh** (opsi 2): paling simpel tapi
  paling mengganggu -- user re-login paksa tiap 2 jam walau refresh token
  masih 7 hari valid, membuang manfaat desain refresh token yang sudah ada.
