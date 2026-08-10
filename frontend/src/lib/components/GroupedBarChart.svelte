<script lang="ts">
  // Grouped bar chart custom SVG (actual vs expected) — bukan library, lihat
  // docs/decision-log/decision-log-design-system-adoption-20260809.md.
  export let data: { name: string; actual: number; expected: number }[];

  const vbW = 640;
  const vbH = 220;
  const padLeft = 30;
  const padBottom = 22;
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
  <!-- Key by index, BUKAN d.name — nama Big Task tidak dijamin unik (dua Big
       Task beda id bisa punya nama sama, apalagi setelah dipotong ke 12 char),
       keyed-each dengan key non-unik bikin Svelte crash "duplicate keys". -->
  {#each data as d, i (i)}
    <g transform="translate({padLeft + i * groupW + groupW / 2}, 0)">
      <rect x={-barW - 2} y={y(d.expected)} width={barW} height={padTop + plotH - y(d.expected)} fill="#AEB6BC" />
      <rect x={2} y={y(d.actual)} width={barW} height={padTop + plotH - y(d.actual)} fill="var(--win-blue)" />
      <text x="0" y={vbH - 6} text-anchor="middle" font-size="9" fill="var(--text-muted)">{d.name}</text>
    </g>
  {/each}
</svg>

<style>
  .bar-chart :global(text) { font-family: inherit; }
</style>
