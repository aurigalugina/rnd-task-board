<script lang="ts">
  export let label: string;
  export let value: string | number;
  export let tone: 'warn' | 'good' | 'accent' | undefined = undefined;
</script>

<div class="stat-card {tone ? `tone-${tone}` : ''}">
  <div class="stat-label">{label}</div>
  <div class="stat-value mono">{value}</div>
</div>
