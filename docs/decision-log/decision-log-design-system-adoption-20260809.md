# Decision Log — Adopsi Design System dari Mockup

**Tanggal:** 2026-08-09
**Konteks:** design-system-adoption

## Konteks/Masalah

Implementasi Fase 0–2 pakai styling polos (plain CSS, tanpa design system) karena `docs/rnd-ops-mockup_3.jsx` belum dibaca/direview waktu itu. User menunjukkan mockup ini sebagai acuan visual yang seharusnya diikuti sebelum lanjut ke fase berikutnya. Mockup berupa satu file React (2498 baris) berisi sistem desain "retro Windows 9x/XP chrome" lengkap dengan 4 varian tema (`retro-light`, `retro-dark`, `modern-light`, `modern-dark` — persis match NFR-08), title bar, menu bar/tab ala window, tombol border outset/inset, progress bar bergaris, badge, charts (pie/bar via Recharts), dan seluruh halaman (Dashboard, Boards/ProjectView, Weekly Plan, Review Queue, Task Detail panel, Profile & Settings modal, Comment section, Cheat Sheet).

User sudah konfirmasi 3 keputusan lewat AskUserQuestion:
1. Mockup ini **final**, jadi source of truth.
2. Port **lengkap termasuk charts**.
3. Bangun **design system dulu**, baru diterapkan bertahap ke semua halaman existing.

Selama proses port, ketemu 3 keputusan implementasi turunan yang perlu diputuskan sendiri (bukan ditanyakan ulang ke user — murni "bagaimana", bukan "apa"):

## Keputusan

1. **Chart custom SVG ringan, bukan library chart pihak ketiga.** Mockup pakai Recharts (React-only, tidak ada versi Svelte resmi). Chart yang dibutuhkan cuma 2 jenis (donut/pie 2–4 slice, grouped bar chart sederhana) — dibuat sebagai komponen Svelte custom pakai SVG polos, bukan menambah dependency chart library baru.
2. **Dashboard cross-board dihitung client-side dari endpoint yang sudah ada**, bukan endpoint backend baru. Dashboard di mockup menampilkan agregat LINTAS SEMUA board (bukan per-board seperti `GET /boards/{id}/summary` yang sudah ada) — frontend akan fetch semua board + summary + big-tasks masing-masing lalu agregasi di client. Skala kecil (NFR-03: puluhan board) bikin ini tidak masalah performa.
3. **Settings & Profile jadi modal overlay** (menggantikan halaman `/settings` dan `/users` penuh yang sudah dibuat di Fase 2), sesuai pola asli mockup (dropdown avatar di topbar → "My Profile" / "Settings" sebagai modal, bukan route terpisah). Route `/settings` dan `/users` yang sudah ada di Fase 2 akan dihapus, isinya dipindah jadi komponen modal yang dipanggil dari topbar baru.

## Alasan

- **Custom SVG chart** menghindari dependency baru yang berat untuk kebutuhan visual yang sebenarnya sederhana (beberapa slice warna + bar berpasangan) — konsisten dengan prinsip "jangan tambah abstraksi/dependency di luar yang dibutuhkan".
- **Agregasi client-side** menghindari nambah endpoint backend baru untuk kebutuhan yang bisa dipenuhi dari endpoint per-board yang sudah ada dan sudah dites — data tetap satu sumber (dihitung server per board), cuma dijumlahkan lagi di frontend untuk tampilan portfolio-wide.
- **Modal, bukan halaman**, mengikuti pola interaksi ASLI di mockup (bukan cuma warnanya) — final-source-of-truth berarti pola interaksinya juga diikuti, bukan cuma token visualnya. Halaman `/settings` dan `/users` dari Fase 2 dianggap implementasi sementara sebelum mockup ditemukan.

## Dampak/File terpengaruh

- CSS/desain baru: token 4 tema + komponen dasar (titlebar, topbar/tabs, stat-card, badge, dualbar, table, panel/modal, form) — lapisan pertama sebelum re-skin halaman lain.
- App shell (`+layout.svelte`) dirombak total: titlebar + tab navigasi + dropdown user (Profile/Settings sebagai modal).
- `routes/settings/+page.svelte` dan `routes/users/+page.svelte` (Fase 2) akan dihapus, dipindah jadi `lib/components/ProfileModal.svelte` dan `lib/components/SettingsModal.svelte`.
- Semua halaman lain (Login, Dashboard, Boards/BigTaskList/DailyTaskPanel) di-re-skin mengikuti token & komponen baru.
