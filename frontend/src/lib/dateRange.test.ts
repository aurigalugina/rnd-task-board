import { describe, expect, it } from 'vitest';
import { dateRangeInclusive, getWeekStart, isWeekend, shiftWeek, weekEnd } from './dateRange';

describe('dateRangeInclusive', () => {
  it('includes both start and end date (inclusive)', () => {
    expect(dateRangeInclusive('2026-08-05', '2026-08-07')).toEqual([
      '2026-08-05',
      '2026-08-06',
      '2026-08-07'
    ]);
  });

  it('returns a single date when start equals end', () => {
    expect(dateRangeInclusive('2026-08-05', '2026-08-05')).toEqual(['2026-08-05']);
  });

  it('returns empty array when end is before start', () => {
    expect(dateRangeInclusive('2026-08-07', '2026-08-05')).toEqual([]);
  });

  it('returns empty array for invalid dates', () => {
    expect(dateRangeInclusive('not-a-date', '2026-08-05')).toEqual([]);
  });

  it('crosses a month boundary correctly', () => {
    expect(dateRangeInclusive('2026-08-30', '2026-09-02')).toEqual([
      '2026-08-30',
      '2026-08-31',
      '2026-09-01',
      '2026-09-02'
    ]);
  });
});

describe('isWeekend', () => {
  it('flags Saturday and Sunday as weekend', () => {
    expect(isWeekend('2026-08-08')).toBe(true); // Saturday
    expect(isWeekend('2026-08-09')).toBe(true); // Sunday
  });

  it('does not flag weekdays as weekend', () => {
    expect(isWeekend('2026-08-10')).toBe(false); // Monday
    expect(isWeekend('2026-08-07')).toBe(false); // Friday
  });
});

describe('getWeekStart', () => {
  it('returns the same date when already a Monday', () => {
    expect(getWeekStart('2026-08-03')).toBe('2026-08-03'); // Senin
  });

  it('rolls back to Monday from a mid-week date', () => {
    expect(getWeekStart('2026-08-06')).toBe('2026-08-03'); // Kamis -> Senin minggu itu
  });

  it('rolls back to Monday from Sunday (edge case getUTCDay()===0)', () => {
    expect(getWeekStart('2026-08-09')).toBe('2026-08-03'); // Minggu -> Senin minggu itu, bukan minggu depan
  });

  it('rolls back correctly across a month boundary', () => {
    expect(getWeekStart('2026-09-01')).toBe('2026-08-31'); // Selasa 1 Sep -> Senin 31 Agu
  });
});

describe('shiftWeek', () => {
  it('moves forward one week', () => {
    expect(shiftWeek('2026-08-03', 1)).toBe('2026-08-10');
  });

  it('moves backward one week', () => {
    expect(shiftWeek('2026-08-03', -1)).toBe('2026-07-27');
  });

  it('is a no-op with delta 0', () => {
    expect(shiftWeek('2026-08-03', 0)).toBe('2026-08-03');
  });
});

describe('weekEnd', () => {
  it('returns the Sunday 6 days after a Monday week_start', () => {
    expect(weekEnd('2026-08-03')).toBe('2026-08-09');
  });
});
