import { writable } from 'svelte/store';
import { api } from '$lib/api/client';
import type { InAppAlert } from '$lib/types';

export const alerts = writable<InAppAlert[]>([]);

let pollTimer: ReturnType<typeof setInterval> | null = null;

export async function loadAlerts() {
  try {
    const data = await api.get<InAppAlert[]>('/notifications');
    alerts.set(data ?? []);
  } catch {
    // Gagal silently — notif bukan fitur kritis yang harus block UI
  }
}

export function startAlertPolling() {
  loadAlerts();
  if (pollTimer) clearInterval(pollTimer);
  pollTimer = setInterval(loadAlerts, 60_000);
}

export function stopAlertPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
  alerts.set([]);
}
