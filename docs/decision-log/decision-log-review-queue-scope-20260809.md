# Decision Log — Scope Review Queue

**Tanggal:** 2026-08-09
**Konteks:** review-queue-scope

## Konteks/Masalah

`05-api-contract.md` §9 bilang `GET /review-queue` "scope ditentukan role — SPV melihat lintas tim" tapi tidak merinci role lain lihat apa. BR-11.1 (`02-brd.md`) secara eksplisit cuma bilang "indikator terpusat bagi peran **SPV**" — tidak menyebut kebutuhan serupa buat role lain. FR-NTF-03 bilang endpoint "mendaftar seluruh item **belum ditinjau**" tapi contoh response menyertakan field `reviewed` yang jadi ambigu (kalau endpoint cuma pernah balikin item belum-review, field itu selalu `false`).

Juga: `item_reviews` (tabel yang didesain di Fase 0, `docs/decision-log/decision-log-review-queue-schema-20260809.md`) sifatnya polymorphic (`item_type`) tapi belum diputuskan tipe item apa saja yang masuk di iterasi ini.

## Keputusan

1. `GET /review-queue` dan `POST /review-queue/{item_type}/{item_id}/mark-reviewed` dibatasi role **`spv` saja** (`RequireRole("spv")`) — BUKAN "semua role dengan scope beda-beda". BRD cuma menyebut kebutuhan ini untuk SPV; role lain tidak punya use case yang didokumentasikan.
2. Item yang masuk queue di iterasi ini cuma `item_type = "daily_task"` — SEMUA Daily Task lintas semua board (bukan per-board), difilter yang belum punya baris di `item_reviews`.
3. Endpoint balikin item yang **belum** direview saja (bukan seluruh item + flag `reviewed`) — sesuai teks FR-NTF-03 secara literal. Field `reviewed` di response tetap ada di bentuk objeknya (selalu `false`) buat jaga bentuk sesuai kontrak, tapi nilainya redundant di iterasi ini.

## Alasan

- **SPV-only, bukan role-aware scope untuk semua orang**: menghindari membangun logic scope yang tidak diminta BRD (yang cuma jelas-jelas menyebut kebutuhan SPV) — kalau nanti benar-benar dibutuhkan role lain punya queue sendiri (mis. review sesama dev), itu perubahan produk yang perlu didiskusikan ulang, bukan diasumsikan sekarang.
- **Cuma `daily_task`** karena itu satu-satunya entitas yang FR-NTF sebutkan sebagai contoh, dan satu-satunya yang punya konsep "progress kerja yang perlu dilihat SPV" yang jelas di data model saat ini. Comments/cheat-sheet belum punya kebutuhan "review" yang terdokumentasi.
- **Cuma item belum-review yang dikembalikan** konsisten dengan teks requirement FR-NTF-03 apa adanya — tidak menambah kompleksitas filtering di frontend untuk kebutuhan yang belum ada (mis. "lihat riwayat yang sudah direview").

## Dampak/File terpengaruh

- `backend/db/migrations/0008_create_item_reviews.up.sql`/`.down.sql` (baru, sesuai desain Fase 0).
- `backend/internal/reviewqueue/handler.go` (modul baru).
- `backend/cmd/api/main.go` — route di grup `RequireRole("spv")`.
- `frontend/src/routes/review-queue/+page.svelte` dan notif dropdown di `+layout.svelte` (FR-NTF-01, badge jumlah item) — cuma tampil kalau role `spv`.
