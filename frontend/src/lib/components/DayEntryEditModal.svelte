<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { api } from '$lib/api/client';
  import type { DayEntry } from '$lib/types';
  import { X, Trash2 } from 'lucide-svelte';

  export let dailyTaskId: string;
  export let entry: DayEntry | null = null;  // null = mode tambah baru
  export let prefillDate = '';               // dipakai saat entry = null
  export let minDate = '';
  export let isPast = false;                 // entri lampau → read-only

  const dispatch = createEventDispatcher<{ saved: void; deleted: void; close: void }>();

  let entryDate = entry?.entry_date ?? prefillDate;
  let plannedText = entry?.planned_text ?? '';
  let blockerText = entry?.blocker_text ?? '';
  let progressPct = entry?.progress_pct ?? 0;
  let saving = false;
  let confirmDelete = false;
  let deleting = false;
  let errorMsg = '';

  $: state = progressPct === 0 ? 'belum' : progressPct === 100 ? 'done' : 'onprogress';

  function setProgress(s: 'belum' | 'onprogress' | 'done') {
    if (s === 'belum') progressPct = 0;
    else if (s === 'done') progressPct = 100;
    else if (progressPct === 0 || progressPct === 100) progressPct = 50;
  }

  async function save() {
    saving = true;
    errorMsg = '';
    try {
      if (entry) {
        await api.patch(`/day-entries/${entry.id}`, {
          planned_text: plannedText,
          progress_pct: progressPct,
          blocker_text: progressPct === 100 ? '' : blockerText,
        });
      } else {
        await api.post(`/daily-tasks/${dailyTaskId}/day-entries`, {
          entry_date: entryDate,
          planned_text: plannedText || undefined,
        });
      }
      dispatch('saved');
    } catch (e) {
      errorMsg = (e as Error).message;
    } finally {
      saving = false;
    }
  }

  async function deleteEntry() {
    if (!entry) return;
    deleting = true;
    errorMsg = '';
    try {
      await api.del(`/day-entries/${entry.id}`);
      dispatch('deleted');
    } catch (e) {
      errorMsg = (e as Error).message;
    } finally {
      deleting = false;
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') dispatch('close');
  }
</script>

<svelte:window on:keydown={onKeydown} />
<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="overlay overlay-center" on:click={() => dispatch('close')}>
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div class="modal-box day-entry-modal" role="dialog" on:click|stopPropagation>
    <div class="de-modal-header">
      <span class="de-modal-title">
        {entry ? 'Edit — ' + entry.entry_date : 'Tambah day entry'}
        {#if isPast}<span class="de-modal-badge-past">lampau · hanya baca</span>{/if}
      </span>
      <button class="icon-btn de-close-btn" on:click={() => dispatch('close')}><X size={12} /></button>
    </div>

    <div class="modal-body">
      {#if !entry}
        <div class="panel-field">
          <label class="small muted" for="de-date">Tanggal</label>
          <input id="de-date" class="inline-input" type="date" bind:value={entryDate} min={minDate} required />
        </div>
      {/if}

      <div class="panel-field">
        <label class="small muted" for="de-planned">Rencana / deskripsi task</label>
        <textarea id="de-planned" class="inline-input inline-textarea" rows="3"
          placeholder="Tulis rencana untuk hari ini..."
          bind:value={plannedText}
          disabled={isPast} />
      </div>

      {#if entry}
        <div class="panel-field">
          <span class="small muted">Status progress</span>
          <div class="dep-group" role="group" aria-label="Status progress">
            <button type="button" class="dep-btn {state === 'belum' ? 'dep-active dep-belum' : ''}"
              disabled={isPast} on:click={() => setProgress('belum')}>Belum</button>
            <button type="button" class="dep-btn {state === 'onprogress' ? 'dep-active dep-onprogress' : ''}"
              disabled={isPast} on:click={() => setProgress('onprogress')}>On Progress</button>
            <button type="button" class="dep-btn {state === 'done' ? 'dep-active dep-done' : ''}"
              disabled={isPast} on:click={() => setProgress('done')}>Selesai</button>
          </div>
          {#if state === 'onprogress'}
            <div class="dep-pct-row">
              <label class="small muted" for="de-pct">Persentase:</label>
              <input id="de-pct" class="inline-input" type="number" min="1" max="99"
                bind:value={progressPct} disabled={isPast} style="width:70px" />
              <span class="small muted">%</span>
            </div>
          {/if}
        </div>

        {#if state !== 'done'}
          <div class="panel-field">
            <label class="small muted" for="de-blocker">Blocker / catatan lanjutan</label>
            <textarea id="de-blocker" class="inline-input inline-textarea" rows="2"
              placeholder="Ada hambatan? Rencana hari berikutnya?"
              bind:value={blockerText}
              disabled={isPast} />
          </div>
        {/if}
      {/if}

      {#if errorMsg}<p class="small" style="color:var(--win-red);margin:0">{errorMsg}</p>{/if}

      <div class="de-modal-actions">
        <div class="inline-form-actions">
          {#if !isPast}
            <button class="quick-btn quick-btn-done" on:click={save} disabled={saving}>
              {saving ? 'Menyimpan...' : 'Simpan'}
            </button>
          {/if}
          <button class="quick-btn" on:click={() => dispatch('close')}>
            {isPast ? 'Tutup' : 'Batal'}
          </button>
        </div>

        {#if entry && !isPast}
          <div class="inline-form-actions">
            {#if !confirmDelete}
              <button class="quick-btn de-delete-btn" on:click={() => (confirmDelete = true)}>
                <Trash2 size={11} />&nbsp;Hapus entri
              </button>
            {:else}
              <span class="small" style="color:var(--win-red)">Yakin hapus?</span>
              <button class="quick-btn de-delete-btn" style="font-weight:bold"
                on:click={deleteEntry} disabled={deleting}>
                {deleting ? 'Menghapus...' : 'Ya, hapus'}
              </button>
              <button class="quick-btn" on:click={() => (confirmDelete = false)}>Batal</button>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>
