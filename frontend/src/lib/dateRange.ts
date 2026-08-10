// Mencerminkan generate Day Entry di backend (dailytask.Create, SRS FR-DLY-01/02):
// satu tanggal kalender per hari dalam rentang [start, end] inklusif. Dipakai
// buat preview jumlah/daftar Day Entry di form Daily Task sebelum submit.
export function dateRangeInclusive(start: string, end: string): string[] {
  const startDate = new Date(`${start}T00:00:00Z`);
  const endDate = new Date(`${end}T00:00:00Z`);
  if (Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime())) return [];
  if (endDate < startDate) return [];

  const dates: string[] = [];
  for (let d = startDate; d <= endDate; d.setUTCDate(d.getUTCDate() + 1)) {
    dates.push(d.toISOString().slice(0, 10));
  }
  return dates;
}

export function isWeekend(dateStr: string): boolean {
  const day = new Date(`${dateStr}T00:00:00Z`).getUTCDay();
  return day === 0 || day === 6;
}

// getWeekStart/shiftWeek/weekEnd dipakai My Weekly Plan (Fase 6) — "Senin
// sebagai awal minggu" (06-db-design.md §3.11, komentar kolom week_start).
export function getWeekStart(dateStr: string): string {
  const d = new Date(`${dateStr}T00:00:00Z`);
  const day = d.getUTCDay(); // 0=Minggu, 1=Senin, ..., 6=Sabtu
  const diffToMonday = day === 0 ? -6 : 1 - day;
  d.setUTCDate(d.getUTCDate() + diffToMonday);
  return d.toISOString().slice(0, 10);
}

export function shiftWeek(weekStartStr: string, deltaWeeks: number): string {
  const d = new Date(`${weekStartStr}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + deltaWeeks * 7);
  return d.toISOString().slice(0, 10);
}

export function weekEnd(weekStartStr: string): string {
  const d = new Date(`${weekStartStr}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + 6);
  return d.toISOString().slice(0, 10);
}
