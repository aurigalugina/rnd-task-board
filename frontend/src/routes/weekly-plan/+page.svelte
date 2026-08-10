<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { getWeekStart, shiftWeek, weekEnd } from '$lib/dateRange';

  type Row = {
    big_task_id: string;
    big_task_name: string;
    board_id: string;
    board_name: string;
    actual_pct: number;
    expected_pct: number;
    last_push: { callback_id: string; pushed_at: string } | null;
  };

  let weekStart = getWeekStart(new Date().toISOString().slice(0, 10));
  let rows: Row[] = [];
  let loading = true;
  let error: string | null = null;
  let pushingId: string | null = null;

  async function load() {
    loading = true;
    try {
      rows = await api.get<Row[]>(`/weekly-plan?week_start=${weekStart}`);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(load);
  $: if (weekStart) load();

  async function push(bigTaskId: string) {
    pushingId = bigTaskId;
    try {
      await api.post('/weekly-plan/push', { big_task_id: bigTaskId, week_start: weekStart });
      await load();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      pushingId = null;
    }
  }

  function formatPushedAt(iso: string): string {
    return iso.slice(0, 16).replace('T', ' ');
  }
</script>

<div class="weekplan-header">
  <button class="quick-btn" on:click={() => (weekStart = shiftWeek(weekStart, -1))}>&larr; Minggu lalu</button>
  <span class="weekplan-range">{weekStart} → {weekEnd(weekStart)}</span>
  <button class="quick-btn" on:click={() => (weekStart = shiftWeek(weekStart, 1))}>Minggu depan &rarr;</button>
</div>

<p class="empty-note" style="margin-bottom:10px">
  Rangkuman ini dihitung dari baris harian di Daily Task yang jatuh di rentang minggu ini. Push ke
  HR bisa diulang tiap hari (upsert) — callback ID tetap sama, cuma timestamp yang update.
</p>

{#if loading}
  <p>Memuat...</p>
{:else if error}
  <p class="small" style="color:var(--win-red)">{error}</p>
{:else}
  <table class="sheet-table weekplan-table">
    <thead>
      <tr>
        <th>Board</th>
        <th>Big task (topic)</th>
        <th>Actual</th>
        <th>Expected</th>
        <th>Push to HR</th>
        <th>Last pushed</th>
        <th>Callback ID</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row (row.big_task_id)}
        <tr>
          <td class="muted small">{row.board_name}</td>
          <td>{row.big_task_name}</td>
          <td class="mono">{row.actual_pct}%</td>
          <td class="mono">{row.expected_pct}%</td>
          <td>
            <button
              class="quick-btn quick-btn-done"
              on:click={() => push(row.big_task_id)}
              disabled={pushingId === row.big_task_id}
            >
              {pushingId === row.big_task_id ? 'Mengirim...' : row.last_push ? 'Push ulang' : 'Push'}
            </button>
          </td>
          <td class="mono small">{row.last_push ? formatPushedAt(row.last_push.pushed_at) : '—'}</td>
          <td class="mono small muted">{row.last_push ? row.last_push.callback_id : '—'}</td>
        </tr>
      {/each}
      {#if rows.length === 0}
        <tr><td colspan="7" class="empty-note">Tidak ada daily task di minggu ini.</td></tr>
      {/if}
    </tbody>
  </table>
{/if}
