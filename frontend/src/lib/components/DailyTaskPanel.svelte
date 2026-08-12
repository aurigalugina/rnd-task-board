<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import { dateRangeInclusive } from '$lib/dateRange';
  import type { AssignableUser, DailyTask } from '$lib/types';
  import Avatar from './Avatar.svelte';
  import DayProgressStatus from './DayProgressStatus.svelte';

  export let bigTaskId: string;
  // Anggota Big Task (dari BigTaskList) — membatasi pilihan PIC daily task &
  // reviewer clone-review ke anggota saja. Lihat
  // decision-log-bigtask-members-refactor-20260811.md.
  export let members: AssignableUser[] = [];

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

  let cloneForm: { dailyTaskId: string; title: string } | null = null;
  let cloneReviewerId = '';
  let cloneStart = '';
  let cloneEnd = '';
  let cloning = false;
  let cloneError: string | null = null;

  // Nama reviewer terpilih (buat preview judul "[Review <nama>] ...").
  $: minDate = $auth.user?.access_level === 'super_user' ? '' : new Date().toLocaleDateString('en-CA');
  $: cloneReviewerName = members.find((m) => m.id === cloneReviewerId)?.display_name ?? '';

  $: previewDates = startDate && endDate ? dateRangeInclusive(startDate, endDate) : [];
  $: picById = Object.fromEntries(assignableUsers.map((u) => [u.id, u]));

  // `silent` dipakai buat refresh SETELAH mutasi kecil (ubah rencana/status
  // per hari, dst) -- kalau loading di-toggle tiap kali, {#if loading} di
  // bawah bikin SELURUH panel (semua tabel/input) unmount-remount tiap satu
  // field disimpan, kerasa kayak "refresh" walau SPA (dilaporkan user
  // 2026-08-10). Cukup load ulang datanya diam-diam, Svelte keyed {#each}
  // (day.id/dt.id) yang urus update DOM in-place tanpa destroy elemen.
  async function load({ silent = false }: { silent?: boolean } = {}) {
    if (!silent) loading = true;
    try {
      const [tasks, users] = await Promise.all([
        api.get<DailyTask[]>(`/big-tasks/${bigTaskId}/daily-tasks`),
        assignableUsers.length ? Promise.resolve(assignableUsers) : api.get<AssignableUser[]>('/users/assignable')
      ]);
      dailyTasks = tasks;
      assignableUsers = users;
      // Default PIC = current user kalau dia anggota, else anggota pertama.
      if (!picUserId) {
        const meId = $auth.user?.id ?? '';
        picUserId = members.some((m) => m.id === meId) ? meId : members[0]?.id ?? '';
      }
      dispatch(
        'tasksLoaded',
        tasks.map((t) => ({ id: t.id, title: t.title }))
      );
    } catch (e) {
      error = (e as Error).message;
    } finally {
      if (!silent) loading = false;
    }
  }

  onMount(() => load());

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
      await load({ silent: true });
      dispatch('updated');
    } catch (e) {
      createError = (e as Error).message;
    } finally {
      creating = false;
    }
  }

  async function updateDayEntry(
    dayEntryId: string,
    patch: { planned_text?: string; progress_pct?: number; blocker_text?: string }
  ) {
    try {
      // Menandai 100% (Selesai) mengosongkan blocker (perilaku mockup, 05-api-contract.md §5).
      const body = patch.progress_pct === 100 ? { ...patch, blocker_text: '' } : patch;
      await api.patch(`/day-entries/${dayEntryId}`, body);
      await load({ silent: true });
      dispatch('updated');
    } catch (e) {
      error = (e as Error).message;
    }
  }

  // Tambah baris day entry baru di tanggal yang sama -- buat PIC yang mau
  // breakdown lebih dari satu task di 1 hari (day_entries gak lagi dibatasi
  // satu baris per tanggal, lihat decision-log-day-entry-add-delete-20260810.md).
  async function addDayEntry(dailyTaskId: string, entryDate: string) {
    try {
      await api.post(`/daily-tasks/${dailyTaskId}/day-entries`, { entry_date: entryDate });
      await load({ silent: true });
      dispatch('updated');
    } catch (e) {
      error = (e as Error).message;
    }
  }

  // Hapus permanen -- dipakai a.l. buat baris weekend/"lembur" yang PIC-nya
  // gak mau kerjain. actual_pct otomatis menyesuaikan (dihitung dari SEMUA
  // baris yang tersisa saat dibaca).
  async function deleteDayEntry(dayEntryId: string) {
    if (!confirm('Hapus baris ini? actual_pct akan dihitung ulang tanpa baris ini.')) return;
    try {
      await api.del(`/day-entries/${dayEntryId}`);
      await load({ silent: true });
      dispatch('updated');
    } catch (e) {
      error = (e as Error).message;
    }
  }

  function isWeekendDate(entryDate: string): boolean {
    const day = new Date(`${entryDate}T00:00:00Z`).getUTCDay();
    return day === 0 || day === 6;
  }

  // today di-compute sekali saat komponen mount (string YYYY-MM-DD lokal).
  // Baris lampau (entry_date < today) di-lock: semua input, tombol hapus,
  // dan tombol tambah per-tanggal di-disable — data historis tidak boleh diubah.
  const today = new Date().toLocaleDateString('en-CA'); // en-CA = YYYY-MM-DD
  function isPastDate(entryDate: string): boolean {
    return entryDate < today;
  }

  function openCloneForm(dailyTaskId: string, dtTitle: string) {
    cloneForm = { dailyTaskId, title: dtTitle };
    cloneReviewerId = members[0]?.id ?? '';
    cloneStart = '';
    cloneEnd = '';
    cloneError = null;
  }

  async function submitClone() {
    if (!cloneForm) return;
    cloneError = null;
    if (!cloneReviewerId) {
      cloneError = 'Pilih reviewer dulu.';
      return;
    }
    if (!cloneStart || !cloneEnd) {
      cloneError = 'Rentang tanggal wajib diisi.';
      return;
    }
    cloning = true;
    try {
      await api.post(`/daily-tasks/${cloneForm.dailyTaskId}/clone-review`, {
        reviewer_user_id: cloneReviewerId,
        start_date: cloneStart,
        end_date: cloneEnd
      });
      cloneForm = null;
      await load({ silent: true });
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
    <div class="daily-task-card" class:daily-task-card-review={dt.title.startsWith('[Review ')}>
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
            <button class="review-clone-btn" on:click={() => openCloneForm(dt.id, dt.title)}>+ Review</button>
          </div>
        </div>
      </div>

      {#if cloneForm?.dailyTaskId === dt.id}
        <form class="inline-form" on:submit|preventDefault={submitClone} style="margin:0; border-top:1px solid #C3C8CC">
          <span class="small muted">Reviewer:</span>
          <select class="inline-input" bind:value={cloneReviewerId} required style="width:180px">
            {#each members as m (m.id)}
              <option value={m.id}>{m.display_name} — {m.roles.join('/')}</option>
            {/each}
          </select>
          <span class="small">→ "[Review {cloneReviewerName}] {dt.title}" untuk tanggal:</span>
          <input class="inline-input" type="date" bind:value={cloneStart} min={minDate} required style="width:140px" />
          <input class="inline-input" type="date" bind:value={cloneEnd} min={minDate} required style="width:140px" />
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
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each dt.days as day (day.id)}
            {@const past = isPastDate(day.entry_date)}
            <tr class="{isWeekendDate(day.entry_date) ? 'row-weekend' : ''}{past ? ' row-past' : ''}">
              <td class="mono small">
                {day.entry_date}
                {#if isWeekendDate(day.entry_date)}<span class="lembur-badge">lembur</span>{/if}
                {#if past}<span class="past-badge">lampau</span>{/if}
              </td>
              <td>
                <input
                  class="inline-input inline-input-cell"
                  value={day.planned_text}
                  placeholder="(belum diisi)"
                  disabled={past}
                  on:change={(e) => updateDayEntry(day.id, { planned_text: e.currentTarget.value })}
                />
              </td>
              <td>
                <DayProgressStatus progressPct={day.progress_pct} disabled={past} onChange={(next) => updateDayEntry(day.id, { progress_pct: next })} />
              </td>
              <td>
                {#if day.progress_pct < 100}
                  <input
                    class="inline-input inline-input-cell"
                    value={day.blocker_text}
                    placeholder="Blocker / rencana lanjut..."
                    disabled={past}
                    on:change={(e) => updateDayEntry(day.id, { blocker_text: e.currentTarget.value })}
                  />
                {:else}
                  <span class="muted small">—</span>
                {/if}
              </td>
              <td class="day-row-actions">
                <button
                  class="icon-btn"
                  title={past ? 'Tanggal sudah lampau' : 'Tambah task lain di tanggal ' + day.entry_date}
                  aria-label="Tambah task lain di tanggal {day.entry_date}"
                  disabled={past}
                  on:click={() => addDayEntry(dt.id, day.entry_date)}
                >
                  ➕
                </button>
                <button
                  class="icon-btn icon-btn-danger"
                  title={past ? 'Tanggal sudah lampau' : 'Hapus baris ' + day.entry_date}
                  aria-label="Hapus baris {day.entry_date}"
                  disabled={past}
                  on:click={() => deleteDayEntry(day.id)}
                >
                  🗑
                </button>
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
        {#each members as u (u.id)}
          <option value={u.id}>{u.display_name} — {u.roles.join('/')}</option>
        {/each}
      </select>
      <div class="inline-form-dates">
        <label class="small muted">
          Mulai
          <input class="inline-input" type="date" bind:value={startDate} min={minDate} required />
        </label>
        <label class="small muted">
          Selesai
          <input class="inline-input" type="date" bind:value={endDate} min={minDate} required />
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
