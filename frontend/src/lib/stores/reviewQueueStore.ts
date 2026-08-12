// Shared store antara +layout.svelte (badge notifikasi, FR-NTF-01) dan
// routes/review-queue/+page.svelte (halaman antrean) — supaya keduanya selalu
// lihat data yang sama. Tanpa ini, mark-reviewed dari halaman tidak otomatis
// mengurangi angka badge di topbar sampai reload/buka-tutup dropdown lagi
// (staleness yang sama polanya seperti BigTaskList<->DailyTaskPanel di Fase 1,
// bedanya di sini dua route sibling, bukan parent/child, jadi dipakai store).
import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export type QueueItem = {
  type: string;
  id: string;
  title: string;
  source_daily_task_title: string;
  reviewed: boolean;
  big_task_id: string;
  big_task_name: string;
  board_id: string;
  board_name: string;
};

export const reviewQueue = writable<QueueItem[]>([]);

export async function loadReviewQueue() {
  try {
    const items = await api.get<QueueItem[]>('/review-queue');
    reviewQueue.set(items);
  } catch {
    reviewQueue.set([]);
  }
}

export async function markItemReviewed(item: Pick<QueueItem, 'type' | 'id'>) {
  await api.post(`/review-queue/${item.type}/${item.id}/mark-reviewed`);
  reviewQueue.update((items) => items.filter((i) => i.id !== item.id));
}
