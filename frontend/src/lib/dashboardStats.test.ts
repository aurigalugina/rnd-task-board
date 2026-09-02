import { describe, expect, it } from 'vitest';
import { aggregateBoards, boardColor, computeDashboardStats, truncateName, type BoardWithTasks } from './dashboardStats';
import type { BigTask } from './types';

function fakeBigTask(overrides: Partial<BigTask>): BigTask {
  return {
    id: 'bt-' + Math.random(),
    board_id: 'board-1',
    name: 'Tahap Analisis',
    description: '',
    severity: 'medium',
    start_date: '2026-08-01',
    deadline: '2026-08-20',
    default_pic_user_id: null,
    on_hold: false,
    actual_pct: 0,
    expected_pct: 0,
    days_left: 5,
    verdict: 'on_progress',
    signed: false,
    signed_by: null,
    signed_at: null,
    signed_at_backdated_by: null,
    updated_by: null,
    member_user_ids: [],
    ...overrides
  };
}

function board(boardId: string, boardName: string, bigTasks: Partial<BigTask>[]): BoardWithTasks {
  return { boardId, boardName, bigTasks: bigTasks.map(fakeBigTask) };
}

describe('aggregateBoards', () => {
  it('a board with zero big tasks is not_started', () => {
    const [b] = aggregateBoards([board('b1', 'Empty Board', [])]);
    expect(b.status).toBe('not_started');
    expect(b.totalBigTasks).toBe(0);
    expect(b.avgActualPct).toBe(0);
  });

  it('status is not_started only when ALL big tasks are 0% and not on hold', () => {
    const [b] = aggregateBoards([board('b1', 'B', [{ actual_pct: 0, on_hold: false }, { actual_pct: 0, on_hold: false }])]);
    expect(b.status).toBe('not_started');
  });

  it('status is done only when ALL big tasks are signed', () => {
    const [b] = aggregateBoards([board('b1', 'B', [{ signed: true }, { signed: true }])]);
    expect(b.status).toBe('done');
  });

  it('status is hold only when ALL (unsigned) big tasks are on_hold', () => {
    const [b] = aggregateBoards([board('b1', 'B', [{ on_hold: true }, { on_hold: true }])]);
    expect(b.status).toBe('hold');
  });

  it('mixed status (some done, some not started) defaults to running, not done/not_started', () => {
    const [b] = aggregateBoards([board('b1', 'B', [{ signed: true, actual_pct: 100 }, { actual_pct: 0 }])]);
    expect(b.status).toBe('running');
  });

  it('mixed status (some on_hold, some not started) defaults to running, not hold', () => {
    const [b] = aggregateBoards([board('b1', 'B', [{ on_hold: true }, { actual_pct: 0, on_hold: false }])]);
    expect(b.status).toBe('running');
  });

  it('verdict is lose if ANY big task is lose, even when the board is otherwise fully done', () => {
    const [b] = aggregateBoards([
      board('b1', 'B', [
        { signed: true, verdict: 'win' },
        { signed: true, verdict: 'lose' }
      ])
    ]);
    expect(b.status).toBe('done');
    expect(b.verdict).toBe('lose');
  });

  it('verdict is won only when status is done AND no big task is lose', () => {
    const [b] = aggregateBoards([board('b1', 'B', [{ signed: true, verdict: 'win' }, { signed: true, verdict: 'win' }])]);
    expect(b.verdict).toBe('won');
  });

  it('verdict is neutral when board is still running with no lose', () => {
    const [b] = aggregateBoards([board('b1', 'B', [{ actual_pct: 50 }])]);
    expect(b.verdict).toBe('neutral');
  });

  it('averages actual_pct and expected_pct across all big tasks in the board', () => {
    const [b] = aggregateBoards([
      board('b1', 'B', [
        { actual_pct: 100, expected_pct: 50 },
        { actual_pct: 0, expected_pct: 30 }
      ])
    ]);
    expect(b.avgActualPct).toBe(50);
    expect(b.avgExpectedPct).toBe(40);
  });

  it('daysLeft is the minimum among unsigned big tasks, ignoring signed ones', () => {
    const [b] = aggregateBoards([
      board('b1', 'B', [
        { signed: true, days_left: -10 },
        { signed: false, days_left: 3 },
        { signed: false, days_left: 8 }
      ])
    ]);
    expect(b.daysLeft).toBe(3);
  });

  it('daysLeft falls back to all big tasks when every one is signed', () => {
    const [b] = aggregateBoards([board('b1', 'B', [{ signed: true, days_left: 7 }, { signed: true, days_left: 2 }])]);
    expect(b.daysLeft).toBe(2);
  });

  it('aggregates multiple boards independently', () => {
    const boards = aggregateBoards([
      board('b1', 'Board A', [{ signed: true, verdict: 'win' }]),
      board('b2', 'Board B', [{ actual_pct: 0 }])
    ]);
    expect(boards).toHaveLength(2);
    expect(boards[0].status).toBe('done');
    expect(boards[1].status).toBe('not_started');
  });
});

