# Decision Log — Detail Implementasi Clone-as-Review

**Tanggal:** 2026-08-09
**Konteks:** clone-review

## Konteks/Masalah

`05-api-contract.md` §5 mendokumentasikan `POST /daily-tasks/{daily_task_id}/clone-review` dengan request `{ "role_tag": "SPV" | "QA" }`, tapi ada 2 gap yang baru kelihatan pas implementasi:

1. Deskripsinya bilang "Rentang tanggal input dari klien" tapi contoh request body cuma punya `role_tag` — tidak jelas apakah `start_date`/`end_date` memang bagian dari request body endpoint ini, atau alur sebenarnya cuma prefill form "Tambah Daily Task" biasa (ini yang dilakukan `docs/rnd-ops-mockup_3.jsx` — `handleReviewClick` di mockup cuma prefill title+PIC ke form tambah, TIDAK ada pemanggilan endpoint clone terpisah sama sekali, tanggal dikumpulkan lewat form yang sama).
2. FR-DLY-07 bilang "PIC default sesuai role terpilih" — tidak dijelaskan gimana kalau ada lebih dari satu user dengan role itu (mis. 2 orang QA).

## Keputusan

1. `start_date` dan `end_date` **memang bagian dari request body** endpoint `POST /daily-tasks/{daily_task_id}/clone-review` — bentuk lengkap: `{ "role_tag": "SPV"|"QA", "start_date", "end_date" }`. Endpoint ini benar-benar membuat Daily Task baru di server (bukan cuma prefill form di klien seperti mockup) — mockup adalah demo frontend murni tanpa backend asli, jadi pola interaksinya di titik ini tidak dijadikan acuan, kontrak API (yang menjanjikan response 201 dengan Daily Task object) yang jadi rujukan.
2. PIC default = user PERTAMA (urut `display_name` ASC) yang punya role `spv` (untuk `role_tag: "SPV"`) atau `qa` (untuk `role_tag: "QA"`) — deterministik, bukan acak. Kalau tidak ada satu pun user dengan role itu, request ditolak 400 (bukan diam-diam pakai PIC lain / PIC kosong).

## Alasan

- **Endpoint asli, bukan prefill-only**: kontrak API sudah menjanjikan response 201 berisi Daily Task object yang benar-benar tersimpan — konsisten dengan seluruh modul lain di aplikasi ini yang state-nya selalu di server, bukan cuma di form klien. Kalau cuma prefill, tidak ada bedanya dengan pengguna mengisi form "Tambah Daily Task" manual, jadi FR-DLY-07 (endpoint terpisah) jadi tidak ada gunanya.
- **PIC deterministik (bukan random/pertama-di-DB)** supaya hasil clone-review bisa diprediksi user (klik dua kali dengan role sama pada kondisi user yang sama akan selalu pilih PIC yang sama) — `display_name ASC` dipilih karena sederhana dan tidak butuh kolom tambahan.
- **Tolak 400 kalau role belum ada usernya**: mencegah Daily Task baru dibuat dengan PIC yang salah/kosong secara diam-diam — lebih baik gagal jelas di titik ini daripada menghasilkan data yang butuh dibenerin manual belakangan.

## Dampak/File terpengaruh

- `backend/internal/dailytask/handler.go` — handler baru `CloneReview`.
- `backend/cmd/api/main.go` — route baru `POST /daily-tasks/{dailyTaskID}/clone-review`.
- `frontend/src/lib/components/DailyTaskPanel.svelte` — tombol "Review SPV"/"Review QA" per Daily Task, submit lewat form kecil (tanggal) ke endpoint ini.
