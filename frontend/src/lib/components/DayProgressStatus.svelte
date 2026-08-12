<script lang="ts">
  // Gantiin Toggle.svelte buat Day Entry (2026-08-10) -- status Day Entry
  // sekarang 3 state (Belum/On Progress/Selesai) diturunkan dari progress_pct
  // (0-100), bukan boolean. Lihat
  // docs/decision-log/decision-log-day-entry-progress-pct-20260810.md.
  import { clampOnProgressPct, progressPctForStatus, statusFromProgressPct, type DayProgressStatus } from '$lib/dayProgress';

  export let progressPct: number;
  export let onChange: (nextPct: number) => void;
  export let disabled = false;

  $: status = statusFromProgressPct(progressPct);

  function handleStatusChange(e: Event) {
    const next = (e.currentTarget as HTMLSelectElement).value as DayProgressStatus;
    onChange(progressPctForStatus(next, progressPct));
  }

  function handlePctInput(e: Event) {
    const raw = Number((e.currentTarget as HTMLInputElement).value);
    onChange(clampOnProgressPct(raw));
  }

  const labels: Record<DayProgressStatus, string> = {
    belum: 'Belum',
    on_progress: 'On Progress',
    selesai: 'Selesai'
  };
</script>

<div class="day-progress">
  <select class="day-progress-select status-{status}" value={status} on:change={handleStatusChange} {disabled}>
    <option value="belum">{labels.belum}</option>
    <option value="on_progress">{labels.on_progress}</option>
    <option value="selesai">{labels.selesai}</option>
  </select>
  {#if status === 'on_progress'}
    <input
      class="day-progress-pct-input"
      type="number"
      min="1"
      max="99"
      value={progressPct}
      on:change={handlePctInput}
      aria-label="Persentase progress"
      {disabled}
    />
    <span class="small muted">%</span>
  {/if}
</div>
