<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { AssignableUser, BigTask } from '$lib/types';
  import VerdictBadge from './VerdictBadge.svelte';
  import DualBar from './DualBar.svelte';
  import StatCard from './StatCard.svelte';
  import Avatar from './Avatar.svelte';
  import DailyTaskPanel from './DailyTaskPanel.svelte';
  import CommentSection from './CommentSection.svelte';
  import CheatSheetSection from './CheatSheetSection.svelte';

  export let boardId: string;

  let assignableUsers: AssignableUser[] = [];
  $: userById = Object.fromEntries(assignableUsers.map((u) => [u.id, u]));

  onMount(async () => {
    assignableUsers = await api.get<AssignableUser[]>('/users/assignable');
  });

  let dailyTasksForComments: { id: string; title: string }[] = [];
  let jumpScope: string | null = null;
  let jumpToken = 0;

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
  let creating = false;
  let createError: string | null = null;

  function toggleMember(userId: string) {
    memberUserIds = memberUserIds.includes(userId)
      ? memberUserIds.filter((id) => id !== userId)
      : [...memberUserIds, userId];
  }

  $: isSpv = $auth.user?.roles.includes('spv') ?? false;

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
      await api.post(`/boards/${boardId}/big-tasks`, {
        name,
        start_date: startDate,
        deadline,
        member_user_ids: memberUserIds
      });
      name = '';
      startDate = '';
      deadline = '';
      memberUserIds = [];
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
      await api.post(`/big-tasks/${bt.id}/sign-off`);
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
            <VerdictBadge verdict={bt.verdict} />
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

      {#if showCreateForm}
        <form class="inline-form inline-form-daily" on:submit|preventDefault={createBigTask}>
          <input class="inline-input" placeholder="Nama big task" bind:value={name} required />
          <div class="inline-form-dates">
            <input class="inline-input" type="date" bind:value={startDate} min={minDate} required />
            <input class="inline-input" type="date" bind:value={deadline} min={minDate} required />
          </div>
          <div class="reviewer-picker">
            <span class="small muted">Anggota — siapa saja yang terlibat (wajib min. 2)</span>
            {#each assignableUsers as u (u.id)}
              <label class="reviewer-picker-row small">
                <input
                  type="checkbox"
                  checked={memberUserIds.includes(u.id)}
                  on:change={() => toggleMember(u.id)}
                />
                {u.display_name} <span class="muted">— {u.roles.join('/')}</span>
              </label>
            {/each}
          </div>
          {#if memberUserIds.length < 2}<span class="small muted">Pilih minimal 2 anggota.</span>{/if}
          {#if createError}<span class="small" style="color:var(--win-red)">{createError}</span>{/if}
          <div class="inline-form-actions">
            <button class="quick-btn quick-btn-done" type="submit" disabled={creating || memberUserIds.length < 2}>
              {creating ? 'Menyimpan...' : 'Simpan'}
            </button>
            <button class="quick-btn" type="button" on:click={() => (showCreateForm = false)}>Batal</button>
          </div>
        </form>
      {:else}
        <button class="add-card-ghost" on:click={() => (showCreateForm = true)}>+ Tambah big task</button>
      {/if}
    </div>

    <div class="bigtask-detail">
      {#if activeBt}
        <div class="section-title bigtask-detail-title">
          <span>{activeBt.name} — daily task</span>
          <div class="bigtask-detail-title-right">
            <VerdictBadge verdict={activeBt.verdict} />
            {#if isSpv}
              {#if activeBt.signed}
                <button class="sign-btn sign-btn-active" on:click={() => undoSignOff(activeBt)}>✓ Signed done</button>
              {:else}
                <button
                  class="sign-btn"
                  disabled={activeBt.actual_pct < 100}
                  title={activeBt.actual_pct < 100 ? 'Progress belum 100%' : ''}
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
            on:jumpToComment={(e) => { jumpScope = e.detail; jumpToken += 1; }}
          />
          <CommentSection bigTaskId={activeBt.id} dailyTasks={dailyTasksForComments} {jumpScope} {jumpToken} />
        {/key}
      {:else}
        <div class="empty-note">Pilih atau tambahkan big task dulu.</div>
      {/if}
    </div>
  </div>
{/if}
