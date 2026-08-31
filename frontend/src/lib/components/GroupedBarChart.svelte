<script lang="ts">
  import { goto } from '$app/navigation';

  // Grouped bar chart custom SVG (actual vs expected) — bukan library, lihat
  // docs/decision-log/decision-log-design-system-adoption-20260809.md.
  //
  // Label nama board di bawah tiap bar (truncateName 12 char fixed) tabrakan
  // begitu board makin banyak (slot per-bar makin sempit, dikeluhkan user).
  // Diganti pola sama seperti DonutChart: warna identitas board (dot kecil di
  // bawah bar) + legend terpisah (nama boleh wrap normal, gak dibatasi lebar
  // kolom bar) -- warna bar sendiri (abu-abu=expected, biru=actual) TETAP,
  // dot cuma buat identitas board bukan gantiin makna warna bar. Lihat
  // decision-log-boards-dashboard-enhancements-20260820.md.
  export let data: { id: string; name: string; actual: number; expected: number; color: string }[];

  // Shortcut klik dari dot/legend langsung ke tab board itu di halaman Boards
  // (2026-08-20, permintaan user) -- pola sama seperti .board-link di app.css.
  function goToBoard(id: string) {
    goto(`/boards?board=${id}`);
  }

  const vbW = 640;
  const vbH = 180;
  const padLeft = 35;
  const padBottom = 14;
  const padTop = 8;
  const padRight = 8;
  const plotW = vbW - padLeft - padRight;
  const plotH = vbH - padTop - padBottom;
  const ticks = [0, 25, 50, 75, 100];

  $: groupW = data.length ? plotW / data.length : plotW;
  $: barW = Math.min(12, groupW / 3);

  function y(v: number) {
    return padTop + plotH - (Math.min(v, 100) / 100) * plotH;
  }
</script>

<svg viewBox="0 0 {vbW} {vbH}" width="100%" height="auto" class="bar-chart" role="img" aria-label="Grafik actual vs expected" style="min-height: 160px">
  {#each ticks as tick}
    <line x1={padLeft} y1={y(tick)} x2={vbW - padRight} y2={y(tick)} stroke="#D6DADD" stroke-width="0.5" opacity="0.4" />
    <text x={padLeft - 3} y={y(tick) + 2} text-anchor="end" font-size="8" fill="var(--text-muted)">{tick}</text>
  {/each}
  {#each data as d, i (i)}
    <g transform="translate({padLeft + i * groupW + groupW / 2}, 0)">
      <rect 
        x={-barW - 2} 
        y={y(d.expected)} 
        width={barW} 
        height={padTop + plotH - y(d.expected)}
        fill="#AEB6BC" 
        rx="2"
        class="bar-expected"
        style="--bar-index: {i}"
      />
      <rect 
        x={2} 
        y={y(d.actual)} 
        width={barW} 
        height={padTop + plotH - y(d.actual)}
        fill="var(--win-blue)" 
        rx="2"
        class="bar-actual"
        style="--bar-index: {i}"
      />
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <circle
        cx="0"
        cy={vbH - padBottom + 6}
        r="4.5"
        fill={d.color}
        class="board-dot"
        on:click={() => goToBoard(d.id)}
      >
        <title>{d.name}</title>
      </circle>
    </g>
  {/each}
</svg>

<div class="chart-legend chart-legend-flow">
  {#each data as d (d.id)}
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="chart-legend-row board-link-row" on:click={() => goToBoard(d.id)}>
      <span class="chart-legend-swatch" style="background:{d.color}" />
      <span class="small">{d.name}</span>
    </div>
  {/each}
</div>

<style>
  .bar-chart :global(text) { font-family: inherit; }
  
  .bar-chart {
    filter: drop-shadow(0 1px 3px rgba(0, 0, 0, 0.06));
  }
  
  .bar-expected,
  .bar-actual {
    transition: fill 0.2s ease, opacity 0.2s ease;
  }
  
  .bar-expected:hover,
  .bar-actual:hover {
    opacity: 0.85;
  }
  
  .board-dot { 
    cursor: pointer;
    transition: r 0.2s ease;
  }
  
  .board-dot:hover {
    r: 5;
  }
  
  @media (max-width: 767px) {
    .bar-chart {
      height: 150px;
    }
  }
</style>
