<script lang="ts">
  export let expected: number;
  export let actual: number;
</script>

<div class="dualbar">
  <div class="dualbar-track">
    <div class="dualbar-fill {actual >= expected ? 'fill-good' : 'fill-bad'}" style="width:{Math.min(actual, 100)}%" />
    <div class="dualbar-tick" style="left:{Math.min(expected, 100)}%" />
  </div>
  <div class="dualbar-legend">
    <span class="mono">{actual}%</span>
    <span class="dualbar-legend-muted">target {expected}%</span>
  </div>
</div>
