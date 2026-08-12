// Helper tampilan change_requests (Vision §6). Label & tone status diekstrak
// jadi fungsi murni supaya konsisten dipakai list + panel dan bisa ditest.
import type { CRStatus } from './types';

const STATUS_LABELS: Record<CRStatus, string> = {
  pending: 'Menunggu triase',
  approved: 'Disetujui',
  rejected: 'Ditolak',
  scheduled: 'Dijadwalkan'
};

// Tone dipetakan ke kelas badge yang sudah ada di app.css (win-* colors).
const STATUS_TONE: Record<CRStatus, 'neutral' | 'good' | 'bad' | 'accent'> = {
  pending: 'neutral',
  approved: 'good',
  rejected: 'bad',
  scheduled: 'accent'
};

export function crStatusLabel(status: string): string {
  return STATUS_LABELS[status as CRStatus] ?? status;
}

export function crStatusTone(status: string): 'neutral' | 'good' | 'bad' | 'accent' {
  return STATUS_TONE[status as CRStatus] ?? 'neutral';
}

// Status target yang boleh dipilih saat triase (semua kecuali dirinya sendiri
// tidak perlu — kita izinkan set ke mana saja termasuk balik ke pending).
export const CR_TRIAGE_TARGETS: CRStatus[] = ['approved', 'scheduled', 'rejected', 'pending'];
