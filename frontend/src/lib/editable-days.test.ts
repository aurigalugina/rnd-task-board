/**
 * Test untuk logic "entries dapat diedit dalam 3 hari ke belakang"
 * Entries 3+ hari ke belakang = read-only
 */

describe('isDayOlderThan3Days', () => {
  const today = new Date().toLocaleDateString('en-CA');
  
  function isDayOlderThan3Days(entryDate: string): boolean {
    const entryDateObj = new Date(`${entryDate}T00:00:00Z`);
    const todayObj = new Date(`${today}T00:00:00Z`);
    const diffMs = todayObj.getTime() - entryDateObj.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    return diffDays > 3;
  }

  it('future dates (hari depan, minggu depan) → editable (false)', () => {
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    const tomorrowStr = tomorrow.toLocaleDateString('en-CA');
    
    expect(isDayOlderThan3Days(tomorrowStr)).toBe(false);
  });

  it('today → editable (false)', () => {
    expect(isDayOlderThan3Days(today)).toBe(false);
  });

  it('1 day ago → editable (false)', () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const yesterdayStr = yesterday.toLocaleDateString('en-CA');
    
    expect(isDayOlderThan3Days(yesterdayStr)).toBe(false);
  });

  it('3 days ago → editable (false)', () => {
    const threeDaysAgo = new Date();
    threeDaysAgo.setDate(threeDaysAgo.getDate() - 3);
    const threeDaysAgoStr = threeDaysAgo.toLocaleDateString('en-CA');
    
    expect(isDayOlderThan3Days(threeDaysAgoStr)).toBe(false);
  });

  it('4 days ago → read-only (true)', () => {
    const fourDaysAgo = new Date();
    fourDaysAgo.setDate(fourDaysAgo.getDate() - 4);
    const fourDaysAgoStr = fourDaysAgo.toLocaleDateString('en-CA');
    
    expect(isDayOlderThan3Days(fourDaysAgoStr)).toBe(true);
  });

  it('7 days ago → read-only (true)', () => {
    const sevenDaysAgo = new Date();
    sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7);
    const sevenDaysAgoStr = sevenDaysAgo.toLocaleDateString('en-CA');
    
    expect(isDayOlderThan3Days(sevenDaysAgoStr)).toBe(true);
  });

  it('30 days ago → read-only (true)', () => {
    const thirtyDaysAgo = new Date();
    thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
    const thirtyDaysAgoStr = thirtyDaysAgo.toLocaleDateString('en-CA');
    
    expect(isDayOlderThan3Days(thirtyDaysAgoStr)).toBe(true);
  });
});
