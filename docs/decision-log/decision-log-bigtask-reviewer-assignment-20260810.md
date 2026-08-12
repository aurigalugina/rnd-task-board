# Decision Log — Reviewer Assignment per Big Task (Menggantikan Review Queue Scope Lama)

**Tanggal:** 2026-08-10
**Konteks:** bigtask-reviewer-assignment

## Konteks/Masalah

Review Queue (Fase 7) di-hardcode cuma role `spv` (lihat `decision-log-review-queue-scope-20260809.md`) — SEMUA daily task yang belum ditinjau tampil ke SEMUA user ber-role spv, tanpa cara pointing "yang review task ini spesifik siapa". User menunjukkan ini gak cukup: gak semua Big Task murni development yang wajar direview SPV — misal Big Task berupa dokumentasi bisa jadi reviewernya orang lain, bukan SPV. User minta: saat bikin Big Task, bisa pointing SIAPA yang akan mereview-nya, dan bisa lebih dari satu orang.

## Keputusan

**Tipe assignment: orang spesifik (bukan role).** Saat create Big Task, bisa multi-select 0..N user tertentu sebagai reviewer. Disimpan di tabel baru `big_task_reviewers (big_task_id, user_id)`.

**Review Queue scope berubah total** (menggantikan/memperluas keputusan lama di atas):
- Middleware `RequireRole("spv")` di route `/review-queue*` DIHAPUS, diganti `RequireAuth` biasa (semua user login boleh akses, tapi hasilnya di-filter per user).
- `GET /review-queue` sekarang mengembalikan daily task yang belum ditinjau DAN (requesting user ada di `big_task_reviewers` milik big task itu) ATAU (big task itu belum punya reviewer di-assign sama sekali DAN requesting user ber-role `spv` — fallback ini WAJIB supaya Big Task lama/belum di-assign tidak hilang dari radar siapapun).
- `POST /review-queue/{type}/{id}/mark-reviewed` pakai otorisasi yang SAMA (kalau kamu gak berhak LIHAT item itu, kamu juga gak berhak mark-reviewed-nya) — dicek eksplisit di handler (bukan cuma dari filter list), supaya endpoint mark-reviewed tidak bisa dipakai langsung oleh user yang tidak berwenang meskipun tahu ID-nya.
- `item_type` yang didukung tetap cuma `daily_task` (keputusan lama itu masih berlaku, tidak berubah).

**Tidak menggantikan clone-review (Fase 5).** Tombol "Review SPV"/"Review QA" di `DailyTaskPanel` itu fitur BEDA (bikin Daily Task baru sebagai unit kerja review) — reviewer assignment ini murni metadata "siapa yang harus tinjau Daily Task-daily task di bawah Big Task ini lewat Review Queue", dua-duanya independen dan tetap dipakai bareng.

## Alasan

- **Orang spesifik dipilih ketimbang role**: lebih presisi sesuai kebutuhan nyata user (mis. dokumentasi direview orang tertentu, bukan "siapapun yang kebetulan role X") — role based dianggap kurang fleksibel untuk kasus campuran yang disebutkan.
- **Fallback ke SPV kalau belum di-assign**: tanpa ini, Big Task LAMA (termasuk semua data seed yang sudah dibuat sebelum fitur ini ada) otomatis hilang dari Review Queue siapapun begitu middleware SPV-only dicabut — regresi diam-diam. Fallback bikin behavior lama (SPV lihat semua yang belum di-assign) tetap jalan sampai reviewer di-assign eksplisit.
- **Cek ulang otorisasi di `mark-reviewed`, bukan cuma di `List`**: kalau cuma difilter di List, endpoint mark-reviewed tetap bisa dipanggil langsung (mis. lewat curl/devtools) oleh user yang harusnya tidak berwenang atas item itu — nge-trust filter di list-endpoint doang buat keamanan action-endpoint yang terpisah itu bug klasik.
- **Big Task reviewer_user_ids ditampilkan di response `GET /boards/{id}/big-tasks` (bukan endpoint terpisah)**: konsisten sama pola `default_pic_user_id` yang sudah ada (ID mentah, nama di-resolve di frontend dari `GET /users/assignable` yang udah di-fetch), bukan bikin bentuk response baru.

## Dampak/File Terpengaruh

- `backend/db/migrations/0009_create_big_task_reviewers.up.sql`/`.down.sql` (baru).
- `backend/internal/bigtask/handler.go` — `BigTask.ReviewerUserIDs`, `ListByBoard`/`loadBigTask` query nambah agregasi reviewer, `Create` terima `reviewer_user_ids` dan insert ke tabel baru.
- `backend/internal/reviewqueue/handler.go` — `List` filter per user (bukan lagi query polos), `MarkReviewed` tambah cek otorisasi eksplisit.
- `backend/cmd/api/main.go` — route `/review-queue*` pindah dari grup `spvOnly` ke grup `protected` biasa.
- `frontend/src/lib/types.ts` — `BigTask.reviewer_user_ids: string[]`.
- `frontend/src/lib/components/BigTaskList.svelte` — form create Big Task nambah multi-select reviewer (checkbox list dari `assignableUsers`, konsisten sumber data sama seperti PIC picker); tampilkan avatar reviewer di card/header.
- `frontend/src/routes/+layout.svelte` — badge notifikasi & tab Review Queue gak lagi di-gate `isSpv`, cukup `pendingReview > 0` (backend yang nentuin isi queue-nya per user).
- `docs/05-api-contract.md` §3/§4/§9 — dokumentasi field & endpoint baru.
- `docs/06-db-design.md` — tabel `big_task_reviewers` baru.
- CLAUDE.md — ringkasan rule ini disinkronkan, dan referensi ke `decision-log-review-queue-scope-20260809.md` diberi catatan "digantikan sebagian oleh keputusan ini".
