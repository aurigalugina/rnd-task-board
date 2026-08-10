<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';

  const dispatch = createEventDispatcher<{ close: void }>();

  // `let` biasa (init sekali) — JANGAN `$: displayName = $auth.user?.display_name`,
  // lihat gotcha di CLAUDE.md (reactive assignment vs bind:value saling timpa).
  let displayName = $auth.user?.display_name ?? '';
  let initials = $auth.user?.initials ?? '';
  let currentPassword = '';
  let newPassword = '';
  let confirmPassword = '';
  let saving = false;
  let error: string | null = null;

  async function save() {
    error = null;
    if (newPassword && newPassword !== confirmPassword) {
      error = 'Konfirmasi password tidak sama.';
      return;
    }
    saving = true;
    try {
      const patch: Record<string, string> = { display_name: displayName, initials };
      if (newPassword) {
        if (!currentPassword) {
          error = 'Password saat ini wajib diisi untuk ganti password.';
          saving = false;
          return;
        }
        patch.current_password = currentPassword;
        patch.password = newPassword;
      }
      await api.patch('/users/me', patch);
      await auth.refreshUser();
      dispatch('close');
    } catch (e) {
      error = (e as Error).message;
    } finally {
      saving = false;
    }
  }
</script>

<svelte:window on:keydown={(e) => e.key === 'Escape' && dispatch('close')} />

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="overlay overlay-center" on:click={() => dispatch('close')}>
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div class="modal-box" role="dialog" aria-modal="true" aria-label="My Profile" on:click|stopPropagation>
    <div class="panel-header">
      <span class="titlebar-title" style="font-size:11px">My Profile</span>
      <button class="icon-btn" on:click={() => dispatch('close')} aria-label="Tutup">✕</button>
    </div>
    <div class="modal-body">
      <div class="panel-field">
        <div class="muted small">Nama</div>
        <input class="inline-input" bind:value={displayName} />
      </div>
      <div class="panel-field">
        <div class="muted small">Inisial avatar</div>
        <input class="inline-input" maxlength="2" bind:value={initials} />
      </div>
      <div class="panel-field">
        <div class="muted small">Password saat ini</div>
        <input type="password" class="inline-input" bind:value={currentPassword} placeholder="Diisi kalau mau ganti password" />
      </div>
      <div class="panel-field">
        <div class="muted small">Password baru</div>
        <input type="password" class="inline-input" bind:value={newPassword} placeholder="••••••••" />
      </div>
      <div class="panel-field">
        <div class="muted small">Konfirmasi password</div>
        <input type="password" class="inline-input" bind:value={confirmPassword} placeholder="••••••••" />
      </div>
      {#if error}<div class="small" style="color:var(--win-red)">{error}</div>{/if}
      <button class="quick-btn quick-btn-done" on:click={save} disabled={saving}>
        {saving ? 'Menyimpan...' : 'Simpan perubahan'}
      </button>
    </div>
  </div>
</div>
