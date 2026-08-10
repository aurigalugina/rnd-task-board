<script lang="ts">
  import { onMount } from 'svelte';
  import { reviewQueue, loadReviewQueue, markItemReviewed, type QueueItem } from '$lib/stores/reviewQueueStore';

  let loading = true;
  let error: string | null = null;
  let markingId: string | null = null;

  onMount(async () => {
    loading = true;
    try {
      await loadReviewQueue();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  });

  async function markReviewed(item: QueueItem) {
    markingId = item.id;
    try {
      await markItemReviewed(item);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      markingId = null;
    }
  }
</script>

<div class="section">
  <div class="section-title">
    Antrean review <span class="muted small">— {$reviewQueue.length} task menunggu atensi lo</span>
  </div>

  {#if loading}
    <p class="small muted">Memuat...</p>
  {:else if error}
    <p class="small" style="color:var(--win-red)">{error}</p>
  {:else}
    <div class="queue-list">
      {#each $reviewQueue as item (item.id)}
        <div class="queue-row">
          <a class="queue-main" href="/boards?board={item.board_id}">
            <div class="queue-title">{item.title}</div>
            <div class="muted small">{item.board_name} / {item.big_task_name}</div>
          </a>
          <button class="approve-btn" on:click={() => markReviewed(item)} disabled={markingId === item.id}>
            ✓ {markingId === item.id ? 'Menyimpan...' : 'Sudah gue lihat'}
          </button>
        </div>
      {/each}
      {#if $reviewQueue.length === 0}
        <div class="empty-note">Semua task udah lo review. Rapi.</div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .queue-main { text-decoration: none; color: inherit; }
</style>
