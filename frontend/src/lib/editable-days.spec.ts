import { describe, it, expect } from 'vitest';

/**
 * Test untuk logic "entries dapat diedit dalam 3 hari ke belakang"
 * Entries 3+ hari ke belakang = read-only
 */

function isDayOlderThan3Days(entryDate: string, today: string): boolean {
  const entryDateObj = new Date(`${entryDate}T00:00:00Z`);
  const todayObj = new Date(`${today}T00:00:00Z`);
  const diffMs = todayObj.getTime() - entryDateObj.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  return diffDays > 3;
}

describe('isDayOlderThan3Days', () => {
  const today = '2026-09-01'; // Fixed date untuk consistency testing

  it('future dates → editable (false)', () => {
    expect(isDayOlderThan3Days('2026-09-02', today)).toBe(false);
    expect(isDayOlderThan3Days('2026-09-10', today)).toBe(false);
  });

  it('today → editable (false)', () => {
    expect(isDayOlderThan3Days(today, today)).toBe(false);
  });

  it('1 day ago → editable (false)', () => {
    expect(isDayOlderThan3Days('2026-08-31', today)).toBe(false);
  });

  it('3 days ago → editable (false)', () => {
    expect(isDayOlderThan3Days('2026-08-29', today)).toBe(false);
  });

  it('4 days ago → read-only (true)', () => {
    expect(isDayOlderThan3Days('2026-08-28', today)).toBe(true);
  });

  it('7 days ago → read-only (true)', () => {
    expect(isDayOlderThan3Days('2026-08-25', today)).toBe(true);
  });

  it('30 days ago → read-only (true)', () => {
    expect(isDayOlderThan3Days('2026-08-02', today)).toBe(true);
  });
});
