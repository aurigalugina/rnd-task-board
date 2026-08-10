<script lang="ts">
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { BigTask } from '$lib/types';
  import VerdictBadge from './VerdictBadge.svelte';
  import DualBar from './DualBar.svelte';
  import StatCard from './StatCard.svelte';
  import DailyTaskPanel from './DailyTaskPanel.svelte';
  import CommentSection from './CommentSection.svelte';
  import CheatSheetSection from './CheatSheetSection.svelte';

  export let boardId: string;

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
  let creating = false;
  let createError: string | null = null;

  $: isSpv = $auth.user?.roles.includes('spv') ?? false;

  async function load() {
    loading = true;
    try {
      bigTasks = await api.get<BigTask[]>(`/boards/${boardId}/big-tasks`);
      if (!selectedBtId || !bigTasks.some((b) => b.id === selectedBtId)) {
        selectedBtId = bigTasks[0]?.id ?? null;
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  $: if (boardId) load();

  $: activeBt = bigTasks.find((b) => b.id === selectedBtId) ?? null;
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
      await api.post(`/boards/${boardId}/big-tasks`, { name, start_date: startDate, deadline });
      name = '';
      startDate = '';
      deadline = '';
      showCreateForm = false;
      await load();
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
      await load();
    } catch (e) {
      actionError = (e as Error).message;
    }
  }

  async function undoSignOff(bt: BigTask) {
    actionError = '';
    try {
      await api.del(`/big-tasks/${bt.id}/sign-off`);
      await load();
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
            <input class="inline-input" type="date" bind:value={startDate} required />
            <input class="inline-input" type="date" bind:value={deadline} required />
          </div>
          {#if createError}<span class="small" style="color:var(--win-red)">{createError}</span>{/if}
          <div class="inline-form-actions">
            <button class="quick-btn quick-btn-done" type="submit" disabled={creating}>
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
        {#key activeBt.id}
          <DailyTaskPanel
            bigTaskId={activeBt.id}
            on:updated={load}
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
