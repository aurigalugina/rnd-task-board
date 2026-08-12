<script lang="ts">
  // Toggle switch modern (bukan tombol retro) -- dipakai buat boolean state
  // yang jelas cuma dua nilai (mis. Day Entry selesai/belum). Tema tetap
  // ikut token warna retro (--win-green dkk), tapi interaksinya pola modern
  // (switch), bukan tombol teks yang gonta-ganti label. Lihat CLAUDE.md
  // bagian Design System -- retro cuma visual, bukan wajib meniru semua
  // widget lawas kalau ada pola modern yang lebih jelas.
  export let checked: boolean;
  export let disabled = false;
  export let onLabel = 'Selesai';
  export let offLabel = 'Belum';
  export let onToggle: (next: boolean) => void = () => {};
</script>

<button
  type="button"
  class="toggle-switch"
  class:toggle-on={checked}
  role="switch"
  aria-checked={checked}
  aria-label={checked ? onLabel : offLabel}
  {disabled}
  on:click={() => onToggle(!checked)}
>
  <span class="toggle-track">
    <span class="toggle-knob" />
  </span>
  <span class="toggle-text small">{checked ? onLabel : offLabel}</span>
</button>
