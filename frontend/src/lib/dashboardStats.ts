// Agregasi Dashboard lintas board (routes/+page.svelte) — diekstrak jadi
// fungsi murni supaya bisa ditest tanpa render komponen. Field turunan
// (actual_pct, expected_pct, verdict) di BigTask itu sendiri SELALU dari
// server (lihat types.ts) — di sini kita agregasi DUA tingkat: Big Task ->
// Board (satu board = satu project, lihat aggregateBoards) lalu Board ->
// statistik portfolio (computeDashboardStats). POV Dashboard adalah per
// BOARD, bukan per Big Task — lihat
// docs/decision-log/decision-log-dashboard-board-level-aggregation-20260810.md
// untuk alasan lengkap rule status/verdict board di bawah.
import type { BigTask } from './types';

export type BigTaskRow = { bt: BigTask; boardName: string };

export type BoardStatus = 'not_started' | 'running' | 'done' | 'hold';
export type BoardVerdict = 'won' | 'lose' | 'neutral';

export type BoardAgg = {
  boardId: string;
  boardName: string;
  totalBigTasks: number;
  avgActualPct: number;
  avgExpectedPct: number;
  daysLeft: number;
  status: BoardStatus;
  verdict: BoardVerdict;
};

export type BoardWithTasks = { boardId: string; boardName: string; bigTasks: BigTask[] };

// Rule status: all-or-nothing per bucket, default "running" kalau campuran
// (project dianggap "in progress" selama belum seragam ke salah satu state).
// Rule verdict: "lose" kalau ADA MINIMAL SATU Big Task lose (sinyal negatif
// tidak boleh ketutupan rata-rata), "won" hanya kalau status "done" DAN tidak
// ada lose sama sekali — asimetris dgn status, ikut filosofi netral-sampai-
// keputusan yang sudah dipakai utk verdict Big Task individual.
export function aggregateBoards(groups: BoardWithTasks[]): BoardAgg[] {
  return groups.map(({ boardId, boardName, bigTasks }) => {
    const total = bigTasks.length;
    const notStarted = bigTasks.filter((bt) => bt.actual_pct === 0 && !bt.on_hold).length;
    const done = bigTasks.filter((bt) => bt.signed).length;
    const hold = bigTasks.filter((bt) => bt.on_hold).length;
    const hasLose = bigTasks.some((bt) => bt.verdict === 'lose');

    let status: BoardStatus;
    if (total === 0) status = 'not_started';
    else if (done === total) status = 'done';
    else if (hold === total) status = 'hold';
    else if (notStarted === total) status = 'not_started';
    else status = 'running';

    const verdict: BoardVerdict = hasLose ? 'lose' : status === 'done' ? 'won' : 'neutral';

    const avgActualPct = total ? Math.round(bigTasks.reduce((s, bt) => s + bt.actual_pct, 0) / total) : 0;
    const avgExpectedPct = total ? Math.round(bigTasks.reduce((s, bt) => s + bt.expected_pct, 0) / total) : 0;
    const unresolved = bigTasks.filter((bt) => !bt.signed);
    const daysLeftSource = unresolved.length ? unresolved : bigTasks;
    const daysLeft = daysLeftSource.length ? Math.min(...daysLeftSource.map((bt) => bt.days_left)) : 0;

    return { boardId, boardName, totalBigTasks: total, avgActualPct, avgExpectedPct, daysLeft, status, verdict };
  });
}

export type DashboardStats = {
  total: number;
  notStarted: number;
  running: number;
  done: number;
  hold: number;
  won: number;
  lose: number;
  completionRate: number;
  activeBoards: BoardAgg[];
  loseBoards: BoardAgg[];
  nearestDeadline: BoardAgg[];
};

// activeBoards = SEMUA board KECUALI yang sudah "done" (selesai, semua Big
// Task sign-off) -- dipakai bareng oleh progress chart, attention-list, DAN
// nearestDeadline (satu aturan konsisten, bukan beda-beda per section).
// Board "done" sudah cukup terwakili di stat card/donut summary, gak perlu
// dipantau progress-nya lagi di sini. Lihat
// docs/decision-log/decision-log-dashboard-progress-scope-20260810.md.
export function computeDashboardStats(boards: BoardAgg[]): DashboardStats {
  const total = boards.length;
  const notStarted = boards.filter((b) => b.status === 'not_started').length;
  const running = boards.filter((b) => b.status === 'running').length;
  const done = boards.filter((b) => b.status === 'done').length;
  const hold = boards.filter((b) => b.status === 'hold').length;
  const won = boards.filter((b) => b.verdict === 'won').length;
  const lose = boards.filter((b) => b.verdict === 'lose').length;
  const completionRate = total ? Math.round((done / total) * 100) : 0;

  const activeBoards = boards.filter((b) => b.status !== 'done');
  const loseBoards = boards.filter((b) => b.verdict === 'lose');
  const nearestDeadline = [...activeBoards].sort((a, b) => a.daysLeft - b.daysLeft).slice(0, 5);

  return { total, notStarted, running, done, hold, won, lose, completionRate, activeBoards, loseBoards, nearestDeadline };
}

// Dipakai buat label sumbu-x GroupedBarChart — nama panjang dipotong biar
// chart tidak berantakan.
export function truncateName(name: string, maxLen = 12): string {
  return name.length > maxLen ? name.slice(0, maxLen - 1) + '…' : name;
}
