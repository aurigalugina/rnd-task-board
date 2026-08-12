<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import { getWeekStart, shiftWeek, weekEnd } from '$lib/dateRange';
  import type { AssignableUser, TeamStatusRow } from '$lib/types';

  // Granularitas PER DAILY TASK. TglRencana/DueDate/UraianTask = preview
  // persis payload HR — ditampilkan di modal "Lihat" tanpa fetch tambahan.
  type Row = {
    daily_task_id: string;
    daily_task_title: string;
    big_task_name: string;
    board_id: string;
    board_name: string;
    actual_pct: number;
    expected_pct: number;
    tgl_rencana: string;
    due_date: string;
    uraian_task: string;
    last_push: { callback_id: string; pushed_at: string } | null;
  };

  const currentWeekStart = getWeekStart(new Date().toISOString().slice(0, 10));
  let weekStart = currentWeekStart;
  let rows: Row[] = [];
  let loading = true;
  let error: string | null = null;
  let pushingId: string | null = null;

  // Modal preview HR — null = tutup.
  let previewRow: Row | null = null;

  $: isSuperUser = $auth.user?.access_level === 'super_user';
  let asUserId = '';
  let assignableUsers: AssignableUser[] = [];
  let teamStatus: TeamStatusRow[] = [];

  $: isCurrentWeek = weekStart === currentWeekStart;

  async function load({ silent = false }: { silent?: boolean } = {}) {
    if (!silent) loading = true;
    try {
      const asParam = asUserId ? `&as_user_id=${asUserId}` : '';
      rows = await api.get<Row[]>(`/weekly-plan?week_start=${weekStart}${asParam}`);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      if (!silent) loading = false;
    }
  }

  async function loadTeamStatus() {
    if (!isSuperUser) return;
    try {
      teamStatus = await api.get<TeamStatusRow[]>(`/weekly-plan/team-status?week_start=${weekStart}`);
    } catch (e) {
      error = (e as Error).message;
    }
  }

  onMount(async () => {
    if (isSuperUser) {
      assignableUsers = await api.get<AssignableUser[]>('/users/assignable');
    }
    await Promise.all([load(), loadTeamStatus()]);
  });
  $: if (weekStart || asUserId) load();
  $: if (weekStart) loadTeamStatus();

  async function push(dailyTaskId: string) {
    pushingId = dailyTaskId;
    try {
      await api.post('/weekly-plan/push', {
        daily_task_id: dailyTaskId,
        week_start: weekStart,
        ...(asUserId ? { as_user_id: asUserId } : {})
      });
      await load({ silent: true });
      await loadTeamStatus();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      pushingId = null;
    }
  }

  function formatPushedAt(iso: string): string {
    return iso.slice(0, 16).replace('T', ' ');
  }

  function closePreview() {
    previewRow = null;
  }

  function handleEsc(e: KeyboardEvent) {
    if (e.key === 'Escape') closePreview();
  }
</script>

<svelte:window on:keydown={handleEsc} />

