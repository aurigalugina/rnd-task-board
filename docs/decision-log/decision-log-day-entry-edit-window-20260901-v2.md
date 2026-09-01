# Day Entry: Semua Entries Bisa Diedit (Menghapus Lock 3 Hari)

**Date:** 2026-09-01
**Status:** Implemented
**Supersedes:** `decision-log-day-entry-edit-window-20260901.md` (window 3
hari sebelumnya diperluas dari "semua lampau dikunci" menjadi "3 hari bisa
edit" -- keputusan ini menghapus batasan itu sepenuhnya)

## Konteks / Masalah

Sebelumnya, Day Entry (progress harian per Daily Task) hanya bisa diedit
kalau tanggalnya dalam 3 hari ke belakang dari hari ini -- entry yang lebih
lama dari itu jadi read-only (field disabled, tombol hapus/tambah
disabled), ditandai badge "lampau".

User (Lugi) minta batasan ini dihapus total -- semua entry, seberapa pun
lama, harus tetap bisa diedit.

## Keputusan

Hapus seluruh mekanisme lock 3-hari di frontend:
- `DailyTaskPanel.svelte`: fungsi `isDayOlderThan3Days()` dan variabel
  `today` (khusus untuk fungsi itu) dihapus; `openEditModal()` tidak lagi
  menghitung/meneruskan status `past`.
- `DayEntryEditModal.svelte`: prop `isPast` dihapus total. Semua field
  (planned text, status progress, persentase, blocker) dan tombol
  (simpan, hapus) selalu aktif, tidak pernah `disabled`.
- CSS terkait dihapus: `.row-past`, `.past-badge`,
  `.de-modal-badge-past` (semua sudah dead code setelah perubahan ini).
- Test file `editable-days.test.ts` dan `editable-days.spec.ts` (unit test
  murni untuk logic yang sekarang dihapus) dihapus juga.

Yang TIDAK berubah: backend TIDAK pernah punya lock 3-hari untuk update
(`PATCH /day-entries/{id}`) atau delete (`DELETE /day-entries/{id}`) --
pembatasan itu murni ada di frontend. Backend hanya menolak "tanggal
lampau" untuk **create** entry baru (`POST /daily-tasks/{id}/day-entries`,
lihat `dailytask/handler.go` baris ~266/335), yang TIDAK diubah oleh
keputusan ini -- tetap tidak bisa membuat entry baru bertanggal masa lalu,
hanya EDIT entry yang sudah ada (berapa pun lama) yang sekarang selalu
diizinkan.

## Alasan

- Batasan 3-hari adalah kebijakan UX yang sebelumnya dianggap perlu untuk
  mencegah user mengubah data historis terlalu jauh ke belakang, tapi dalam
  praktiknya user butuh koreksi data lampau lebih fleksibel (mis. lupa isi
  progress minggu lalu, salah catat, dsb).
- Backend tidak pernah punya validasi setara -- jadi menghapus batasan di
  frontend tidak membuka celah keamanan/data-integrity baru, cuma
  menghilangkan friction UI yang sebelumnya ada.
- Audit trail tetap ada lewat `updated_by`/timestamp di record (tidak
  hilang, tidak diubah oleh keputusan ini).

## Dampak / File Terpengaruh

- `frontend/src/lib/components/DailyTaskPanel.svelte` -- hapus
  `isDayOlderThan3Days()`, sederhanakan `openEditModal()`, hapus badge
  "lampau" & class `row-past` dari row table.
- `frontend/src/lib/components/DayEntryEditModal.svelte` -- hapus prop
  `isPast` dan semua `disabled={isPast}`/conditional rendering terkait.
- `frontend/src/app.css` -- hapus `.row-past`, `.past-badge`,
  `.de-modal-badge-past`.
- Dihapus: `frontend/src/lib/editable-days.test.ts`,
  `frontend/src/lib/editable-days.spec.ts`.
- Tidak ada perubahan backend (tidak diperlukan -- backend tidak pernah
  menegakkan batasan ini untuk update/delete).

## Verifikasi

- `npm run check` -- 0 errors.
- `npm run test` -- 122/122 passed (turun dari 136 karena 2 file test
  logic-3-hari dihapus, sesuai).
- Build & restart Docker lokal -- OK.
