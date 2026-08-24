<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { AssignableUser, BigTask, DailyTask } from '$lib/types';
  import VerdictBadge from './VerdictBadge.svelte';
  import DualBar from './DualBar.svelte';
  import StatCard from './StatCard.svelte';
  import Avatar from './Avatar.svelte';
  import DailyTaskPanel from './DailyTaskPanel.svelte';
  import CommentModal from './CommentModal.svelte';
  import CheatSheetSection from './CheatSheetSection.svelte';
  import { Check, CirclePause, CirclePlay, Pencil, X } from 'lucide-svelte';

  export let boardId: string;

  let assignableUsers: AssignableUser[] = [];
  $: userById = Object.fromEntries(assignableUsers.map((u) => [u.id, u]));

  onMount(async () => {
    assignableUsers = await api.get<AssignableUser[]>('/users/assignable');
  });

  let dailyTasksForComments: { id: string; title: string }[] = [];
  // Modal komentar — dibuka lewat tombol Komentar di DailyTaskPanel.
  let commentModal: { open: boolean; jumpScope: string | null; jumpToken: number } = {
    open: false, jumpScope: null, jumpToken: 0
  };

  let bigTasks: BigTask[] = [];
  let loading = true;
  let error: string | null = null;
  let selectedBtId: string | null = null;
  let actionError = '';

  let showCreateForm = false;
  let name = '';
  let startDate = '';
  let deadline = '';
  let memberUserIds: string[] = [];
  // Persentase awal (opsional) -- bisa diisi langsung saat create, super_user
  // only. Sama dengan baseline_pct di edit, cuma gak perlu buka Edit lagi
  // abis Big Task kebuat. Lihat decision-log-bigtask-baseline-progress-20260824.md.
  let createBaselinePct = '';
  let creating = false;
  let createError: string | null = null;

  function toggleMember(userId: string) {
    memberUserIds = memberUserIds.includes(userId)
      ? memberUserIds.filter((id) => id !== userId)
      : [...memberUserIds, userId];
  }

  $: isSpv = $auth.user?.roles.includes('spv') ?? false;
  // Backdate signed_at & edit judul/tanggal Big Task -- super_user only, lihat
  // decision-log-verdict-backfill-signoff-20260820.md.
  $: isSuperUser = $auth.user?.access_level === 'super_user';
  let signOffDate = '';

  let editingBigTask = false;
  let editName = '';
  let editStartDate = '';
  let editDeadline = '';
  let savingBigTaskEdit = false;
  let bigTaskEditError = '';
  // "Baseline Awal" -- persentase progress yang sudah berjalan di lapangan
  // sebelum Big Task ini dicatat di sistem (mis. migrasi data ke staging).
  // Kosong = tidak diubah. Lihat decision-log-bigtask-baseline-progress-20260824.md.
  let editBaselinePct = '';
  let baselineExistedBeforeEdit = false;

  async function openEditBigTask() {
    if (!activeBt) return;
    editName = activeBt.name;
    editStartDate = activeBt.start_date;
    editDeadline = activeBt.deadline;
    bigTaskEditError = '';
    editBaselinePct = '';
    baselineExistedBeforeEdit = false;
    try {
      const days = await api.get<DailyTask[]>(`/big-tasks/${activeBt.id}/daily-tasks`);
      const baseline = days.find((d) => d.is_baseline);
      if (baseline) {
        editBaselinePct = String(baseline.actual_pct);
        baselineExistedBeforeEdit = true;
      }
    } catch {
      // gagal ambil baseline existing bukan blocker -- field tetap kosong,
      // user bisa isi ulang manual kalau perlu.
    }
    editingBigTask = true;
  }
  async function saveBigTaskEdit() {
    if (!activeBt) return;
    savingBigTaskEdit = true;
    bigTaskEditError = '';
    try {
      const payload: Record<string, unknown> = {
        name: editName,
        start_date: editStartDate,
        deadline: editDeadline
      };
      const trimmed = editBaselinePct.trim();
      if (trimmed === '') {
        if (baselineExistedBeforeEdit) payload.clear_baseline = true;
      } else {
        payload.baseline_pct = Number(trimmed);
      }
      await api.patch(`/big-tasks/${activeBt.id}`, payload);
      editingBigTask = false;
      await load({ silent: true });
    } catch (e) {
      bigTaskEditError = (e as Error).message;
    } finally {
      savingBigTaskEdit = false;
    }
  }

  // Anggota Big Task aktif (objek user lengkap) — dipakai buat summary di header
  // DAN diteruskan ke DailyTaskPanel supaya PIC picker + reviewer clone-review
  // dibatasi ke anggota. Lihat decision-log-bigtask-members-refactor-20260811.md.
  $: activeMembers = activeBt ? activeBt.member_user_ids.map((id) => userById[id]).filter(Boolean) : [];

  // Edit anggota Big Task existing (tambah/kurang, tetap min 2).
  let editingMembers = false;
  let editMemberIds: string[] = [];
  let savingMembers = false;
  let memberError = '';

  function openEditMembers() {
    editMemberIds = activeBt ? [...activeBt.member_user_ids] : [];
    memberError = '';
    editingMembers = true;
  }
  function toggleEditMember(userId: string) {
    editMemberIds = editMemberIds.includes(userId)
      ? editMemberIds.filter((id) => id !== userId)
      : [...editMemberIds, userId];
  }
  async function saveMembers() {
    if (!activeBt) return;
    if (editMemberIds.length < 2) {
      memberError = 'Minimal 2 anggota.';
      return;
    }
    savingMembers = true;
    memberError = '';
    try {
      await api.put(`/big-tasks/${activeBt.id}/members`, { member_user_ids: editMemberIds });
      editingMembers = false;
      await load({ silent: true });
    } catch (e) {
      memberError = (e as Error).message;
    } finally {
      savingMembers = false;
    }
  }

  // silent: dipakai buat refresh SETELAH mutasi (create/sign-off/undo, atau
  // event 'updated' dari DailyTaskPanel anak) -- kalau loading di-toggle,
  // {#if loading} di bawah unmount SELURUH project-layout (termasuk
  // DailyTaskPanel yang lagi dibuka), kerasa kayak "refresh" walau SPA
  // (dilaporkan user 2026-08-10). Cukup load ulang datanya diam-diam.
  async function load({ silent = false }: { silent?: boolean } = {}) {
    if (!silent) loading = true;
    try {
      bigTasks = await api.get<BigTask[]>(`/boards/${boardId}/big-tasks`);
      if (!selectedBtId || !bigTasks.some((b) => b.id === selectedBtId)) {
        selectedBtId = bigTasks[0]?.id ?? null;
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      if (!silent) loading = false;
    }
  }

  $: if (boardId) load();

  $: activeBt = bigTasks.find((b) => b.id === selectedBtId) ?? null;
  $: minDate = $auth.user?.access_level === 'super_user' ? '' : new Date().toLocaleDateString('en-CA');
  $: totalBt = bigTasks.length;
  $: doneBt = bigTasks.filter((b) => b.signed).length;
  $: wonBt = bigTasks.filter((b) => b.verdict === 'win').length;
  $: loseBt = bigTasks.filter((b) => b.verdict === 'lose').length;
  $: boardCompletion = totalBt ? Math.round(bigTasks.reduce((s, b) => s + b.actual_pct, 0) / totalBt) : 0;
  $: projectDone = totalBt > 0 && doneBt === totalBt;

  async function createBigTask() {
    createError = null;
    creating = true;
    try {
      const payload: Record<string, unknown> = {
        name,
        start_date: startDate,
        deadline,
        member_user_ids: memberUserIds
      };
      const trimmed = createBaselinePct.trim();
      if (trimmed !== '') payload.baseline_pct = Number(trimmed);
      await api.post(`/boards/${boardId}/big-tasks`, payload);
      name = '';
      startDate = '';
      deadline = '';
      memberUserIds = [];
      createBaselinePct = '';
      showCreateForm = false;
      await load({ silent: true });
    } catch (e) {
      createError = (e as Error).message;
    } finally {
      creating = false;
    }
  }

  async function signOff(bt: BigTask) {
    actionError = '';
    try {
      const body = isSuperUser && signOffDate ? { signed_at: signOffDate } : {};
      await api.post(`/big-tasks/${bt.id}/sign-off`, body);
      signOffDate = '';
      await load({ silent: true });
    } catch (e) {
      actionError = (e as Error).message;
    }
  }

  async function undoSignOff(bt: BigTask) {
    actionError = '';
    try {
      await api.del(`/big-tasks/${bt.id}/sign-off`);
      await load({ silent: true });
    } catch (e) {
      actionError = (e as Error).message;
    }
  }

  async function toggleOnHold(bt: BigTask) {
    actionError = '';
    try {
      await api.patch(`/big-tasks/${bt.id}/on-hold`, {});
      await load({ silent: true });
    } catch (e) {
      actionError = (e as Error).message;
    }
  }
</script>

{#if loading}
  <p>Memuat big task...</p>
{:else if error}
  <p class="small" style="color:var(--win-red)">{error}</p>
{:else}
  <div class="project-status-banner">
    <span class="project-status-dot {projectDone ? 'status-done' : 'status-progress'}" />
    <span class="small">
      Status project: <strong>{projectDone ? 'Selesai (semua big task sign-off)' : 'Berjalan'}</strong>
    </span>
    <span class="muted small">{doneBt}/{totalBt} big task sign-off</span>
  </div>

  <div class="stat-row" style="grid-template-columns:repeat(5,1fr)">
    <StatCard label="Total big task" value={totalBt} />
    <StatCard label="Big task selesai" value={doneBt} tone="good" />
    <StatCard label="Won" value={wonBt} tone="good" />
    <StatCard label="Lose" value={loseBt} tone="warn" />
    <StatCard label="Completion rate" value="{boardCompletion}%" tone="accent" />
  </div>

  <CheatSheetSection {boardId} />

  <div class="project-layout">
    <div class="bigtask-list">
      <div class="section-title" style="margin-bottom:8px">Big task</div>
      {#each bigTasks as bt (bt.id)}
        <button
          class="bigtask-list-item {selectedBtId === bt.id ? 'bigtask-list-item-active' : ''}"
          on:click={() => (selectedBtId = bt.id)}
        >
          <div class="bigtask-list-top">
            <span class="bigtask-list-name">{bt.name}</span>
            <div style="display:flex;gap:4px;align-items:center">
              {#if bt.on_hold}<span class="badge badge-neutral">Di hold</span>{/if}
              <VerdictBadge verdict={bt.verdict} />
            </div>
          </div>
          <div class="dualbar-track" style="margin-top:4px">
            <div class="dualbar-fill fill-good" style="width:{bt.actual_pct}%" />
          </div>
          <div class="bigtask-list-bottom">
            <span class="mono small muted">{bt.actual_pct}%</span>
            <span class="mono small {bt.days_left < 0 ? 'days-late' : 'muted'}">
              {bt.days_left < 0 ? `telat ${-bt.days_left}h` : `${bt.days_left}h lagi`}
            </span>
          </div>
        </button>
      {/each}

      <button class="add-card-ghost" on:click={() => (showCreateForm = true)}>+ Tambah big task</button>
    </div>

    <div class="bigtask-detail">
      {#if activeBt}
        <div class="section-title bigtask-detail-title">
          <span>{activeBt.name} — daily task</span>
          <div class="bigtask-detail-title-right">
            <VerdictBadge verdict={activeBt.verdict} />
            {#if isSuperUser}
              <button class="quick-btn" on:click={openEditBigTask} title="Edit judul/tanggal"><Pencil size={12} />&nbsp;Edit</button>
            {/if}
            {#if isSpv}
              <button
                class="hold-btn {activeBt.on_hold ? 'hold-btn-active' : ''}"
                on:click={() => toggleOnHold(activeBt)}
                title={activeBt.on_hold ? 'Lanjutkan task ini' : 'Tahan (on hold)'}
              >
                {#if activeBt.on_hold}
                  <CirclePlay size={12} />&nbsp;Lanjutkan
                {:else}
                  <CirclePause size={12} />&nbsp;On hold
                {/if}
              </button>
              {#if activeBt.signed}
                <button class="sign-btn sign-btn-active" on:click={() => undoSignOff(activeBt)}><Check size={12} />&nbsp;Signed done</button>
                {#if activeBt.signed_at_backdated_by}
                  <span class="muted small" title="Tanggal sign-off di-backdate manual oleh super_user">(backdated)</span>
                {/if}
              {:else}
                {#if isSuperUser}
                  <input
                    class="inline-input"
                    type="date"
                    bind:value={signOffDate}
                    style="width:130px"
                    title="Tanggal sign-off custom (opsional, kosongkan buat pakai hari ini)"
                  />
                {/if}
                <button
                  class="sign-btn"
                  disabled={activeBt.actual_pct < 100 || activeBt.on_hold}
                  title={activeBt.on_hold ? 'Task sedang di-hold' : activeBt.actual_pct < 100 ? 'Progress belum 100%' : ''}
                  on:click={() => signOff(activeBt)}
                >
                  Tandai selesai (SPV)
                </button>
              {/if}
            {/if}
          </div>
        </div>
        {#if actionError}<p class="small" style="color:var(--win-red)">{actionError}</p>{/if}

        <!-- Summary tim yang terlibat di Big Task ini (di atas daftar daily task). -->
        <div class="member-summary">
          <span class="small muted">Anggota tim:</span>
          {#each activeMembers as m (m.id)}
            <span class="member-chip">
              <Avatar initials={m.initials} size={18} title={m.display_name} />
              <span class="small">{m.display_name}</span>
              <span class="muted small">— {m.roles.join('/')}</span>
            </span>
          {/each}
          {#if activeMembers.length === 0}
            <span class="small" style="color:var(--win-amber)">Belum ada anggota — tambahkan dulu.</span>
          {/if}
          <button class="quick-btn member-manage-btn" on:click={openEditMembers}>Kelola anggota</button>
        </div>

        {#if editingMembers}
          <div class="reviewer-picker member-editor">
            <span class="small muted">Pilih anggota (min. 2)</span>
            {#each assignableUsers as u (u.id)}
              <label class="reviewer-picker-row small">
                <input type="checkbox" checked={editMemberIds.includes(u.id)} on:change={() => toggleEditMember(u.id)} />
                {u.display_name} <span class="muted">— {u.roles.join('/')}</span>
              </label>
            {/each}
            {#if memberError}<span class="small" style="color:var(--win-red)">{memberError}</span>{/if}
            <div class="inline-form-actions">
              <button class="quick-btn quick-btn-done" on:click={saveMembers} disabled={savingMembers || editMemberIds.length < 2}>
                {savingMembers ? 'Menyimpan...' : 'Simpan anggota'}
              </button>
              <button class="quick-btn" on:click={() => (editingMembers = false)}>Batal</button>
            </div>
          </div>
        {/if}

        {#key activeBt.id}
          <DailyTaskPanel
            bigTaskId={activeBt.id}
            members={activeMembers}
            on:updated={() => load({ silent: true })}
            on:tasksLoaded={(e) => (dailyTasksForComments = e.detail)}
            on:jumpToComment={(e) => {
              commentModal = { open: true, jumpScope: e.detail, jumpToken: commentModal.jumpToken + 1 };
            }}
          />
        {/key}
      {:else}
        <div class="empty-note">Pilih atau tambahkan big task dulu.</div>
      {/if}
    </div>
  </div>
{/if}

<!-- Modal tambah big task -->
{#if showCreateForm}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="overlay overlay-center" on:click={() => (showCreateForm = false)}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal-box bigtask-create-modal" role="dialog" on:click|stopPropagation>
      <div class="de-modal-header">
        <span class="de-modal-title">Tambah big task</span>
        <button class="de-close-btn icon-btn" on:click={() => (showCreateForm = false)}><X size={12} /></button>
      </div>
      <div class="modal-body">
        <form on:submit|preventDefault={createBigTask}>
          <div class="panel-field">
            <label class="small muted" for="bt-name">Nama big task</label>
            <input id="bt-name" class="inline-input" placeholder="Nama big task" bind:value={name} required />
          </div>
          <div class="panel-field">
            <span class="small muted">Rentang waktu</span>
            <div class="inline-form-dates" style="padding:0">
              <label class="small muted">
                Mulai
                <input class="inline-input" type="date" bind:value={startDate} min={minDate} required />
              </label>
              <label class="small muted">
                Deadline
                <input class="inline-input" type="date" bind:value={deadline} min={minDate} required />
              </label>
            </div>
          </div>
          <div class="panel-field">
            <span class="small muted">Anggota — siapa saja yang terlibat (min. 2)</span>
            <div class="reviewer-picker" style="margin-top:4px">
              {#each assignableUsers as u (u.id)}
                <label class="reviewer-picker-row small">
                  <input type="checkbox" checked={memberUserIds.includes(u.id)} on:change={() => toggleMember(u.id)} />
                  {u.display_name} <span class="muted">— {u.roles.join('/')}</span>
                </label>
              {/each}
            </div>
            {#if memberUserIds.length < 2}<span class="small muted">Pilih minimal 2 anggota.</span>{/if}
          </div>
          {#if isSuperUser}
            <div class="panel-field">
              <label class="small muted" for="bt-create-baseline">Persentase awal (opsional)</label>
              <input
                id="bt-create-baseline"
                class="inline-input"
                type="number"
                min="0"
                max="100"
                placeholder="mis. 40"
                bind:value={createBaselinePct}
              />
              <span class="small muted">
                Progress yang sudah berjalan di lapangan sebelum dicatat di sistem (mis. migrasi data ke
                staging) — dihitung sebagai Daily Task "Baseline Awal" terpisah. Bisa diisi/diadjust lagi
                kapan saja lewat Edit.
              </span>
            </div>
          {/if}
          {#if createError}<p class="small" style="color:var(--win-red);margin:0">{createError}</p>{/if}
          <div class="inline-form-actions" style="padding-top:4px">
            <button class="quick-btn quick-btn-done" type="submit" disabled={creating || memberUserIds.length < 2}>
              {creating ? 'Menyimpan...' : 'Simpan big task'}
            </button>
            <button class="quick-btn" type="button" on:click={() => (showCreateForm = false)}>Batal</button>
          </div>
        </form>
      </div>
    </div>
  </div>
{/if}

<!-- Modal edit big task (judul/tanggal, super_user only) -->
{#if editingBigTask && activeBt}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="overlay overlay-center" on:click={() => (editingBigTask = false)}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal-box bigtask-create-modal" role="dialog" on:click|stopPropagation>
      <div class="de-modal-header">
        <span class="de-modal-title">Edit big task</span>
        <button class="de-close-btn icon-btn" on:click={() => (editingBigTask = false)}><X size={12} /></button>
      </div>
      <div class="modal-body">
        <form on:submit|preventDefault={saveBigTaskEdit}>
          <div class="panel-field">
            <label class="small muted" for="bt-edit-name">Nama big task</label>
            <input id="bt-edit-name" class="inline-input" bind:value={editName} required />
          </div>
          <div class="panel-field">
            <span class="small muted">Rentang waktu</span>
            <div class="inline-form-dates" style="padding:0">
              <label class="small muted">
                Mulai
                <input class="inline-input" type="date" bind:value={editStartDate} required />
              </label>
              <label class="small muted">
                Deadline
                <input class="inline-input" type="date" bind:value={editDeadline} required />
              </label>
            </div>
          </div>
          <div class="panel-field">
            <label class="small muted" for="bt-edit-baseline">Persentase awal (opsional)</label>
            <input
              id="bt-edit-baseline"
              class="inline-input"
              type="number"
              min="0"
              max="100"
              placeholder="mis. 40"
              bind:value={editBaselinePct}
            />
            <span class="small muted">
              Progress yang sudah berjalan di lapangan sebelum dicatat di sistem (mis. migrasi data ke
              staging) — dihitung sebagai Daily Task "Baseline Awal" terpisah, tidak mengubah Daily Task
              lain. Kosongkan untuk menghapus baseline.
            </span>
          </div>
          {#if bigTaskEditError}<p class="small" style="color:var(--win-red);margin:0">{bigTaskEditError}</p>{/if}
          <div class="inline-form-actions" style="padding-top:4px">
            <button class="quick-btn quick-btn-done" type="submit" disabled={savingBigTaskEdit}>
              {savingBigTaskEdit ? 'Menyimpan...' : 'Simpan perubahan'}
            </button>
            <button class="quick-btn" type="button" on:click={() => (editingBigTask = false)}>Batal</button>
          </div>
        </form>
      </div>
    </div>
  </div>
{/if}

<!-- Modal komentar -->
{#if commentModal.open && activeBt}
  <CommentModal
    bigTaskId={activeBt.id}
    bigTaskName={activeBt.name}
    dailyTasks={dailyTasksForComments}
    jumpScope={commentModal.jumpScope}
    jumpToken={commentModal.jumpToken}
    on:close={() => (commentModal = { ...commentModal, open: false })}
  />
{/if}
