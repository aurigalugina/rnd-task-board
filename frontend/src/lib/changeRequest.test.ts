import { describe, expect, it } from 'vitest';
import { crStatusLabel, crStatusTone } from './changeRequest';

describe('crStatusLabel', () => {
  it('memetakan tiap status ke label Indonesia', () => {
    expect(crStatusLabel('pending')).toBe('Menunggu triase');
    expect(crStatusLabel('approved')).toBe('Disetujui');
    expect(crStatusLabel('rejected')).toBe('Ditolak');
    expect(crStatusLabel('scheduled')).toBe('Dijadwalkan');
  });

  it('fallback ke nilai mentah untuk status tak dikenal', () => {
    expect(crStatusLabel('weird')).toBe('weird');
  });
});

describe('crStatusTone', () => {
  it('memetakan status ke tone badge', () => {
    expect(crStatusTone('approved')).toBe('good');
    expect(crStatusTone('rejected')).toBe('bad');
    expect(crStatusTone('scheduled')).toBe('accent');
    expect(crStatusTone('pending')).toBe('neutral');
  });

  it('fallback neutral untuk status tak dikenal', () => {
    expect(crStatusTone('weird')).toBe('neutral');
  });
});
