import { describe, expect, it } from 'vitest';
import { clampOnProgressPct, progressPctForStatus, statusFromProgressPct } from './dayProgress';

describe('statusFromProgressPct', () => {
  it('0 is belum', () => {
    expect(statusFromProgressPct(0)).toBe('belum');
  });
  it('100 is selesai', () => {
    expect(statusFromProgressPct(100)).toBe('selesai');
  });
  it('1-99 is on_progress', () => {
    expect(statusFromProgressPct(1)).toBe('on_progress');
    expect(statusFromProgressPct(50)).toBe('on_progress');
    expect(statusFromProgressPct(99)).toBe('on_progress');
  });
});

describe('progressPctForStatus', () => {
  it('belum always resolves to 0', () => {
    expect(progressPctForStatus('belum', 73)).toBe(0);
  });
  it('selesai always resolves to 100', () => {
    expect(progressPctForStatus('selesai', 12)).toBe(100);
  });
  it('on_progress defaults to 50 when coming from belum (0)', () => {
    expect(progressPctForStatus('on_progress', 0)).toBe(50);
  });
  it('on_progress defaults to 50 when coming from selesai (100)', () => {
    expect(progressPctForStatus('on_progress', 100)).toBe(50);
  });
  it('on_progress preserves an existing in-range value', () => {
    expect(progressPctForStatus('on_progress', 65)).toBe(65);
  });
});

describe('clampOnProgressPct', () => {
  it('clamps below 1 up to 1', () => {
    expect(clampOnProgressPct(0)).toBe(1);
    expect(clampOnProgressPct(-5)).toBe(1);
  });
  it('clamps above 99 down to 99', () => {
    expect(clampOnProgressPct(100)).toBe(99);
    expect(clampOnProgressPct(250)).toBe(99);
  });
  it('rounds fractional values', () => {
    expect(clampOnProgressPct(45.6)).toBe(46);
  });
  it('falls back to 1 for NaN', () => {
    expect(clampOnProgressPct(NaN)).toBe(1);
  });
  it('leaves valid in-range integers untouched', () => {
    expect(clampOnProgressPct(42)).toBe(42);
  });
});
