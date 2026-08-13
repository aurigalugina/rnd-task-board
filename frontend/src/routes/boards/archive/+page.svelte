<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { ArchivedBoard } from '$lib/types';
  import { Archive, Undo2 } from 'lucide-svelte';

  // Board Archive -- cuma super_user (lihat decision-log-board-archive-20260812.md).
  // Board yang di-archive HILANG dari Dashboard & tab Boards (GET /boards
  // sudah memfilternya), tapi tetap muncul apa adanya di Weekly Plan/Review
  // Queue (query di situ tidak difilter archived_at sama sekali, sengaja).
  $: isSuperUser = $auth.user?.access_level === 'super_user';

  let boards: ArchivedBoard[] = [];
  let loading = true;
  let error: string | null = null;
  let unarchivingId: string | null = null;

  async function load() {
    loading = true;
    error = null;
    try {
      boards = await api.get<ArchivedBoard[]>('/boards/archive');
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    if (isSuperUser) load();
  });

  async function unarchive(board: ArchivedBoard) {
    unarchivingId = board.id;
    try {
      await api.patch(`/boards/${board.id}/unarchive`);
      boards = boards.filter((b) => b.id !== board.id);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      unarchivingId = null;
    }
  }

  function fmtDate(iso: string): string {
    return new Date(iso).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' });
  }
</script>

<div class="section">
  <div class="section-title">Board Archive</div>

  {#if !isSuperUser}
    <p class="small" style="color:var(--win-red)">Halaman ini cuma untuk super user.</p>
  {:else if loading}
    <p class="small muted">Memuat...</p>
  {:else if error}
    <p class="small" style="color:var(--win-red)">{error}</p>
  {:else}
    <div class="queue-list">
      {#each boards as board (board.id)}
        <div class="queue-row">
          <div class="queue-main">
            <div class="queue-title"><strong>{board.name}</strong></div>
            {#if board.description}<div class="muted small">{board.description}</div>{/if}
            <div class="muted small">Diarsipkan oleh {board.archived_by_name || '—'} · {fmtDate(board.archived_at)}</div>
          </div>
          <button class="approve-btn" on:click={() => unarchive(board)} disabled={unarchivingId === board.id}>
            <Undo2 size={13} />&nbsp;{unarchivingId === board.id ? 'Memproses...' : 'Unarchive'}
          </button>
        </div>
      {/each}
      {#if boards.length === 0}
        <div class="empty-state">
          <div class="empty-state-icon"><Archive size={36} /></div>
          <div class="empty-state-text">Belum ada board yang diarsipkan.</div>
        </div>
      {/if}
    </div>
  {/if}
</div>
