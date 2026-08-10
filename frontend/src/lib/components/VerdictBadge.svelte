<script lang="ts">
  import type { Verdict } from '$lib/types';

  export let verdict: Verdict;

  const label: Record<Verdict, string> = {
    on_progress: 'Berjalan',
    win: 'Win',
    lose: 'Lose'
  };
</script>

<span class="badge badge-{verdict}">{label[verdict]}</span>