describe('computeDashboardStats', () => {
  it('returns all zeros for an empty portfolio', () => {
    const stats = computeDashboardStats([]);
    expect(stats.total).toBe(0);
    expect(stats.completionRate).toBe(0);
    expect(stats.activeBoards).toEqual([]);
  });

  it('counts boards into status/verdict buckets', () => {
    const boards = aggregateBoards([
      board('b1', 'Done Board', [{ signed: true, verdict: 'win' }]),
      board('b2', 'Not Started Board', [{ actual_pct: 0 }]),
      board('b3', 'Running Board', [{ actual_pct: 50 }]),
      board('b4', 'Hold Board', [{ on_hold: true }]),
      board('b5', 'Lose Board', [{ actual_pct: 40, verdict: 'lose' }])
    ]);
    const stats = computeDashboardStats(boards);
    expect(stats.total).toBe(5);
    expect(stats.done).toBe(1);
    expect(stats.notStarted).toBe(1);
    expect(stats.running).toBe(2); // Running Board + Lose Board (status defaults to running)
    expect(stats.hold).toBe(1);
    expect(stats.won).toBe(1);
    expect(stats.lose).toBe(1);
  });

  it('rounds completion rate to the nearest integer', () => {
    const boards = aggregateBoards([
      board('b1', 'A', [{ signed: true }]),
      board('b2', 'B', [{ signed: false }]),
      board('b3', 'C', [{ signed: false }])
    ]);
    expect(computeDashboardStats(boards).completionRate).toBe(33);
  });

  it('sorts nearestDeadline by daysLeft ascending and caps at 5, among all non-done boards', () => {
    const boards = aggregateBoards(
      [10, 3, 7, 1, 9, 2, 8].map((daysLeft, i) => board(`b${i}`, `B${i}`, [{ actual_pct: 50, days_left: daysLeft }]))
    );
    const stats = computeDashboardStats(boards);
    expect(stats.nearestDeadline).toHaveLength(5);
    expect(stats.nearestDeadline.map((b) => b.daysLeft)).toEqual([1, 2, 3, 7, 8]);
  });

  it('activeBoards/nearestDeadline include ALL boards from backend, including done -- hiding is now archive-only, not status-based', () => {
    const boards = aggregateBoards([
      board('b1', 'Done', [{ signed: true, days_left: -99 }]),
      board('b2', 'Running', [{ actual_pct: 50, days_left: 5 }]),
      board('b3', 'NotStarted', [{ actual_pct: 0, days_left: 20 }]),
      board('b4', 'Hold', [{ on_hold: true, days_left: 3 }])
    ]);
    const stats = computeDashboardStats(boards);
    expect(stats.activeBoards.map((b) => b.boardName).sort()).toEqual(['Done', 'Hold', 'NotStarted', 'Running']);
    expect(stats.nearestDeadline.map((b) => b.boardName)).toContain('Done');
  });

  it('a freshly created empty board (0 big tasks) shows up in activeBoards, not just the stat counts', () => {
    const boards = aggregateBoards([board('b1', 'Fresh Empty Board', [])]);
    const stats = computeDashboardStats(boards);
    expect(stats.notStarted).toBe(1);
    expect(stats.activeBoards).toHaveLength(1);
    expect(stats.activeBoards[0].boardName).toBe('Fresh Empty Board');
  });
});

describe('truncateName', () => {
  it('leaves short names untouched', () => {
    expect(truncateName('Analisis')).toBe('Analisis');
  });

  it('truncates names longer than the max length and appends an ellipsis', () => {
    expect(truncateName('Tahap Analisis Mendalam')).toBe('Tahap Anali…');
  });

  it('respects a custom max length', () => {
    expect(truncateName('Hello World', 5)).toBe('Hell…');
  });
});

describe('boardColor', () => {
  it('is deterministic — same id always returns the same color', () => {
    const id = 'board-abc-123';
    expect(boardColor(id)).toBe(boardColor(id));
    expect(boardColor(id, true)).toBe(boardColor(id, true));
  });

  it('returns a different palette for dark vs light for at least one id', () => {
    // Bukan tiap id pasti beda (bisa kebetulan sama), tapi minimal SATU id
    // di antara beberapa contoh harus beda -- kalau semua sama, berarti
    // parameter `dark` gak ngaruh sama sekali (bug).
    const ids = ['a', 'bb', 'ccc', 'dddd', 'eeeee', 'ffffff'];
    const anyDifferent = ids.some((id) => boardColor(id) !== boardColor(id, true));
    expect(anyDifferent).toBe(true);
  });

  it('always returns a valid 6-digit hex color', () => {
    for (const id of ['x', 'board-1', 'a-very-long-board-id-string-here']) {
      expect(boardColor(id)).toMatch(/^#[0-9a-f]{6}$/i);
      expect(boardColor(id, true)).toMatch(/^#[0-9a-f]{6}$/i);
    }
  });
});
