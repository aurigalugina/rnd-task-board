// Logic murni buat DayProgressStatus.svelte -- diekstrak supaya testable
// tanpa render komponen. 3 state (Belum/On Progress/Selesai) SELALU
// diturunkan dari progress_pct (0-100), tidak pernah disimpan sebagai field
// terpisah -- lihat docs/decision-log/decision-log-day-entry-progress-pct-20260810.md.

export type DayProgressStatus = 'belum' | 'on_progress' | 'selesai';

export function statusFromProgressPct(pct: number): DayProgressStatus {
  if (pct <= 0) return 'belum';
  if (pct >= 100) return 'selesai';
  return 'on_progress';
}

// Dipanggil saat user ganti dropdown status -- tentuin progress_pct yang
// dikirim ke server. Pindah ke "on_progress" dari belum/selesai default ke
// 50 (titik tengah netral); kalau sebelumnya sudah di rentang itu, pertahankan
// nilainya (mis. user cuma ganti-ganti dropdown tanpa maksud reset angka).
export function progressPctForStatus(status: DayProgressStatus, currentPct: number): number {
  if (status === 'belum') return 0;
  if (status === 'selesai') return 100;
  return currentPct > 0 && currentPct < 100 ? currentPct : 50;
}

// Input manual angka on-progress selalu diklem ke 1-99 (0/100 punya
// state/dropdown sendiri) dan dibulatkan ke integer.
export function clampOnProgressPct(pct: number): number {
  if (Number.isNaN(pct)) return 1;
  return Math.min(99, Math.max(1, Math.round(pct)));
}
