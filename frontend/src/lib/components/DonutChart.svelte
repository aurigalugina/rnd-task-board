<script lang="ts">
  // Donut chart custom SVG (bukan library) — lihat
  // docs/decision-log/decision-log-design-system-adoption-20260809.md.
  // 2026-09-01: Reverted to balanced size (too large was overwhelming)
  export let data: { name: string; value: number; color: string }[];
  export let size = 130; // Back to smaller, cleaner size

  const strokeWidth = size * 0.23;
  $: r = size / 2 - strokeWidth / 2 - 2;
  $: circumference = 2 * Math.PI * r;
  $: total = data.reduce((s, d) => s + d.value, 0) || 1;
  $: segments = (() => {
    let offset = 0;
    return data.map((d, i) => {
      const length = (d.value / total) * circumference;
      const seg = { ...d, length, offset, index: i };
      offset += length;
      return seg;
    });
  })();
</script>

<div class="donut-wrapper">
  <svg 
    width="100%" 
    height="100%" 
    viewBox="0 0 {size} {size}" 
    role="img" 
    aria-label="Diagram donut"
    class="donut-chart"
    preserveAspectRatio="xMidYMid meet"
  >
    <g transform="rotate(-90 {size / 2} {size / 2})">
      {#each segments as seg (seg.index)}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke={seg.color}
          stroke-width={strokeWidth}
          stroke-dasharray="{seg.length} {circumference - seg.length}"
          stroke-dashoffset={-seg.offset}
          stroke-linecap="round"
          class="donut-segment"
          style="--segment-index: {seg.index}"
        />
      {/each}
    </g>
  </svg>
</div>

<style>
  .donut-wrapper {
    width: 100%;
    max-width: 160px;
    aspect-ratio: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .donut-chart {
    max-width: 100%;
    height: auto;
    filter: drop-shadow(0 2px 6px rgba(0, 0, 0, 0.08));
  }

  .donut-segment {
    animation: drawDonut 1.2s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
    animation-delay: calc(var(--segment-index) * 0.15s);
    transform-origin: center;
    transition: stroke-width 0.3s ease, filter 0.3s ease, opacity 0.3s ease;
  }

  .donut-segment:hover {
    stroke-width: 0;
    filter: brightness(1.15);
    opacity: 0.9;
  }

  @keyframes drawDonut {
    from {
      stroke-dasharray: 0 var(--circumference);
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @media (max-width: 1023px) {
    .donut-wrapper {
      max-width: 140px;
    }
  }

  @media (max-width: 767px) {
    .donut-wrapper {
      max-width: 120px;
    }
  }

  @media (max-width: 479px) {
    .donut-wrapper {
      max-width: 100px;
    }
  }
</style>

