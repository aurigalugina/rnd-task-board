<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import { dateRangeInclusive } from '$lib/dateRange';
  import type { AssignableUser, DailyTask, DayEntry } from '$lib/types';
  import Avatar from './Avatar.svelte';
  import DayEntryEditModal from './DayEntryEditModal.svelte';
  import { Plus, Trash2, MessageSquare, ChevronDown, ChevronRight } from 'lucide-svelte';

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

  let entryModal: { entry: DayEntry | null; dailyTaskId: string; prefillDate: string; isPast: boolean } | null = null;

  // Collapse daily task card -- UI state doang, TIDAK persist ke DB (localStorage
  // browser per-device, default SEMUA collapsed). Lihat
  // decision-log-boards-dashboard-enhancements-20260820.md.
  const COLLAPSE_STORAGE_KEY = 'rndops_daily_task_expanded_ids';
  let expandedIds = new Set<string>();
  if (typeof window !== 'undefined') {
    try {
      const raw = localStorage.getItem(COLLAPSE_STORAGE_KEY);
      if (raw) expandedIds = new Set(JSON.parse(raw));
    } catch {
      // localStorage gak kebaca (mode private/disabled dst) -- fallback ke semua collapsed.
    }
  }
  function toggleCollapse(id: string) {
    if (expandedIds.has(id)) expandedIds.delete(id);
    else expandedIds.add(id);
    expandedIds = expandedIds; // Set mutation butuh reassign biar Svelte re-render.
    if (typeof window !== 'undefined') {
      try {
        localStorage.setItem(COLLAPSE_STORAGE_KEY, JSON.stringify([...expandedIds]));
      } catch {
        // abaikan -- collapse cuma preferensi tampilan, gak fatal kalau gagal disimpan.
      }
    }
  }

  // Filter status: default cuma nampilin yang "ongoing" (belum 100% DAN belum
  // lewat end_date) -- toggle buat nampilin yang sudah selesai/lampau juga.
  let showCompleted = false;
  function isOngoing(dt: DailyTask): boolean {
    return !(dt.actual_pct === 100 || dt.end_date < today);
  }
  $: visibleDailyTasks = showCompleted ? dailyTasks : dailyTasks.filter(isOngoing);
  $: hiddenCount = dailyTasks.length - visibleDailyTasks.length;

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
  // Baris lampau (entry_date < today) di-lock: edit dari modal hanya-baca.
  const today = new Date().toLocaleDateString('en-CA'); // en-CA = YYYY-MM-DD
  function isPastDate(entryDate: string): boolean {
    return entryDate < today;
  }

  function openEditModal(day: DayEntry, dailyTaskId: string, past: boolean) {
    entryModal = { entry: day, dailyTaskId, prefillDate: day.entry_date, isPast: past };
  }

  function openNewEntryModal(dailyTaskId: string, prefillDate: string) {
    entryModal = { entry: null, dailyTaskId, prefillDate, isPast: false };
  }

  async function handleModalSaved() {
    entryModal = null;
    await load({ silent: true });
    dispatch('updated');
  }

  async function handleModalDeleted() {
    entryModal = null;
    await load({ silent: true });
    dispatch('updated');
  }

  function progressBadge(pct: number): { label: string; cls: string } {
    if (pct === 100) return { label: 'Selesai', cls: 'badge-good' };
    if (pct > 0) return { label: `${pct}%`, cls: 'badge-accent' };
    return { label: 'Belum', cls: 'badge-neutral' };
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

  {#if dailyTasks.length > 0}
    <label class="small muted daily-task-filter">
      <input type="checkbox" bind:checked={showCompleted} />
      Tampilkan yang sudah selesai/lampau {#if !showCompleted && hiddenCount > 0}({hiddenCount} disembunyikan){/if}
    </label>
  {/if}

  {#each visibleDailyTasks as dt (dt.id)}
    {@const pic = picById[dt.pic_user_id]}
    {@const expanded = expandedIds.has(dt.id)}
    <div class="daily-task-card" class:daily-task-card-review={dt.title.startsWith('[Review ')}>
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="daily-task-head" on:click={() => toggleCollapse(dt.id)} style="cursor:pointer">
        <div style="display:flex;align-items:center;gap:4px">
          {#if expanded}<ChevronDown size={13} />{:else}<ChevronRight size={13} />{/if}
          <span class="daily-task-title">{dt.title}</span>
          {#if dt.is_baseline}<span class="badge badge-neutral" title="Progress awal migrasi data (adjustment percentage), bukan Daily Task PIC biasa">Adjustment</span>{/if}
          <span class="muted small" style="margin-left:8px">{dt.start_date} → {dt.end_date}</span>
        </div>
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div class="daily-task-head-right" on:click|stopPropagation>
          {#if pic}
            <Avatar initials={pic.initials} size={20} title={pic.display_name} />
            <span class="small">{pic.display_name}</span>
          {/if}
          <span class="mono small accent-text">{dt.actual_pct}%</span>
          <button class="quick-btn" on:click={() => dispatch('jumpToComment', dt.id)}>
            <MessageSquare size={12} />&nbsp;Komentar
          </button>
          <button class="quick-btn" on:click={() => openCloneForm(dt.id, dt.title)}>+ Review</button>
        </div>
      </div>

      {#if expanded}
        {#if cloneForm?.dailyTaskId === dt.id}
          <form class="inline-form" on:submit|preventDefault={submitClone} style="margin:0; border-top:1px solid var(--np-border, #C3C8CC)">
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
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each dt.days as day (day.id)}
              {@const past = isPastDate(day.entry_date)}
              {@const pb = progressBadge(day.progress_pct)}
              <!-- svelte-ignore a11y-click-events-have-key-events -->
              <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
              <tr class="de-row {isWeekendDate(day.entry_date) ? 'row-weekend' : ''} {past ? 'row-past' : ''}"
                on:click={() => openEditModal(day, dt.id, past)}>
                <td class="mono small">
                  {day.entry_date}
                  {#if isWeekendDate(day.entry_date)}<span class="lembur-badge">lembur</span>{/if}
                  {#if past}<span class="past-badge">lampau</span>{/if}
                </td>
                <td class="de-planned-cell small">{day.planned_text || '—'}</td>
                <td><span class="badge {pb.cls}">{pb.label}</span></td>
                <!-- svelte-ignore a11y-click-events-have-key-events -->
                <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
                <td class="day-row-actions" on:click|stopPropagation>
                  <button
                    class="icon-btn"
                    title={past ? 'Tanggal sudah lampau' : 'Tambah task lain di tanggal ' + day.entry_date}
                    aria-label="Tambah task lain di tanggal {day.entry_date}"
                    disabled={past}
                    on:click={() => openNewEntryModal(dt.id, day.entry_date)}
                  >
                    <Plus size={13} />
                  </button>
                  <button
                    class="icon-btn icon-btn-danger"
                    title={past ? 'Tanggal sudah lampau' : 'Hapus baris ' + day.entry_date}
                    aria-label="Hapus baris {day.entry_date}"
                    disabled={past}
                    on:click={() => deleteDayEntry(day.id)}
                  >
                    <Trash2 size={13} />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  {/each}

  {#if dailyTasks.length === 0}
    <div class="empty-note">Belum ada daily task buat big task ini.</div>
  {:else if visibleDailyTasks.length === 0}
    <div class="empty-note">Semua daily task di sini sudah selesai/lampau — centang di atas buat lihat.</div>
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

{#if entryModal}
  <DayEntryEditModal
    dailyTaskId={entryModal.dailyTaskId}
    entry={entryModal.entry}
    prefillDate={entryModal.prefillDate}
    minDate={entryModal.entry === null ? minDate : ''}
    isPast={entryModal.isPast}
    on:saved={handleModalSaved}
    on:deleted={handleModalDeleted}
    on:close={() => (entryModal = null)}
  />
{/if}
