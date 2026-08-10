# Decision Log — Endpoint Daftar User untuk Assignment PIC

**Tanggal:** 2026-08-09
**Konteks:** pic-assignment-endpoint

## Konteks/Masalah

FR-ASG-02 (`03-srs.md`) mensyaratkan form assignment PIC (dipakai saat membuat Daily Task) menyediakan filter berdasarkan role sebelum memilih pengguna — ini berlaku untuk **semua kontributor** (`dev`, `sa`, `qa`, `admin`), bukan cuma SPV. Tapi satu-satunya endpoint yang mengembalikan daftar user, `GET /users`, didokumentasikan di `05-api-contract.md` §10 sebagai otorisasi **admin/spv saja**. Kalau dipakai apa adanya untuk form PIC picker, kontributor non-spv/admin akan kena 403 saat membuat Daily Task — bertentangan dengan FR-ASG-02.

## Keputusan

Tambah endpoint baru `GET /users/assignable` — bisa diakses **semua pengguna terautentikasi** (tidak dibatasi role), mengembalikan daftar ringkas `{ id, display_name, initials, roles, org_team }` (tanpa `email`). `GET /users` (otorisasi admin/spv, kontrak lama, termasuk `email`) tetap dipertahankan khusus untuk kebutuhan manajemen user (FR-USR-03/04).

## Alasan

- **Pemisahan berdasarkan kebutuhan data, bukan duplikasi sembarangan**: form PIC picker cuma butuh identitas+role buat filter (FR-ASG-02), sedangkan halaman manajemen user butuh data lebih lengkap (email, buat FR-USR-03). Memisahkan endpoint menghindari harus membuka data yang tidak perlu (email) ke seluruh kontributor sekaligus menghindari 403 yang menghalangi alur inti (assignment).
- **Tidak melonggarkan otorisasi `GET /users` yang sudah ada** — mengikuti prinsip "ubah seminim mungkin dari kontrak yang sudah difinalisasi" ketimbang mengubah semantik endpoint yang sudah dipakai bagian lain (halaman admin).
- Skala tim kecil (BRD §8: bukan skala organisasi besar) membuat exposure identitas+role antar sesama kontributor internal bukan risiko keamanan berarti — ini bukan data sensitif seperti password/email pribadi.

## Dampak/File terpengaruh

- `docs/05-api-contract.md` §10 — tambah dokumentasi `GET /users/assignable`.
- `backend/internal/user/handler.go` — handler baru `ListAssignable`.
- `backend/cmd/api/main.go` — route baru di grup `protected` (bukan grup admin-only).
- `frontend/src/lib/components/DailyTaskPanel.svelte` — form create Daily Task pakai endpoint ini buat dropdown PIC (menggantikan default diam-diam ke user yang sedang login).
