<script lang="ts">
  export let initials: string;
  export let size = 26;
  export let title = '';
</script>

<div class="avatar" style="width:{size}px;height:{size}px;font-size:{size * 0.4}px" {title}>
  {initials}
</div>
