<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { X } from 'lucide-svelte';
  import CommentSection from './CommentSection.svelte';

  export let bigTaskId: string;
  export let bigTaskName: string;
  export let dailyTasks: { id: string; title: string }[] = [];
  export let jumpScope: string | null = null;
  export let jumpToken = 0;

  const dispatch = createEventDispatcher<{ close: void }>();

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') dispatch('close');
  }
</script>

<svelte:window on:keydown={onKeydown} />
<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="overlay overlay-center" on:click={() => dispatch('close')}>
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div class="modal-box comment-modal" role="dialog" on:click|stopPropagation>
    <div class="panel-header">
      <span style="font-size:11px;font-weight:bold">Komentar — {bigTaskName}</span>
      <button
        class="icon-btn"
        style="color:var(--titlebar-text);background:transparent;border-color:transparent"
        aria-label="Tutup"
        on:click={() => dispatch('close')}
      ><X size={12} /></button>
    </div>
    <div class="modal-body comment-modal-body">
      <CommentSection {bigTaskId} {dailyTasks} {jumpScope} {jumpToken} />
    </div>
  </div>
</div>
