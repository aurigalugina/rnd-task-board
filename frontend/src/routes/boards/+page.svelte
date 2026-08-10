<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import type { Board } from '$lib/types';
  import BigTaskList from '$lib/components/BigTaskList.svelte';

  let boards: Board[] = [];
  let loading = true;
  let error: string | null = null;
  let selectedBoardId: string | null = null;

  let showCreateForm = false;
  let name = '';
  let tag = '';
  let creating = false;
  let createError: string | null = null;

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

  function selectBoard(id: string) {
    selectedBoardId = id;
    goto(`/boards?board=${id}`, { replaceState: true, noScroll: true, keepFocus: true });
  }

  async function createBoard() {
    createError = null;
    creating = true;
    try {
      const board = await api.post<Board>('/boards', { name, tag });
      boards = [...boards, board];
      name = '';
      tag = '';
      showCreateForm = false;
      selectBoard(board.id);
    } catch (e) {
      createError = (e as Error).message;
    } finally {
      creating = false;
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
      {#if !showCreateForm}
        <button class="board-pill board-pill-ghost" on:click={() => (showCreateForm = true)}>+ Board baru</button>
      {/if}
    </div>
  </div>

  {#if showCreateForm}
    <form class="inline-form" on:submit|preventDefault={createBoard}>
      <input class="inline-input" style="width:180px" placeholder="Nama board" bind:value={name} required />
      <input class="inline-input" style="width:140px" placeholder="Tag" bind:value={tag} />
      <button class="quick-btn quick-btn-done" type="submit" disabled={creating}>{creating ? 'Menyimpan...' : 'Simpan'}</button>
      <button class="quick-btn" type="button" on:click={() => (showCreateForm = false)}>Batal</button>
    </form>
  {/if}
  {#if createError}<p class="small" style="color:var(--win-red)">{createError}</p>{/if}

  {#if selectedBoardId}
    <BigTaskList boardId={selectedBoardId} />
  {:else}
    <p class="empty-note">Belum ada board. Buat board pertama di atas.</p>
  {/if}
{/if}
