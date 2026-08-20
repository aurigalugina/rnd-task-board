<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { Board, BoardCategory, ReferensiTim } from '$lib/types';
  import BigTaskList from '$lib/components/BigTaskList.svelte';
  import { Archive, Pencil, X } from 'lucide-svelte';

  function handlePageKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (confirmingArchive && !archiving) confirmingArchive = false;
      else if (showCreateForm && !creating) showCreateForm = false;
      else if (editingBoard && !savingBoardEdit) editingBoard = false;
    }
  }

  let boards: Board[] = [];
  let loading = true;
  let error: string | null = null;
  let selectedBoardId: string | null = null;

  let showCreateForm = false;
  let name = '';
  let description = '';
  let category: BoardCategory = 'project';
  let creating = false;
  let createError: string | null = null;

  // Archiving board (super_user only) -- lihat decision-log-board-archive-20260812.md.
  $: isSuperUser = $auth.user?.access_level === 'super_user';
  let confirmingArchive = false;
  let archiving = false;
  let archiveError: string | null = null;

  // Edit board (deskripsi + kategori + tim, super_user only) -- lihat
  // decision-log-boards-dashboard-enhancements-20260820.md.
  let teams: ReferensiTim[] = [];
  let editingBoard = false;
  let editDescription = '';
  let editCategory: BoardCategory | '' = '';
  let editTeamIds: string[] = [];
  let savingBoardEdit = false;
  let boardEditError = '';

  onMount(async () => {
    try {
      const [boardList, timList] = await Promise.all([
        api.get<Board[]>('/boards'),
        api.get<ReferensiTim[]>('/referensi-tim')
      ]);
      boards = boardList;
      teams = timList;
      const fromUrl = $page.url.searchParams.get('board');
      selectedBoardId = fromUrl && boards.some((b) => b.id === fromUrl) ? fromUrl : (boards[0]?.id ?? null);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  });

  $: selectedBoard = boards.find((b) => b.id === selectedBoardId) ?? null;

  function openEditBoard() {
    if (!selectedBoard) return;
    editDescription = selectedBoard.description;
    editCategory = selectedBoard.category ?? '';
    editTeamIds = [...selectedBoard.team_ids];
    boardEditError = '';
    editingBoard = true;
  }
  function toggleEditTeam(teamId: string) {
    editTeamIds = editTeamIds.includes(teamId)
      ? editTeamIds.filter((id) => id !== teamId)
      : [...editTeamIds, teamId];
  }
  async function saveBoardEdit() {
    if (!selectedBoardId) return;
    savingBoardEdit = true;
    boardEditError = '';
    try {
      const updated = await api.patch<Board>(`/boards/${selectedBoardId}`, {
        description: editDescription,
        category: editCategory || null,
        team_ids: editTeamIds
      });
      boards = boards.map((b) => (b.id === updated.id ? updated : b));
      editingBoard = false;
    } catch (e) {
      boardEditError = (e as Error).message;
    } finally {
      savingBoardEdit = false;
    }
  }

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
      const board = await api.post<Board>('/boards', { name, description, category });
      boards = [...boards, board];
      name = '';
      description = '';
      category = 'project';
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
      <button class="quick-btn" on:click={openEditBoard}><Pencil size={13} />&nbsp;Edit board</button>
      <button class="quick-btn" on:click={() => (confirmingArchive = true)}><Archive size={13} />&nbsp;Archive board</button>
    {/if}
  </div>

  {#if createError}<p class="small" style="color:var(--win-red)">{createError}</p>{/if}

  {#if selectedBoard?.description}
    <div class="board-desc-banner">{selectedBoard.description}</div>
  {/if}

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
            <textarea id="board-desc" class="inline-input inline-textarea" placeholder="Deskripsi board (boleh beberapa baris)..." bind:value={description} />
          </div>
          <div class="panel-field">
            <label class="small muted" for="board-category">Kategori</label>
            <select id="board-category" class="inline-input" bind:value={category}>
              <option value="project">Project</option>
              <option value="routine">Routine</option>
            </select>
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

<!-- Modal edit board (deskripsi + kategori + tim, super_user only) -->
{#if editingBoard && selectedBoard}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="overlay overlay-center" on:click={() => !savingBoardEdit && (editingBoard = false)}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal-box" role="dialog" aria-modal="true" on:click|stopPropagation style="width:400px;max-width:calc(100vw - 32px)">
      <div class="de-modal-header">
        <span class="de-modal-title">Edit board — {selectedBoard.name}</span>
        <button class="de-close-btn icon-btn" aria-label="Tutup" on:click={() => (editingBoard = false)}><X size={12} /></button>
      </div>
      <div class="modal-body">
        <form on:submit|preventDefault={saveBoardEdit}>
          <div class="panel-field">
            <label class="small muted" for="board-edit-desc">Deskripsi</label>
            <textarea id="board-edit-desc" class="inline-input inline-textarea" placeholder="Deskripsi board (boleh beberapa baris)..." bind:value={editDescription} />
          </div>
          <div class="panel-field">
            <label class="small muted" for="board-edit-category">Kategori</label>
            <select id="board-edit-category" class="inline-input" bind:value={editCategory}>
              <option value="">— Belum dikategorikan —</option>
              <option value="project">Project</option>
              <option value="routine">Routine</option>
            </select>
          </div>
          <div class="panel-field">
            <span class="small muted">Tim (regular user cuma lihat board sesuai tim-nya)</span>
            <div class="reviewer-picker" style="margin-top:4px">
              {#each teams as t (t.id)}
                <label class="reviewer-picker-row small">
                  <input type="checkbox" checked={editTeamIds.includes(t.id)} on:change={() => toggleEditTeam(t.id)} />
                  {t.name}
                </label>
              {/each}
              {#if teams.length === 0}
                <span class="small muted">Belum ada tim di referensi_tim.</span>
              {/if}
            </div>
          </div>
          {#if boardEditError}<p class="small" style="color:var(--win-red);margin:0">{boardEditError}</p>{/if}
          <div class="inline-form-actions" style="padding-top:4px">
            <button class="quick-btn quick-btn-done" type="submit" disabled={savingBoardEdit}>
              {savingBoardEdit ? 'Menyimpan...' : 'Simpan perubahan'}
            </button>
            <button class="quick-btn" type="button" on:click={() => (editingBoard = false)}>Batal</button>
          </div>
        </form>
      </div>
    </div>
  </div>
{/if}