<div class="week-nav">
  <button class="week-nav-arrow" on:click={() => (weekStart = shiftWeek(weekStart, -1))} aria-label="Minggu sebelumnya">
    &lsaquo;
  </button>
  <div class="week-nav-range">
    <span class="week-nav-dates">{weekStart} → {weekEnd(weekStart)}</span>
    {#if isCurrentWeek}<span class="week-nav-label muted small">Minggu ini</span>{/if}
  </div>
  <button class="week-nav-arrow" on:click={() => (weekStart = shiftWeek(weekStart, 1))} aria-label="Minggu berikutnya">
    &rsaquo;
  </button>
</div>

{#if isSuperUser}
  <div class="inline-form" style="margin-bottom:10px">
    <span class="small muted">Lihat sebagai:</span>
    <select class="inline-input" bind:value={asUserId} style="flex:1 1 220px">
      <option value="">Diri sendiri ({$auth.user?.display_name})</option>
      {#each assignableUsers as u (u.id)}
        <option value={u.id}>{u.display_name}</option>
      {/each}
    </select>
  </div>
{/if}

<p class="empty-note" style="margin-bottom:10px">
  Satu baris per daily task yang punya entri harian di minggu ini. Klik <strong>Lihat</strong> untuk
  pratinjau data yang akan dikirim ke HR. Push bisa diulang tiap hari (upsert).
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
        <th>Big task</th>
        <th>Daily task (judul task ke HR)</th>
        <th>Actual</th>
        <th>Expected</th>
        <th>Aksi</th>
        <th>Last pushed</th>
        <th>Callback ID</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row (row.daily_task_id)}
        <tr>
          <td class="muted small">{row.board_name}</td>
          <td class="muted small">{row.big_task_name}</td>
          <td>{row.daily_task_title}</td>
          <td class="mono">{row.actual_pct}%</td>
          <td class="mono">{row.expected_pct}%</td>
          <td class="td-actions">
            <button
              class="quick-btn"
              on:click={() => (previewRow = row)}
            >
              Lihat
            </button>
            <button
              class="quick-btn quick-btn-done"
              on:click={() => push(row.daily_task_id)}
              disabled={pushingId === row.daily_task_id}
            >
              {pushingId === row.daily_task_id ? 'Mengirim...' : row.last_push ? 'Push ulang' : 'Push'}
            </button>
          </td>
          <td class="mono small">{row.last_push ? formatPushedAt(row.last_push.pushed_at) : '—'}</td>
          <td class="mono small muted">{row.last_push ? row.last_push.callback_id : '—'}</td>
        </tr>
      {/each}
      {#if rows.length === 0}
        <tr><td colspan="8" class="empty-note">Tidak ada daily task di minggu ini.</td></tr>
      {/if}
    </tbody>
  </table>
{/if}

{#if isSuperUser}
  <div class="section" style="margin-top:20px">
    <div class="section-title">Status push tim minggu ini</div>
    <table class="sheet-table weekplan-table">
      <thead>
        <tr><th>Nama</th><th>Daily task minggu ini</th><th>Sudah di-push</th><th>Status</th></tr>
      </thead>
      <tbody>
        {#each teamStatus as row (row.user_id)}
          <tr>
            <td>{row.display_name}</td>
            <td class="mono small">{row.total_daily_tasks}</td>
            <td class="mono small">{row.pushed_daily_tasks}</td>
            <td>
              {#if row.total_daily_tasks === 0}
                <span class="muted small">Tidak ada task</span>
              {:else if row.all_pushed}
                <span class="badge badge-win">Sudah push</span>
              {:else}
                <span class="badge badge-lose">Belum lengkap</span>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<!-- Modal pratinjau data HR — read-only, layout mirip form "Tambah Rencana Task Baru" -->
{#if previewRow}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="overlay" on:click={closePreview}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal-box wp-preview-modal" role="dialog" on:click|stopPropagation>
      <div class="titlebar" style="border-radius:4px 4px 0 0">
        <span>Pratinjau Data HR — {previewRow.daily_task_title}</span>
        <button class="modal-close" on:click={closePreview}>✕</button>
      </div>

      <div class="wp-preview-body">
        <div class="wp-preview-note">
          <span class="small">Data ini yang akan dikirim ke sistem HR saat klik Push.</span>
        </div>

        <div class="wp-field">
          <span class="wp-label">Tanggal Rencana Task</span>
          <div class="wp-value mono">{previewRow.tgl_rencana || '—'}</div>
        </div>

        <div class="wp-field">
          <span class="wp-label">Judul Task</span>
          <div class="wp-value">{previewRow.daily_task_title}</div>
        </div>

        <div class="wp-field">
          <span class="wp-label">Uraian Task</span>
          <pre class="wp-textarea">{previewRow.uraian_task || '(belum ada rencana diisi)'}</pre>
        </div>

        <div class="wp-field">
          <span class="wp-label">Due Date</span>
          <div class="wp-value mono">{previewRow.due_date || '—'}</div>
        </div>

        <div class="wp-field">
          <span class="wp-label">Target <span class="badge badge-neutral" style="margin-left:6px">Prosentase (%)</span></span>
          <div class="wp-value mono">{previewRow.expected_pct}%</div>
        </div>

        <div class="wp-field">
          <span class="wp-label">Capaian</span>
          <div class="wp-value mono">{previewRow.actual_pct}%</div>
        </div>

        <div class="wp-field">
          <span class="wp-label">Prosentase Capaian (Otomatis)</span>
          <div class="wp-value mono wp-pct-bar">
            <div class="wp-pct-fill" style="width:{previewRow.actual_pct}%"></div>
            <span class="wp-pct-label">{previewRow.actual_pct}%</span>
          </div>
        </div>

        {#if previewRow.last_push}
          <div class="wp-field">
            <span class="wp-label muted">Terakhir di-push</span>
            <div class="wp-value small muted">{formatPushedAt(previewRow.last_push.pushed_at)} · Callback: {previewRow.last_push.callback_id}</div>
          </div>
        {/if}

        <div class="wp-preview-actions">
          <button
            class="quick-btn quick-btn-done"
            on:click={() => { if (previewRow) push(previewRow.daily_task_id); closePreview(); }}
            disabled={pushingId === previewRow?.daily_task_id}
          >
            {previewRow.last_push ? 'Push ulang ke HR' : 'Push ke HR'}
          </button>
          <button class="quick-btn" on:click={closePreview}>Tutup</button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .td-actions {
    white-space: nowrap;
    display: flex;
    gap: 4px;
    align-items: center;
  }

  .wp-preview-modal {
    width: 480px;
    max-width: 96vw;
    padding: 0;
  }

  .wp-preview-body {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .wp-preview-note {
    background: color-mix(in srgb, var(--win-blue) 12%, var(--content-bg));
    border: 1px solid color-mix(in srgb, var(--win-blue) 30%, transparent);
    border-radius: 4px;
    padding: 8px 12px;
    color: var(--text-primary);
  }

  .wp-field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .wp-label {
    font-size: 11px;
    font-weight: 700;
    color: var(--text-primary);
    display: flex;
    align-items: center;
  }

  .wp-value {
    background: var(--content-alt);
    border: 1px solid var(--np-border, var(--win-border, #ccc));
    border-radius: 3px;
    padding: 6px 10px;
    font-size: 11px;
    color: var(--text-primary);
    min-height: 28px;
  }

  .wp-textarea {
    background: var(--content-alt);
    border: 1px solid var(--np-border, var(--win-border, #ccc));
    border-radius: 3px;
    padding: 8px 10px;
    font-size: 11px;
    color: var(--text-primary);
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    min-height: 60px;
    font-family: inherit;
    line-height: 1.5;
  }

  .wp-pct-bar {
    position: relative;
    height: 28px;
    overflow: hidden;
    display: flex;
    align-items: center;
  }

  .wp-pct-fill {
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    background: color-mix(in srgb, var(--win-green, #34d399) 35%, transparent);
    transition: width 0.3s ease;
  }

  .wp-pct-label {
    position: relative;
    z-index: 1;
    padding-left: 10px;
  }

  .wp-preview-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    margin-top: 4px;
    padding-top: 10px;
    border-top: 1px solid var(--np-border, var(--win-border, #ccc));
  }
</style>
