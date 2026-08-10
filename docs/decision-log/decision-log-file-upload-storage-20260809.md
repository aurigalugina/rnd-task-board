# Decision Log — Penyimpanan File Upload (Cheat Sheet tipe `file`)

**Tanggal:** 2026-08-09
**Konteks:** file-upload-storage

## Konteks/Masalah

`05-api-contract.md` §10 mendokumentasikan `POST /uploads` (multipart) buat Cheat Sheet tipe `file`, tapi tidak merinci di mana file disimpan atau bagaimana file itu diambil kembali. `docker-compose.yml` saat ini tidak punya volume untuk backend selain database — kalau file ditulis ke filesystem container tanpa volume, hilang tiap kali container di-rebuild/restart.

## Keputusan

1. File disimpan di disk lokal container backend, path `/app/uploads` (env `UPLOAD_DIR`, default ke path itu kalau tidak diset), dipetakan ke docker volume baru `uploads` di `docker-compose.yml` — bukan S3/object storage cloud.
2. Nama file yang disimpan = `<uuid>_<nama-asli-yang-disanitasi>` — mencegah collision antar upload dengan nama sama, dan `filepath.Base()` dipakai buat strip komponen path apa pun dari nama asli (mencegah path traversal lewat nama file yang dimanipulasi).
3. File yang sudah diupload bisa diambil kembali lewat `GET /api/v1/uploads/{filename}` (endpoint baru, tidak didokumentasikan eksplisit di kontrak tapi konsisten dengan tujuan "referensi kerja" — cheat sheet tipe file percuma kalau tidak bisa dibuka lagi). Frontend menampilkan link unduh mengarah ke situ, bukan cuma teks nama file seperti di mockup.

## Alasan

- **Disk lokal, bukan cloud storage**: konsisten dengan NFR-06 (portabilitas, jalan penuh di `docker compose` tanpa dependensi layanan cloud pihak ketiga) — menambah S3/GCS akan melanggar prinsip itu tanpa kebutuhan yang jelas pada skala tim kecil ini.
- **Volume baru wajib**, bukan opsional — tanpa ini, upload hilang di restart pertama, jadi fitur "referensi" yang tidak bisa diandalkan malah lebih buruk daripada tidak ada.
- **UUID prefix + sanitasi nama** adalah mitigasi standar buat dua masalah nyata: dua file bernama sama dari orang berbeda saling menimpa, dan nama file yang berisi `../` bisa dipakai buat baca/tulis di luar folder upload.
- **Endpoint download ditambahkan** (bukan cuma catat metadata) karena tanpa ini fitur cheat sheet tipe file tidak ada gunanya — mockup menampilkannya sebagai teks statis, tapi itu keterbatasan mockup (data dummy, tidak ada file asli), bukan keputusan produk yang harus diikuti persis.

## Dampak/File terpengaruh

- `backend/internal/upload/handler.go` (modul baru) — `POST /uploads`, `GET /uploads/{filename}`.
- `docker-compose.yml` — volume baru `uploads:/app/uploads` di service `backend`.
- `backend/.env.example` — tambah `UPLOAD_DIR`.
- `frontend/src/lib/components/CheatSheetSection.svelte` (baru) — link unduh untuk tipe `file`.
