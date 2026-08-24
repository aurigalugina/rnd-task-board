# Decision Log — Big Task: Baseline Awal (Adjustment Persen Progress Migrasi Staging)

**Tanggal:** 2026-08-24
**Konteks:** bigtask-baseline-progress

## Konteks/Masalah

Saat migrasi awal ke staging/production, ada Big Task yang di lapangan sebenarnya sudah berjalan sebagian (mis. sudah 40%) sebelum data itu dicatat di sistem. `actual_pct` Big Task murni computed dari `AVG` per Daily Task lalu `AVG` antar-Daily Task (`backend/internal/bigtask/handler.go` `ListByBoard`/`loadBigTask`) — kalau Big Task baru dibuat tanpa Daily Task apa pun, `actual_pct`-nya 0%, tidak mencerminkan kondisi riil. User butuh cara input persentase awal ini supaya kelihatan benar di Dashboard, TANPA mengubah prinsip computed-field yang sudah dipegang project ini (tidak boleh ada kolom `actual_pct` tersimpan terpisah yang bisa desync dari day_entries — lihat `decision-log-day-entry-progress-pct-20260810.md`).

Concern eksplisit dari user: penyesuaian ini **tidak boleh mengubah/menyentuh Daily Task lain yang sudah berjalan/selesai** — termasuk yang progress-nya sudah 100%.

## Keputusan

- **Bukan kolom override terpisah.** Input persen baseline tetap di-cascade jadi day_entries beneran, konsisten dengan prinsip "satu sumber kebenaran" (`AVG(progress_pct)`) yang sudah dipakai di semua level (Daily Task/Big Task/Board/Dashboard).
- **Mekanisme:** super_user isi field "Persentase awal (opsional)" di form edit Big Task (`PATCH /big-tasks/{bigTaskID}`) **ATAU langsung di form create** (`POST /boards/{boardID}/big-tasks`) — dua jalur pakai logika upsert yang sama, gak perlu buka Edit lagi setelah Big Task baru dibuat. Backend otomatis:
  - Auto-create **satu Daily Task khusus** berjudul `Baseline Awal`, `pic_user_id = default_pic_user_id` (fallback ke anggota pertama kalau `default_pic_user_id` NULL), `start_date = end_date = big_tasks.start_date`, ditandai `is_baseline = true` (kolom boolean baru, bukan match string judul — biar robust kalau judul diedit/diterjemahkan nanti).
  - Satu `day_entries` di tanggal itu dengan `progress_pct = input user`.
  - Daily Task ini TETAP MUNCUL NORMAL di daftar Daily Task (bukan disembunyikan) — transparan, PIC/anggota bisa lihat itu darimana asalnya.
- **Re-edit:** kalau super_user ubah angka baseline lagi nanti (mis. 40% → 55%), sistem **UPDATE `progress_pct` day_entries yang sudah ada** (unique per Big Task via constraint `is_baseline`), BUKAN insert baris baru — mencegah history menumpuk/AVG tercampur ganda.
- **Tidak menyentuh Daily Task lain sama sekali.** Baseline murni Daily Task tambahan yang berdiri sendiri — Daily Task real (termasuk yang sudah 100%) day_entries-nya tidak pernah ditulis ulang oleh mekanisme ini.
- **Baseline permanen ikut rata-rata Big Task selamanya** (bukan cuma starting point yang di-exclude otomatis begitu Daily Task real jalan) — dikonfirmasi eksplisit ke user: baseline dianggap salah satu "unit kerja" juga, sama seperti Daily Task lain, bukan mekanisme sementara yang perlu di-drop.
- **Akses:** super_user only, konsisten dengan field sensitif lain di Big Task (edit board, backdate sign-off, dst).
- **Kosongkan/hapus baseline:** kalau field dikosongkan lagi di form edit (nilai `null`/dihapus), Daily Task + day_entries baseline **dihapus** (bukan di-set ke 0 — 0% dari "sengaja 0" beda makna dari "gak ada baseline sama sekali", dan biar gak nyisa Daily Task kosong 0% di daftar).

## Alasan

- **Sesuai prinsip computed-field project**: tidak menambah kolom `actual_pct`/override yang bisa desync. Baseline diwujudkan sebagai data day_entries beneran, sama seperti progress harian biasa — `AVG` yang sudah ada otomatis "just work" tanpa cabang logika baru di level agregasi.
- **Levelling via Daily Task struct yang sudah ada** menjaga isolasi terhadap Daily Task lain — karena agregasi Big Task adalah AVG-of-AVG (per Daily Task dulu, baru antar-Daily Task), baseline sebagai Daily Task terpisah cuma menambah SATU elemen tambahan ke rata-rata itu, tidak pernah menulis ulang Daily Task manapun yang sudah ada — memenuhi concern user secara struktural, bukan cuma janji di level aplikasi.
- **`is_baseline` boolean, bukan match judul string**: robust terhadap rename/typo, dan bikin constraint DB (unique per big_task_id where is_baseline) mudah dan predictable untuk UPDATE-bukan-INSERT.
- **UPDATE bukan INSERT baru saat re-edit**: mencegah drift AVG dari akumulasi baseline lama yang seharusnya sudah digantikan.

## Dampak/File Terpengaruh

- `backend/db/migrations/0024_bigtask_baseline_progress.up.sql`/`.down.sql` (baru) — `daily_tasks.is_baseline BOOLEAN NOT NULL DEFAULT false` + partial unique index `(big_task_id) WHERE is_baseline`.
- `backend/internal/bigtask/handler.go` — `Update`: field baru `BaselinePct *int` di `updateBigTaskRequest`, logic upsert/delete Daily Task+day_entry baseline (transaksional, dalam `tx`). **`Create`: field `BaselinePct *int` juga di `createBigTaskRequest`** — dipanggil `upsertBaseline` yang sama di dalam transaksi create, super_user only (403 kalau non-super_user isi field ini saat create), validasi 0-100.
- `backend/internal/dailytask/handler.go` — `DailyTask` struct dapat field `IsBaseline bool`.
- `frontend/src/lib/types.ts` — `DailyTask.is_baseline`.
- `frontend/src/lib/components/BigTaskList.svelte` — form edit Big Task dapat input "Persentase awal (opsional)"; **form create JUGA dapat field yang sama (super_user only)**; badge/label "Baseline" di daftar Daily Task kalau `is_baseline`.
- `docs/05-api-contract.md` §4 (`PATCH /big-tasks/{id}` **dan `POST /boards/{id}/big-tasks`**), `docs/06-db-design.md` §3.7 — update kontrak & skema.
