<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { Board } from '$lib/types';
  import BigTaskList from '$lib/components/BigTaskList.svelte';
  import { Archive, X } from 'lucide-svelte';

  function handlePageKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (confirmingArchive && !archiving) confirmingArchive = false;
      else if (showCreateForm && !creating) showCreateForm = false;
    }
  }

  let boards: Board[] = [];
  let loading = true;
  let error: string | null = null;
  let selectedBoardId: string | null = null;

  let showCreateForm = false;
  let name = '';
  let description = '';
  let creating = false;
  let createError: string | null = null;

  // Archiving board (super_user only) -- lihat decision-log-board-archive-20260812.md.
  $: isSuperUser = $auth.user?.access_level === 'super_user';
  let confirmingArchive = false;
  let archiving = false;
  let archiveError: string | null = null;

  onMount(async () => {
    try {
      boards = await api.get<Board[]>('/boards');
      const fromUrl = $page.url.searchParams.get('board');
      selectedBoardId = fromUrl && boards.some((b) => b.id === fromUrl) ? fromUrl : (boards[0]?.id ?? null);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  });

  function selectBoard(id: string | null) {
    selectedBoardId = id;
    confirmingArchive = false;
    archiveError = null;
    goto(id ? `/boards?board=${id}` : '/boards', { replaceState: true, noScroll: true, keepFocus: true });
  }

  async function createBoard() {
    createError = null;
    creating = true;
    try {
      const board = await api.post<Board>('/boards', { name, description });
      boards = [...boards, board];
      name = '';
      description = '';
      showCreateForm = false;
      selectBoard(board.id);
    } catch (e) {
      createError = (e as Error).message;
    } finally {
      creating = false;
    }
  }

  function handleArchiveModalKeydown(_e: KeyboardEvent) { /* handled by handlePageKeydown */ }

  async function archiveBoard() {
    if (!selectedBoardId) return;
    archiveError = null;
    archiving = true;
    try {
      await api.patch(`/boards/${selectedBoardId}/archive`);
      boards = boards.filter((b) => b.id !== selectedBoardId);
      confirmingArchive = false;
      selectBoard(boards[0]?.id ?? null);
    } catch (e) {
      archiveError = (e as Error).message;
    } finally {
      archiving = false;
    }
  }
</script>

{#if loading}
  <p>Memuat...</p>
{:else if error}
  <p class="small" style="color:var(--win-red)">Gagal memuat data: {error}. Pastikan backend berjalan di :8080.</p>
{:else}
  <div class="board-pills-row">
    <div class="board-pills">
      {#each boards as board (board.id)}
        <button
          class="board-pill {board.id === selectedBoardId ? 'board-pill-active' : ''}"
          on:click={() => selectBoard(board.id)}
        >
          {board.name}
        </button>
      {/each}
      <button class="board-pill board-pill-ghost" on:click={() => (showCreateForm = true)}>+ Board baru</button>
    </div>
    {#if isSuperUser && selectedBoardId}
      <button class="quick-btn" on:click={() => (confirmingArchive = true)}><Archive size={13} />&nbsp;Archive board</button>
    {/if}
  </div>

  {#if createError}<p class="small" style="color:var(--win-red)">{createError}</p>{/if}

  {#if selectedBoardId}
    <BigTaskList boardId={selectedBoardId} />
  {:else}
    <p class="empty-note">Belum ada board. Buat board pertama di atas.</p>
  {/if}
{/if}

<svelte:window on:keydown={handlePageKeydown} />

<!-- Modal buat board baru -->
{#if showCreateForm}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="overlay overlay-center" on:click={() => !creating && (showCreateForm = false)}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal-box" role="dialog" aria-modal="true" on:click|stopPropagation style="width:400px;max-width:calc(100vw - 32px)">
      <div class="de-modal-header">
        <span class="de-modal-title">Board baru</span>
        <button class="de-close-btn icon-btn" aria-label="Tutup" on:click={() => (showCreateForm = false)}><X size={12} /></button>
      </div>
      <div class="modal-body">
        <form on:submit|preventDefault={createBoard}>
          <div class="panel-field">
            <label class="small muted" for="board-name">Nama board</label>
            <input id="board-name" class="inline-input" placeholder="Nama board" bind:value={name} required />
          </div>
          <div class="panel-field">
            <label class="small muted" for="board-desc">Deskripsi</label>
            <input id="board-desc" class="inline-input" placeholder="Deskripsi singkat" bind:value={description} />
          </div>
          {#if createError}<p class="small" style="color:var(--win-red);margin:0">{createError}</p>{/if}
          <div class="inline-form-actions" style="padding-top:4px">
            <button class="quick-btn quick-btn-done" type="submit" disabled={creating}>
              {creating ? 'Menyimpan...' : 'Buat board'}
            </button>
            <button class="quick-btn" type="button" on:click={() => (showCreateForm = false)}>Batal</button>
          </div>
        </form>
      </div>
    </div>
  </div>
{/if}

{#if confirmingArchive}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="overlay overlay-center" on:click={() => !archiving && (confirmingArchive = false)}>
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal-box" role="dialog" aria-modal="true" aria-label="Archive board" on:click|stopPropagation>
      <div class="panel-header">
        <span class="titlebar-title" style="font-size:11px">Archive board</span>
        <button class="icon-btn" on:click={() => (confirmingArchive = false)} aria-label="Tutup" disabled={archiving}><X size={13} /></button>
      </div>
      <div class="modal-body">
        <p class="small">
          Yakin archive board <strong>"{boards.find((b) => b.id === selectedBoardId)?.name}"</strong>? Board akan
          hilang dari Dashboard &amp; daftar ini, tapi tetap muncul di Weekly Plan/Review Queue.
        </p>
        {#if archiveError}<p class="small" style="color:var(--win-red)">{archiveError}</p>{/if}
        <div class="inline-form-actions">
          <button class="quick-btn quick-btn-done" disabled={archiving} on:click={archiveBoard}>
            {archiving ? 'Mengarsipkan...' : 'Ya, archive'}
          </button>
          <button class="quick-btn" type="button" disabled={archiving} on:click={() => (confirmingArchive = false)}>
            Batal
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
