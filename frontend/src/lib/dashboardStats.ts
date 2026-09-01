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
  // Start = MIN(start_date) semua Big Task; Due = deadline Big Task yang JADI
  // SUMBER daysLeft di atas (konsisten, bukan MAX terpisah) -- lihat
  // decision-log-boards-dashboard-enhancements-20260820.md. '' kalau board
  // belum punya Big Task sama sekali.
  startDate: string;
  dueDate: string;
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
    let daysLeft = 0;
    let dueDate = '';
    if (daysLeftSource.length) {
      const nearest = daysLeftSource.reduce((min, bt) => (bt.days_left < min.days_left ? bt : min));
      daysLeft = nearest.days_left;
      dueDate = nearest.deadline;
    }
    const startDate = total ? bigTasks.reduce((min, bt) => (bt.start_date < min ? bt.start_date : min), bigTasks[0].start_date) : '';

    return { boardId, boardName, totalBigTasks: total, avgActualPct, avgExpectedPct, daysLeft, status, verdict, startDate, dueDate };
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

// activeBoards = SEMUA board yang di-return backend (backend sudah otomatis
// mengecualikan board yang di-archive manual lewat GET /boards/archive --
// lihat decision-log-board-archive-20260812.md). Board berstatus "done"
// SENGAJA TETAP ditampilkan di sini (progress chart, attention-list,
// nearestDeadline) -- sebelumnya disembunyikan begitu semua Big Task
// sign-off, tapi itu bikin project yang baru selesai "menghilang" dari
// dashboard padahal belum di-archive manual. Sekarang mekanisme "hilang
// dari dashboard" MURNI lewat archive manual (aksi eksplisit user), bukan
// otomatis dari status. Lihat decision-log-dashboard-progress-scope-20260901.md
// (supersedes decision-log-dashboard-progress-scope-20260810.md).
export function computeDashboardStats(boards: BoardAgg[]): DashboardStats {
  const total = boards.length;
  const notStarted = boards.filter((b) => b.status === 'not_started').length;
  const running = boards.filter((b) => b.status === 'running').length;
  const done = boards.filter((b) => b.status === 'done').length;
  const hold = boards.filter((b) => b.status === 'hold').length;
  const won = boards.filter((b) => b.verdict === 'won').length;
  const lose = boards.filter((b) => b.verdict === 'lose').length;
  const completionRate = total ? Math.round((done / total) * 100) : 0;

  const activeBoards = boards;
  const loseBoards = boards.filter((b) => b.verdict === 'lose');
  const nearestDeadline = [...activeBoards].sort((a, b) => a.daysLeft - b.daysLeft).slice(0, 5);

  return { total, notStarted, running, done, hold, won, lose, completionRate, activeBoards, loseBoards, nearestDeadline };
}

// Dipakai buat label sumbu-x GroupedBarChart — nama panjang dipotong biar
// chart tidak berantakan.
export function truncateName(name: string, maxLen = 12): string {
  return name.length > maxLen ? name.slice(0, maxLen - 1) + '…' : name;
}

// Palet kategorikal buat IDENTITAS board di chart (dot/legend), BUKAN warna
// bar expected/actual (itu tetap abu-abu/biru, maknanya jangan berubah) --
// gantiin teks nama panjang yang saling tabrakan di sumbu-X (dikeluhkan
// user), pola sama seperti DonutChart (warna + legend terpisah). Palet dari
// skill dataviz (8 hue, tervalidasi CVD-safe utk light & dark). Lihat
// decision-log-boards-dashboard-enhancements-20260820.md.
const BOARD_CHART_COLORS_LIGHT = [
  '#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#4a3aa7', '#e34948'
];
const BOARD_CHART_COLORS_DARK = [
  '#3987e5', '#d95926', '#199e70', '#c98500', '#d55181', '#008300', '#9085e9', '#e66767'
];

// Hash deterministik (bukan index posisi array) -- board yang sama SELALU
// dapat warna sama walau daftar board di-filter/berubah urutan (identity
// harus stabil, bukan ngikut rank). >8 board bisa share warna (batas palet
// tervalidasi) -- legend tetap disambiguasi lewat nama, bukan warna doang.
export function boardColor(boardId: string, dark = false): string {
  const palette = dark ? BOARD_CHART_COLORS_DARK : BOARD_CHART_COLORS_LIGHT;
  let hash = 0;
  for (let i = 0; i < boardId.length; i++) {
    hash = (hash * 31 + boardId.charCodeAt(i)) | 0;
  }
  return palette[Math.abs(hash) % palette.length];
}
