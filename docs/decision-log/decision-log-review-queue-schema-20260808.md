# Decision Log — Skema Tabel Review Queue

**Tanggal:** 2026-08-08
**Konteks:** review-queue-schema

## Konteks/Masalah

`03-srs.md` mendefinisikan FR-NTF-01/02/03 (indikator item belum ditinjau, aksi mark-reviewed, panel review queue), dan `05-api-contract.md` §9 mendokumentasikan endpoint `GET /review-queue` serta `POST /review-queue/{item_type}/{item_id}/mark-reviewed`. Namun `06-db-design.md` **tidak** mendefinisikan tabel mana pun untuk menyimpan status "sudah ditinjau" — gap ini baru ketahuan saat menyusun development roadmap ([[decision-log-development-roadmap-20260808]]), bukan didokumentasikan sejak awal.

Bentuk endpoint di API contract (`{item_type}/{item_id}`) mengindikasikan desain polymorphic dari awal — hanya belum diturunkan ke skema tabel.

## Keputusan

Tambahkan tabel generik `item_reviews` (bukan kolom `reviewed_by`/`reviewed_at` langsung di `daily_tasks`), dengan bentuk:

```sql
CREATE TABLE item_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_type TEXT NOT NULL,       -- mis. 'daily_task' (nilai lain menyusul kalau scope review meluas)
    item_id UUID NOT NULL,
    reviewed_by UUID NOT NULL REFERENCES users(id),
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (item_type, item_id)
);
```

Keberadaan baris (per kombinasi `item_type` + `item_id`) = item tersebut sudah ditinjau — pola yang sama seperti `big_task_signoffs` (keberadaan baris = signed).

Migration belum dibuat sekarang — ini baru keputusan desain untuk Fase 7 ([[decision-log-development-roadmap-20260808]]), akan dieksekusi sebagai `0008_create_item_reviews.up.sql` saat fase itu dikerjakan.

## Alasan

- **Tabel generik dipilih daripada kolom langsung di `daily_tasks`** karena `item_type` di kontrak API sudah mengisyaratkan desain lintas-entitas — kalau scope review meluas (mis. ke `comments` atau `change_requests`) di masa depan, kolom langsung per tabel akan berarti migrasi berulang tiap kali scope bertambah, melanggar prinsip produk #4 di `01-vision-product.md` ("sistem harus bisa tumbuh tanpa dirombak ulang").
- **Pola "keberadaan baris = status"** dipilih agar konsisten dengan pola yang sudah ada di `big_task_signoffs`, bukan menambah pola baru (flag boolean) untuk konsep yang serupa.
- **`UNIQUE (item_type, item_id)`** memastikan `mark-reviewed` bersifat idempoten — mark ulang pada item yang sama tidak menghasilkan duplikat baris, cukup `ON CONFLICT DO UPDATE reviewed_by/reviewed_at`.
- Tidak menambahkan foreign key ke `item_id` (karena polymorphic, tidak bisa FK ke satu tabel spesifik) — validasi bahwa `item_id` benar-benar ada dilakukan di layer aplikasi saat `mark-reviewed` dipanggil, bukan di level database.

## Dampak/File terpengaruh

- `docs/06-db-design.md` — ditambahkan §3.13 (tabel baru) dan disebut di ERD ringkas §2.
- Migration baru `backend/db/migrations/0008_create_item_reviews.*.sql` — dibuat saat Fase 7 dikerjakan, bukan sekarang.
- `backend/internal/reviewqueue/` (modul baru) akan query tabel ini saat diimplementasikan.
