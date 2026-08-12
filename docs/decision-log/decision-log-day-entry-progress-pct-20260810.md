# Decision Log — Day Entry: `is_done` (boolean) Diganti `progress_pct` (0-100)

**Tanggal:** 2026-08-10
**Konteks:** day-entry-progress-pct

## Konteks/Masalah

Status Day Entry selama ini binary: `is_done` (`Selesai`/`Belum`). User minta 3 flag: **Belum** (0%), **On Progress** (1-99%, granular), **Selesai** (otomatis 100%) — supaya progress Big Task lebih faktual: hari yang dikerjakan 90% tapi belum tuntas semestinya kontribusi 90% ke `actual_pct`, bukan dianggap 0% cuma karena belum "Selesai" penuh.

## Keputusan

- Kolom `day_entries.is_done boolean` **DIGANTI** `progress_pct SMALLINT` (0-100, `CHECK (progress_pct BETWEEN 0 AND 100)`, default 0) — bukan ditambah berdampingan (hindari dua sumber kebenaran buat satu konsep yang sama, konsisten sama prinsip computed-field project ini).
- 3 state di UI diturunkan dari `progress_pct`, TIDAK disimpan sebagai kolom terpisah: `progress_pct === 0` → **Belum**, `progress_pct === 100` → **Selesai**, selain itu → **On Progress**.
- `actual_pct` di SEMUA level (Daily Task, Big Task, Weekly Plan rollup) berubah dari `100 * count(is_done) / count(*)` (persentase hari yang FULL selesai) jadi `AVG(progress_pct)` (rata-rata progress harian) — generalisasi murni: kalau semua `progress_pct` cuma pernah 0/100 (kayak `is_done` lama), hasilnya identik; begitu ada nilai 1-99, hasilnya jadi lebih akurat/faktual sesuai yang diminta user.
- **Tidak ada migrasi data lossy** — baris existing `is_done=true` → `progress_pct=100`, `is_done=false` → `progress_pct=0` (migration `UPDATE ... SET progress_pct = CASE WHEN is_done THEN 100 ELSE 0 END` sebelum drop kolom lama).
- **UI**: dropdown 3 pilihan (Belum/On Progress/Selesai) + input angka (1-99) yang MUNCUL CUMA kalau state = On Progress. Pilih "Belum"/"Selesai" langsung set 0/100 tanpa perlu isi angka. Komponen baru `DayProgressStatus.svelte`, gantiin `Toggle.svelte` yang barusan dibuat sesi sebelumnya (`Toggle.svelte` TETAP DIPERTAHANKAN sebagai komponen reusable buat boolean state LAIN yang genuinely dua-nilai — cuma gak dipakai lagi di Day Entry).

## Alasan

- **Ganti kolom, bukan tambah berdampingan**: kalau `is_done` dan `progress_pct` dua-duanya disimpan, bisa ke-desync (mis. `is_done=true` tapi `progress_pct=60`) — tidak ada aturan jelas mana yang "benar". `progress_pct` sendirian sudah cukup nentuin ketiga state (turunan murni), sesuai prinsip computed-field yang sudah dipegang project ini di banyak tempat lain (`verdict`, `expected_pct`, dst).
- **`AVG(progress_pct)` menggantikan `COUNT(is_done)/COUNT(*)` di SEMUA level sekaligus** (bukan cuma Daily Task) — supaya granularitas baru ini KETARIK sampai ke Big Task/Dashboard/Weekly Plan, bukan cuma keliatan di level Day Entry doang (kalau cuma diubah di satu level, faktual di bawah tapi tetap ke-summary jadi 0/100 di atasnya — sia-sia).
- **Dropdown + input kondisional, bukan slider**: input angka lebih presisi buat isi manual (PIC biasanya tahu pasti "kemarin saya kelarin 70%", bukan geser-geser slider buat dapat angka yang tepat), dan dropdown 3 pilihan tegas sesuai istilah yang diminta user (Belum/On Progress/Selesai) — gak butuh mikir "geser sampai mana baru dianggap Selesai".

## Dampak/File Terpengaruh

- `backend/db/migrations/0011_day_entries_progress_pct.up.sql`/`.down.sql` (baru).
- `backend/internal/dailytask/handler.go` — `DayEntry.ProgressPct`, `loadDays` (rata-rata bukan count), `updateDayEntryRequest.ProgressPct` (+validasi 0-100), `AddDayEntry` RETURNING.
- `backend/internal/bigtask/handler.go` — 2x subquery (`ListByBoard`, `loadBigTask`) + `SignOff` validation query, semua `100.0*COUNT FILTER(is_done)/COUNT` → `AVG(progress_pct)`.
- `backend/internal/board/handler.go` — `Summary` subquery, sama.
- `backend/internal/weeklyplan/handler.go` — `List` dan `Push`, sama.
- `frontend/src/lib/types.ts` — `DayEntry.is_done` → `progress_pct: number`.
- `frontend/src/lib/components/DayProgressStatus.svelte` (baru), dipasang di `DailyTaskPanel.svelte` gantiin `Toggle`.
- `docs/05-api-contract.md` §5, `docs/06-db-design.md` §3.8/§5.1/§5.2 — update kontrak & formula.
