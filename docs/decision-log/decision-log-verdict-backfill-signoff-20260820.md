# Decision Log — Verdict Backfill & Sign-off Historis

**Tanggal:** 2026-08-20
**Konteks:** verdict-backfill-signoff
**Status:** 🟢 FINAL (menunggu kata "gas" dari user buat mulai implementasi — desain sudah disepakati, belum ada baris kode yang diubah).

## Konteks/Masalah

User import data historical (board/big task/daily task/day entries) via fitur Import (`dataport`). Big Task hasil import punya `actual_pct` 100% tapi verdict-nya "Lose" — dicoba di-sign-off manual pun TETAP "Lose" (screenshot: deadline `2026-07-28`, sign-off diklik `2026-08-20`, badge "Lose" + "Signed done" muncul BARENGAN, label "telat 23 hari").

### Root cause yang sudah dikonfirmasi (baca kode langsung, bukan dugaan)

**Lapis 1 — bug umum, ngaruh ke SEMUA Big Task (bukan cuma yang diimpor):**
`backend/internal/bigtask/handler.go: computeVerdict(deadline, signed, now)` — kalau `signed=true`, tetap bandingin `deadline` vs `now` (waktu REQUEST/baca, `time.Now()` real-time), BUKAN vs `signed_at` (waktu sign-off beneran terjadi):
```go
if signed {
    if daysLeft >= 0 { return "win" }   // daysLeft = deadline - NOW, bukan deadline - signed_at
    return "lose"
}
```
Konsekuensi: Big Task yang sign-off-nya SAH dan ON TIME (mis. deadline 15 Agustus, sign-off 14 Agustus) akan tetap "Win" SELAMA dilihat sebelum 15 Agustus, tapi begitu kalender lewat 15 Agustus dan dashboard dibuka lagi, verdict-nya BERUBAH jadi "Lose" — padahal keputusan menang itu sudah sah & final di masa lalu. Ini bug serius: project lama yang sudah "Menang" bakal pelan-pelan "berubah" jadi Lose seiring waktu berjalan, murni karena dashboard dibuka lagi setelah deadline lewat kalender (independen dari isu import).

Ditest eksplisit di `bigtask/handler_test.go` (`TestComputeVerdict`, kasus `"signed, deadline already passed"` → expect `"lose"`) — jadi perilaku ini SENGAJA per interpretasi BRD RULE-05 saat ini ("win sah kalau sign-off terjadi sebelum/tepat deadline"), tapi perbandingannya salah pakai `now` bukan `signed_at`.

**Lapis 2 — gap khusus data backfill/historis:**
`SignOff` handler (`backend/internal/bigtask/handler.go` baris ~376-380) selalu pakai `now()` buat `signed_at`, TIDAK ADA parameter buat override:
```go
INSERT INTO big_task_signoffs (id, big_task_id, signed_by)
VALUES ($1, $2, $3)
ON CONFLICT (big_task_id) DO UPDATE SET signed_by = $3, signed_at = now()
```
Konsekuensi: bahkan KALAU Lapis 1 dibenerin (pakai `signed_at` bukan `now`), kasus user TETAP "Lose" — karena `signed_at` bakal kecatat "hari ini" (2026-08-20), padahal deadline historis (2026-07-28) sudah lewat 23 hari SEBELUM sign-off itu terjadi. `signed_at` gak bisa dibackdate ke tanggal yang mencerminkan kapan pekerjaan itu BENERAN selesai di dunia nyata.

## Keputusan Final

### Lapis 1 — Fix `computeVerdict` pakai `signed_at`, bukan `now()` (wajib, dasar buat A & B)

Signature berubah dari `computeVerdict(deadline, signed bool, now)` jadi `computeVerdict(deadline, signedAt *time.Time, now)`:

```go
func computeVerdict(deadline time.Time, signedAt *time.Time, now time.Time) (verdict string, daysLeft int) {
    if signedAt != nil {
        daysLeft = int(deadline.Sub(*signedAt).Hours() / 24)
        if daysLeft >= 0 {
            return "win", daysLeft
        }
        return "lose", daysLeft
    }
    daysLeft = int(deadline.Sub(now).Hours() / 24)
    if daysLeft < 0 {
        return "lose", daysLeft
    }
    return "on_progress", daysLeft
}
```

