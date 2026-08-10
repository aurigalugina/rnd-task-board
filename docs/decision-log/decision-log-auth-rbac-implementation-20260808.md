# Decision Log — Implementasi Refresh Token & Role Enforcement

**Tanggal:** 2026-08-08
**Konteks:** auth-rbac-implementation

## Konteks/Masalah

`04-architecture.md` §7 dan `05-api-contract.md` §2 menyebutkan refresh token "tersimpan sebagai HTTP-only cookie", tapi tidak merinci apakah refresh token itu stateless (JWT lain) atau dipersist di tabel (mis. `refresh_tokens`). `06-db-design.md` tidak punya tabel semacam itu. Begitu juga cara `RequireRole` middleware (Fase 0) tahu role suatu user — apakah query DB tiap request, atau role di-embed ke JWT claims saat login.

## Keputusan

1. **Refresh token = JWT kedua yang juga stateless** (bukan dipersist di tabel baru), dengan claim `{"sub": userID, "typ": "refresh", "exp": now+7d}`, ditandatangani pakai secret yang sama (`JWT_SECRET`). Dikirim via `Set-Cookie` httpOnly, `Path=/api/v1/auth`, `SameSite=Lax`, `Secure` (kondisional — nonaktif kalau `APP_ENV=development` supaya jalan di `http://localhost` tanpa TLS).
2. **Role di-embed ke claim access token saat login** (`"roles": ["spv","dev"]`), bukan di-query ulang ke DB tiap request oleh `RequireAuth`. `POST /auth/refresh` query ulang role dari DB saat mint access token baru, supaya perubahan role user ter-refresh maksimal setiap masa berlaku access token (2 jam).
3. **Logout = clear cookie saja** (`Set-Cookie` dengan `Max-Age=-1`, path & atribut sama persis dengan saat di-set), tidak ada revocation list. Access token lama tetap valid sampai expire (2 jam) meski sudah logout — trade-off yang diterima untuk skala tim kecil ini.

## Alasan

- **Stateless (bukan tabel `refresh_tokens`)** konsisten dengan keputusan arsitektur yang sudah difinalisasi di `04-architecture.md` §6 ("JWT (stateless auth) — Menghindari kebutuhan session store terpisah pada skala pengguna kecil ini"). Menambah tabel session/refresh-token akan bertentangan dengan alasan itu tanpa kebutuhan yang jelas pada skala tim ini.
- **Role di-claim, bukan di-query ulang tiap request** untuk menghindari satu query DB ekstra di setiap request terautentikasi (kontributor akan sering hit endpoint inline-update Day Entry — NFR-02 mensyaratkan respons < 300ms). Window staleness maksimal 2 jam (umur access token) dianggap dapat diterima untuk perubahan role, yang tidak sering terjadi di tim kecil ini.
- **Tidak ada revocation list** untuk menghindari kompleksitas/state tambahan yang sama-sama ingin dihindari oleh keputusan JWT stateless — kalau kelak dibutuhkan (mis. ada insiden akun dibobol), ini titik yang perlu direvisit sebagai decision log baru, bukan diam-diam ditambahkan.

## Dampak/File terpengaruh

- `backend/internal/auth/auth.go` — tambah generate/parse refresh token, embed roles ke access token claims, `RequireRole` middleware baru, cookie handling di Login/Logout/Refresh.
- `backend/cmd/api/main.go` — pasang `RequireRole("spv")` di endpoint sign-off/undo-signoff.
- `CLAUDE.md` — bagian "Auth" perlu disinkronkan setelah implementasi selesai (role enforcement sudah tidak lagi jadi gap).
