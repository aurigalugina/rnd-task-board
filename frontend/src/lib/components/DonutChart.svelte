<script lang="ts">
  // Donut chart custom SVG (bukan library) — lihat
  // docs/decision-log/decision-log-design-system-adoption-20260809.md.
  export let data: { name: string; value: number; color: string }[];
  export let size = 130;

  const strokeWidth = size * 0.23;
  $: r = size / 2 - strokeWidth / 2 - 2;
  $: circumference = 2 * Math.PI * r;
  $: total = data.reduce((s, d) => s + d.value, 0) || 1;
  $: segments = (() => {
    let offset = 0;
    return data.map((d) => {
      const length = (d.value / total) * circumference;
      const seg = { ...d, length, offset };
      offset += length;
      return seg;
    });
  })();
</script>

<svg width={size} height={size} viewBox="0 0 {size} {size}" role="img" aria-label="Diagram donut">
  <g transform="rotate(-90 {size / 2} {size / 2})">
    <!-- Key by index — caller saat ini selalu kirim label kategori tetap
         (unik), tapi index tetap lebih aman kalau komponen ini dipakai ulang
         nanti dengan data yang name-nya bisa duplikat (lihat fix serupa di
         GroupedBarChart.svelte). -->
    {#each segments as seg, i (i)}
      <circle
        cx={size / 2}
        cy={size / 2}
        {r}
        fill="none"
        stroke={seg.color}
        stroke-width={strokeWidth}
        stroke-dasharray="{seg.length} {circumference - seg.length}"
        stroke-dashoffset={-seg.offset}
      />
    {/each}
  </g>
</svg>
