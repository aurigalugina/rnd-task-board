# Decision Log — Dashboard: Section "Progress" & "Deadline Terdekat" Ikut Semua Board Non-Selesai

**Tanggal:** 2026-08-10
**Konteks:** dashboard-progress-scope

## Konteks/Masalah

User laporkan: board baru yang baru dibuat (belum ada Big Task sama sekali) sudah benar muncul di stat card ("Total project"/"Belum berjalan") dan donut "Status project" — TAPI tidak pernah muncul di section "Progress: actual vs expected (%)" (bar chart + baris `DualBar` di bawahnya) maupun tabel "Deadline terdekat", sampai board itu punya minimal satu Big Task dengan progress (status board jadi `running`). Dikonfirmasi lewat reproduksi langsung: `progressChartData` dan `nearestDeadline` di `lib/dashboardStats.ts` cuma diambil dari `runningBoards` (`status === 'running'`), jadi board `not_started` (baru dibuat), `done` (sudah selesai), dan `hold` (di-pause) semuanya tidak pernah tampil di dua bagian ini — user merasa ini "cuma muncul kalau sudah sign off" karena baru nyadar gap-nya pas ngetes board yang progress-nya masih nol.

## Keputusan

Section "Progress: actual vs expected (%)" (chart + attention-list) dan tabel "Deadline terdekat" SEKARANG diambil dari SEMUA board KECUALI yang statusnya `done` (sudah selesai — semua Big Task sign-off). Jadi `not_started`, `running`, DAN `hold` semua ikut muncul; cuma `done` yang dikeluarkan (board yang sudah selesai dianggap tidak perlu dipantau progress-nya lagi di section ini — sudah cukup terwakili di stat card/donut summary).

Field baru di `DashboardStats`: `activeBoards` (menggantikan `runningBoards` yang sebelumnya cuma `status === 'running'`) — dipakai bareng oleh `progressChartData`, attention-list, DAN `nearestDeadline` (linear, konsisten, bukan aturan beda-beda per section).

## Alasan

- **Board `not_started` tetap relevan ditampilkan** (0% actual vs 0% expected, atau `expected_pct` non-zero kalau ada Big Task yang start_date-nya sudah lewat tapi belum ada progress dicatat) — user secara eksplisit minta board baru "impact ke data-data dashboard" segera setelah dibuat, bukan nunggu ada progress dulu.
- **Board `done` dikeluarkan**: begitu semua Big Task sign-off, board itu sudah "selesai ceritanya" — progress 100% vs target sudah tidak informatif buat dipantau terus, dan tetap representasikan lewat "Sudah selesai" stat card + donut "Status project" (tidak hilang dari Dashboard sama sekali, cuma tidak di section progress-tracking ini).
- **Satu aturan dipakai konsisten di 2 tempat (chart & deadline terdekat)**: user pilih opsi ini secara eksplisit ketimbang punya rule beda-beda per section — lebih predictable, dan `nearestDeadline` secara logis memang perlu tahu board yang belum selesai (termasuk yang belum mulai — deadline-nya tetap "mendatang" meski progress masih nol).

## Dampak/File Terpengaruh

- `frontend/src/lib/dashboardStats.ts` — `computeDashboardStats`: field `runningBoards` diganti `activeBoards` (filter `status !== 'done'`, bukan `status === 'running'`); `nearestDeadline` sort dari `activeBoards`.
- `frontend/src/lib/dashboardStats.test.ts` — test disesuaikan (termasuk kasus board `not_started`/`hold` yang sekarang HARUS muncul di `activeBoards`/`nearestDeadline`, dan board `done` yang HARUS TIDAK muncul).
- `frontend/src/routes/+page.svelte` — referensi `runningBoards` → `activeBoards` di `progressChartData` dan blok attention-list.
