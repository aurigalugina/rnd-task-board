# Decision Log — Gap di Kontrak API Users

**Tanggal:** 2026-08-09
**Konteks:** users-api-gaps

## Konteks/Masalah

Mulai implementasi modul Users & Roles (Fase 2 roadmap, [[decision-log-development-roadmap-20260808]]) dan ketemu 3 gap di `05-api-contract.md` §10 yang baru kelihatan begitu mau benar-benar diimplementasikan:

1. `POST /users` (buat user baru) tidak punya field password sama sekali di request body — tidak jelas bagaimana user baru mendapat kredensial awal.
2. Tidak ada endpoint untuk membaca profil milik sendiri (`GET /users/me`) — hanya ada `PATCH /users/me`. Data user di frontend saat ini cuma didapat dari response `POST /auth/login`, di-cache di `sessionStorage`; kalau cache kosong tapi sesi masih valid (lewat refresh token), tidak ada cara mengambil ulang profil user.
3. `PATCH /users/me` untuk ganti password tidak mensyaratkan verifikasi password lama (`current_password`) — sesi yang dibajak bisa mengganti password tanpa tahu password asli, mengunci pemilik akun sesungguhnya.

## Keputusan

1. `POST /users` request ditambah field `password` (wajib, plain text di body — di-hash bcrypt di server sebelum simpan, sama seperti `password_hash` user lain). Admin/SPV yang membuat user menentukan password awal, disampaikan ke user baru secara out-of-band (lisan/chat internal tim) — tidak ada mekanisme email/invite pada iterasi ini (skala tim kecil, sesuai `02-brd.md` §8 asumsi).
2. Tambah `GET /users/me` — mengembalikan profil milik user yang sedang login (bentuk sama seperti `UserSummary` di `05-api-contract.md` §2, ditambah `theme_preference`). Dipanggil frontend saat `sessionStorage` cache kosong tapi status auth `authenticated`.
3. `PATCH /users/me` mensyaratkan `current_password` di body request kalau field `password` disertakan (ganti password). Request ditolak 401 kalau `current_password` tidak cocok. Field lain (`display_name`, `initials`, `theme_preference`) tidak butuh `current_password`.

## Alasan

- **Password di `POST /users`, bukan invite/email**: NFR-06 (portabilitas, tanpa dependensi layanan cloud pihak ketiga) dan skala tim kecil membuat flow email-invite tidak proporsional. Password wajib di body (bukan auto-generate lalu ditampilkan sekali) supaya admin yang membuat user bisa langsung komunikasikan kredensial dengan cara yang sudah biasa dipakai tim (chat internal).
- **`GET /users/me` melengkapi `PATCH /users/me`** yang sudah ada — pasangan get/update yang wajar, dan menutup gap nyata: tanpa ini, refresh token yang valid tapi cache browser kosong bikin frontend tidak punya cara menampilkan profil user yang benar.
- **`current_password` untuk ganti password** adalah mitigasi dasar terhadap sesi yang dibajak (access token dicuri lewat XSS dsb) — tanpa ini, penyerang yang punya access token bisa mengunci pemilik akun asli dari akunnya sendiri hanya dengan satu request. Ini konsisten dengan NFR-04 (keamanan akses) tanpa menambah kompleksitas berarti (satu query bcrypt tambahan).

## Dampak/File terpengaruh

- `docs/05-api-contract.md` §10 — update bentuk request `POST /users` dan `PATCH /users/me`, tambah dokumentasi `GET /users/me`.
- `backend/internal/user/` (modul baru) — implementasi ketiga endpoint di atas.
- `frontend/src/lib/stores/authStore.ts` — panggil `GET /users/me` sebagai fallback saat cache `sessionStorage` kosong.
