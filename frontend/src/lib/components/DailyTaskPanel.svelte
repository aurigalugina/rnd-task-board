<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import { dateRangeInclusive } from '$lib/dateRange';
  import type { AssignableUser, DailyTask } from '$lib/types';
  import Avatar from './Avatar.svelte';

  export let bigTaskId: string;

  // BigTaskList menampilkan actual_pct/expected_pct/verdict/status sign-off
  // agregat milik Big Task ini di header-nya — nilai itu HARUS ikut refresh
  // begitu ada perubahan di sini (create/update), jadi event ini dipakai
  // parent buat re-fetch daftar big task-nya. Tanpa ini, header sempat basi
  // dan tombol sign-off bisa gagal 409 padahal progress sebenarnya sudah 100%.
  const dispatch = createEventDispatcher<{ updated: void; tasksLoaded: { id: string; title: string }[]; jumpToComment: string }>();

  let dailyTasks: DailyTask[] = [];
  let assignableUsers: AssignableUser[] = [];
  let loading = true;
  let error: string | null = null;

  let showCreateForm = false;
  let title = '';
  let picUserId = '';
  let startDate = '';
  let endDate = '';
  let creating = false;
  let createError: string | null = null;

  let cloneForm: { dailyTaskId: string; roleTag: 'SPV' | 'QA' } | null = null;
  let cloneStart = '';
  let cloneEnd = '';
  let cloning = false;
  let cloneError: string | null = null;

  $: previewDates = startDate && endDate ? dateRangeInclusive(startDate, endDate) : [];
  $: picById = Object.fromEntries(assignableUsers.map((u) => [u.id, u]));

  async function load() {
    loading = true;
    try {
      const [tasks, users] = await Promise.all([
        api.get<DailyTask[]>(`/big-tasks/${bigTaskId}/daily-tasks`),
        assignableUsers.length ? Promise.resolve(assignableUsers) : api.get<AssignableUser[]>('/users/assignable')
      ]);
      dailyTasks = tasks;
      assignableUsers = users;
      if (!picUserId) picUserId = $auth.user?.id ?? assignableUsers[0]?.id ?? '';
      dispatch(
        'tasksLoaded',
        tasks.map((t) => ({ id: t.id, title: t.title }))
      );
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(load);

  async function createDailyTask() {
    createError = null;
    if (previewDates.length === 0) {
      createError = 'Rentang tanggal tidak valid.';
      return;
    }
    creating = true;
    try {
      await api.post(`/big-tasks/${bigTaskId}/daily-tasks`, {
        title,
        pic_user_id: picUserId,
        start_date: startDate,
        end_date: endDate
      });
      title = '';
      startDate = '';
      endDate = '';
      showCreateForm = false;
      await load();
      dispatch('updated');
    } catch (e) {
      createError = (e as Error).message;
    } finally {
      creating = false;
    }
  }

  async function updateDayEntry(
    dayEntryId: string,
    patch: { planned_text?: string; is_done?: boolean; blocker_text?: string }
  ) {
    try {
      // Menandai selesai mengosongkan blocker (perilaku mockup, 05-api-contract.md §5).
      const body = patch.is_done === true ? { ...patch, blocker_text: '' } : patch;
      await api.patch(`/day-entries/${dayEntryId}`, body);
      await load();
      dispatch('updated');
    } catch (e) {
      error = (e as Error).message;
    }
  }

  function isWeekendDate(entryDate: string): boolean {
    const day = new Date(`${entryDate}T00:00:00Z`).getUTCDay();
    return day === 0 || day === 6;
  }

  function openCloneForm(dailyTaskId: string, roleTag: 'SPV' | 'QA') {
    cloneForm = { dailyTaskId, roleTag };
    cloneStart = '';
    cloneEnd = '';
    cloneError = null;
  }

  async function submitClone() {
    if (!cloneForm) return;
    cloneError = null;
    if (!cloneStart || !cloneEnd) {
      cloneError = 'Rentang tanggal wajib diisi.';
      return;
    }
    cloning = true;
    try {
      await api.post(`/daily-tasks/${cloneForm.dailyTaskId}/clone-review`, {
        role_tag: cloneForm.roleTag,
        start_date: cloneStart,
        end_date: cloneEnd
      });
      cloneForm = null;
      await load();
      dispatch('updated');
    } catch (e) {
      cloneError = (e as Error).message;
    } finally {
      cloning = false;
    }
  }
</script>

{#if loading}
  <p class="small muted">Memuat daily task...</p>
{:else}
  {#if error}
    <p class="small" style="color:var(--win-red)">{error}</p>
  {/if}

  {#each dailyTasks as dt (dt.id)}
    {@const pic = picById[dt.pic_user_id]}
    <div class="daily-task-card">
      <div class="daily-task-head">
        <div>
          <span class="daily-task-title">{dt.title}</span>
          <span class="muted small" style="margin-left:8px">{dt.start_date} → {dt.end_date}</span>
        </div>
        <div class="daily-task-head-right">
          {#if pic}
            <Avatar initials={pic.initials} size={20} title={pic.display_name} />
            <span class="small">{pic.display_name}</span>
          {/if}
          <span class="mono small accent-text">{dt.actual_pct}%</span>
          <button class="comment-jump-btn" on:click={() => dispatch('jumpToComment', dt.id)}>💬 Komentar</button>
          <div class="review-clone-group">
            <span class="muted small">Review:</span>
            <button class="review-clone-btn" on:click={() => openCloneForm(dt.id, 'SPV')}>SPV</button>
            <button class="review-clone-btn" on:click={() => openCloneForm(dt.id, 'QA')}>QA</button>
          </div>
        </div>
      </div>

      {#if cloneForm?.dailyTaskId === dt.id}
        <form class="inline-form" on:submit|preventDefault={submitClone} style="margin:0; border-top:1px solid #C3C8CC">
          <span class="small">"[Review {cloneForm.roleTag}] {dt.title}" untuk tanggal:</span>
          <input class="inline-input" type="date" bind:value={cloneStart} required style="width:140px" />
          <input class="inline-input" type="date" bind:value={cloneEnd} required style="width:140px" />
          {#if cloneError}<span class="small" style="color:var(--win-red)">{cloneError}</span>{/if}
          <button class="quick-btn quick-btn-done" type="submit" disabled={cloning}>{cloning ? 'Menyimpan...' : 'Buat'}</button>
          <button class="quick-btn" type="button" on:click={() => (cloneForm = null)}>Batal</button>
        </form>
      {/if}
      <table class="sheet-table daily-day-table">
        <thead>
          <tr>
            <th>Tanggal</th>
            <th>Rencana</th>
            <th>Status</th>
            <th>Blocker / catatan lanjutan</th>
          </tr>
        </thead>
        <tbody>
          {#each dt.days as day (day.id)}
            <tr class={isWeekendDate(day.entry_date) ? 'row-weekend' : ''}>
              <td class="mono small">
                {day.entry_date}
                {#if isWeekendDate(day.entry_date)}<span class="lembur-badge">lembur</span>{/if}
              </td>
              <td>
                <input
                  class="inline-input inline-input-cell"
                  value={day.planned_text}
                  placeholder="(belum diisi)"
                  on:change={(e) => updateDayEntry(day.id, { planned_text: e.currentTarget.value })}
                />
              </td>
              <td>
                <button
                  class="day-status-btn {day.is_done ? 'day-status-done' : 'day-status-open'}"
                  on:click={() => updateDayEntry(day.id, { is_done: !day.is_done })}
                >
                  {day.is_done ? 'Selesai' : 'Belum'}
                </button>
              </td>
              <td>
                {#if !day.is_done}
                  <input
                    class="inline-input inline-input-cell"
                    value={day.blocker_text}
                    placeholder="Blocker / rencana lanjut..."
                    on:change={(e) => updateDayEntry(day.id, { blocker_text: e.currentTarget.value })}
                  />
                {:else}
                  <span class="muted small">—</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/each}

  {#if dailyTasks.length === 0}
    <div class="empty-note">Belum ada daily task buat big task ini.</div>
  {/if}

  {#if showCreateForm}
    <form class="inline-form inline-form-daily" on:submit|preventDefault={createDailyTask}>
      <input class="inline-input" placeholder="Judul daily task" bind:value={title} required />
      <select class="inline-input" bind:value={picUserId} required>
        {#each assignableUsers as u (u.id)}
          <option value={u.id}>{u.display_name} — {u.roles.join('/')}</option>
        {/each}
      </select>
      <div class="inline-form-dates">
        <label class="small muted">
          Mulai
          <input class="inline-input" type="date" bind:value={startDate} required />
        </label>
        <label class="small muted">
          Selesai
          <input class="inline-input" type="date" bind:value={endDate} required />
        </label>
      </div>
      {#if createError}<span class="small" style="color:var(--win-red)">{createError}</span>{/if}
      <div class="inline-form-actions">
        <button class="quick-btn quick-btn-done" type="submit" disabled={creating}>
          {creating ? 'Menyimpan...' : `Buat ${previewDates.length} baris harian`}
        </button>
        <button class="quick-btn" type="button" on:click={() => (showCreateForm = false)}>Batal</button>
      </div>
    </form>
  {:else}
    <button class="add-card-ghost" on:click={() => (showCreateForm = true)}>+ Tambah daily task</button>
  {/if}
{/if}
