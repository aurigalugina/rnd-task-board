<script lang="ts">
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

  const vbW = 640;
  const vbH = 200;
  const padLeft = 30;
  const padBottom = 14;
  const padTop = 10;
  const padRight = 10;
  const plotW = vbW - padLeft - padRight;
  const plotH = vbH - padTop - padBottom;
  const ticks = [0, 25, 50, 75, 100];

  $: groupW = data.length ? plotW / data.length : plotW;
  $: barW = Math.min(16, groupW / 3);

  function y(v: number) {
    return padTop + plotH - (Math.min(v, 100) / 100) * plotH;
  }
</script>

<svg viewBox="0 0 {vbW} {vbH}" width="100%" height={vbH} class="bar-chart" role="img" aria-label="Grafik actual vs expected">
  {#each ticks as tick}
    <line x1={padLeft} y1={y(tick)} x2={vbW - padRight} y2={y(tick)} stroke="#D6DADD" />
    <text x={padLeft - 4} y={y(tick) + 3} text-anchor="end" font-size="9" fill="var(--text-muted)">{tick}</text>
  {/each}
  <!-- Key by index, BUKAN d.id -- gak masalah walau id unik, index cukup buat
       chart visual murni (lihat gotcha lama soal keyed-each di CLAUDE.md). -->
  {#each data as d, i (i)}
    <g transform="translate({padLeft + i * groupW + groupW / 2}, 0)">
      <rect x={-barW - 2} y={y(d.expected)} width={barW} height={padTop + plotH - y(d.expected)} fill="#AEB6BC" />
      <rect x={2} y={y(d.actual)} width={barW} height={padTop + plotH - y(d.actual)} fill="var(--win-blue)" />
      <circle cx="0" cy={vbH - padBottom + 6} r="4" fill={d.color}>
        <title>{d.name}</title>
      </circle>
    </g>
  {/each}
</svg>

<div class="chart-legend chart-legend-flow">
  {#each data as d (d.id)}
    <div class="chart-legend-row">
      <span class="chart-legend-swatch" style="background:{d.color}" />
      <span class="small">{d.name}</span>
    </div>
  {/each}
</div>

<style>
  .bar-chart :global(text) { font-family: inherit; }
</style>
