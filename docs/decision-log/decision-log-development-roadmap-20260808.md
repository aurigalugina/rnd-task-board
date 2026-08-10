# Decision Log — Urutan Roadmap Pengembangan

**Tanggal:** 2026-08-08
**Konteks:** development-roadmap

## Konteks/Masalah

Backend baru mengimplementasikan 3 dari ~8 modul yang didokumentasikan di `05-api-contract.md` (`board`, `bigtask`, `dailytask`), dan frontend baru berupa shell navigasi tanpa satu pun form fungsional atau halaman login. Auth middleware (`RequireAuth`) baru memvalidasi token, belum menegakkan role. Perlu urutan pengerjaan yang jelas supaya tim tahu prioritas, dan supaya AI assistant di sesi berikutnya tidak menebak-nebak prioritas sendiri.

## Keputusan

Urutan pengerjaan disepakati sebagai berikut:

| Fase | Scope |
|---|---|
| 0 | Auth & RBAC hardening — refresh token via httpOnly cookie sesuai kontrak, middleware `RequireRole`, halaman login frontend |
| 1 | Wiring frontend ke 3 modul yang backend-nya sudah lengkap (board/bigtask/dailytask + sign-off/undo-signoff UI) |
| 2 | Modul Users & Roles (backend CRUD + halaman Settings/admin) |
| 3 | Comments (FR-CMT) |
| 4 | Cheat Sheet + endpoint upload (FR-REF) |
| 5 | Clone-as-review (FR-DLY-07) |
| 6 | Weekly Plan + push HR — disimulasikan/dicatat lokal saja (Fase 2 integrasi HR asli di luar cakupan, lihat `04-architecture.md` §5.2) |
| 7 | Review Queue & Notifikasi (FR-NTF) — lihat [[decision-log-review-queue-schema-20260808]] untuk keputusan skema pendukungnya |
| 8 | Testing — prioritaskan logika computed murni (`computeVerdict`, `expected_pct`, weekly rollup) karena paling gampang salah diam-diam dan paling sering disebut sebagai prinsip tidak bisa ditawar di BRD |

## Alasan

- **Fase 0 duluan** karena tanpa role enforcement, NFR-04 (otorisasi berbasis role) belum terpenuhi sama sekali walau backend "kelihatan" jalan — dan semua modul lain butuh identitas user yang valid (author_id, signed_by, pushed_by, dst).
- **Fase 1 sebelum modul-modul lain** karena backend-nya sudah lengkap tapi belum ada satu pun UI — ini risiko bisnis tertinggi yang eksplisit disebut di `02-brd.md` §12: tim balik pakai spreadsheet paralel kalau interaksi harian dirasa lebih ribet dari solusi lama. Menyambungkan yang sudah ada memberi nilai tercepat.
- **Fase 2 (users/roles) sebelum modul kolaboratif lain** karena assignment PIC di Big Task/Daily Task saat ini cuma bisa diisi manual via UUID — perlu daftar user nyata dulu sebelum modul lain (comments/mention, cheat-sheet author) terasa berguna.
- **Fase 3–6** relatif independen satu sama lain dan tidak saling blocking, diurutkan sesuai urutan modul di `05-api-contract.md`.
- **Fase 7 di akhir** karena butuh keputusan skema baru dulu (lihat decision log terpisah) — bukan sekadar mengikuti pola modul yang sudah ada seperti fase lain.
- **Testing di fase 8, bukan disebar dari awal**, karena skala tim kecil (NFR-01) dan supaya iterasi awal tidak melambat sebelum bentuk akhir modul-modul stabil — tapi begitu masuk fase 8, testing tidak boleh diskip.

## Dampak/File terpengaruh

- Tidak ada perubahan kode langsung dari keputusan ini — murni penentuan urutan kerja.
- `CLAUDE.md` tidak diubah karena keputusan ini tidak mengubah pola arsitektur yang sudah didokumentasikan di sana, hanya urutan eksekusi.