**Efek yang KELIHATAN di UI (user sudah di-informasikan & setuju):** `days_left` SELALU ditampilkan di sidebar Big Task (`BigTaskList.svelte`, badge "Xh lagi"/"telat Xh"), gak peduli verdict apa. Begitu fix ini jalan, buat Big Task yang SUDAH sign-off, angka itu makna-nya berubah dari "deadline vs hari ini" jadi "deadline vs tanggal sign-off" — lebih bermakna buat item yang statusnya udah final (keputusan menang/kalah gak lagi "bergerak" seiring waktu berjalan, sesuai tujuan awal BRD RULE-05).

### Opsi A — `signed_at` custom di endpoint sign-off (super_user only)

`POST /big-tasks/{id}/sign-off` terima body opsional `{ "signed_at": "2026-07-28" }` — **tanggal doang** (bukan datetime), konsisten sama `start_date`/`deadline` yang emang kolom `DATE`.

- Field ini CUMA dihormati kalau requester `super_user`. Non-super_user kirim field ini → 403 (fail-loud, pola sama `as_user_id` di Weekly Plan).
- Validasi range (400 kalau melanggar):
  1. `signed_at` gak boleh di masa depan (> hari ini beneran).
  2. `signed_at` gak boleh sebelum `start_date` Big Task.
  3. `signed_at` gak boleh sebelum `MAX(entry_date)` di antara semua day_entries Big Task itu.
- Audit trail: kolom baru `big_task_signoffs.signed_at_backdated_by UUID NULL REFERENCES users(id)` — NULL kalau sign-off normal (signed_at = hari ini, gak di-override), keisi user_id super_user itu kalau `signed_at` di-backdate manual. `signed_by` TETAP makna sama seperti sekarang (siapa yang klik/actor sign-off, gak berubah).

### Opsi B — `PATCH /big-tasks/{id}` buat edit judul + tanggal (super_user only)

Body `{ name?, start_date?, deadline? }` (partial update, semua optional). `on_hold` TETAP lewat endpoint terpisah yang sudah ada (`PATCH /big-tasks/{id}/on-hold`), gak masuk sini.

- Super_user only (in-handler check, bukan `RequireRole` — konsisten pola `access_level`).
- Audit trail: kolom baru `big_tasks.updated_by UUID NULL REFERENCES users(id)` — diisi user_id super_user yang terakhir edit (`updated_at` sudah ada dari awal, tinggal ditambah siapanya).

### Poin C — Fitur Import TIDAK disentuh

Proses migrasi data historical user sudah selesai (one-time backfill). Format Excel Import TIDAK perlu diubah untuk bawa field sign-off — remediasi verdict buat data lama cukup manual lewat Opsi A/B di atas. Kalau nanti ada kebutuhan impor massal berulang yang butuh backfill sign-off otomatis, itu didiskusikan ulang sebagai topik terpisah (bukan bagian keputusan ini).

## Dampak/File Terpengaruh

- `backend/internal/bigtask/handler.go` — `computeVerdict` (Lapis 1, signature berubah + kedua caller-nya ikut diupdate), `SignOff` (Opsi A: terima+validasi `signed_at`, isi `signed_at_backdated_by`), handler baru `Update` (Opsi B: `PATCH /big-tasks/{id}`, isi `updated_by`).
- Migration baru: `big_task_signoffs.signed_at_backdated_by` (nullable FK users) + `big_tasks.updated_by` (nullable FK users).
- `backend/cmd/api/main.go` — route baru `PATCH /big-tasks/{bigTaskID}` (Opsi B).
- `bigtask/handler_test.go` — `TestComputeVerdict` disesuaikan (signature baru, tambah kasus signed dengan `signed_at` sebelum/sesudah deadline).
- Frontend: `BigTaskList.svelte` — tombol sign-off dapat input tanggal opsional (cuma muncul kalau `access_level=super_user`); kemungkinan tombol/form edit Big Task baru (Opsi B, super_user only).
- `frontend/src/lib/types.ts` — field baru di `BigTask` type kalau perlu ditampilkan (`signed_at_backdated_by`, dst — TBD saat implementasi).
- `docs/06-db-design.md`/`05-api-contract.md` — update skema `big_task_signoffs`/`big_tasks` + dokumentasi endpoint baru.
