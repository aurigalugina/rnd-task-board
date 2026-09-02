<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { X, User, Lock, Check, Pencil } from 'lucide-svelte';
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
  let savedFlash = false;

  // Password section collapsed by default -- ganti nama itu aksi paling
  // umum di sini (2026-09-01, permintaan user), tidak perlu 3 field password
  // selalu kelihatan tiap buka modal ini.
  let showPasswordSection = false;

  $: previewInitials = (initials || displayName.slice(0, 2) || '??').toUpperCase().slice(0, 2);

  async function save() {
    error = null;
    if (showPasswordSection && newPassword && newPassword !== confirmPassword) {
      error = 'Konfirmasi password tidak sama.';
      return;
    }
    saving = true;
    try {
      const patch: Record<string, string> = { display_name: displayName, initials };
      if (showPasswordSection && newPassword) {
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
      currentPassword = '';
      newPassword = '';
      confirmPassword = '';
      showPasswordSection = false;
      savedFlash = true;
      setTimeout(() => (savedFlash = false), 1800);
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
  <div class="profile-card" role="dialog" aria-modal="true" aria-label="My Profile" on:click|stopPropagation>
    <button class="profile-card-close" on:click={() => dispatch('close')} aria-label="Tutup"><X size={16} /></button>

    <div class="profile-card-hero">
      <div class="profile-card-avatar">{previewInitials}</div>
      <div class="profile-card-hero-text">
        <div class="profile-card-name">{$auth.user?.display_name ?? ''}</div>
        <div class="profile-card-role">{$auth.user?.roles.join(' · ') ?? ''}</div>
      </div>
    </div>

    <div class="profile-card-body">
      <div class="profile-field-group">
        <label class="profile-label" for="pf-name"><User size={13} />&nbsp;Nama</label>
        <input id="pf-name" class="profile-input" bind:value={displayName} placeholder="Nama lengkap" />
      </div>

      <div class="profile-field-group">
        <label class="profile-label" for="pf-initials">Inisial avatar</label>
        <input id="pf-initials" class="profile-input profile-input-short" maxlength="2" bind:value={initials} placeholder="AB" />
      </div>

      <button
        type="button"
        class="profile-password-toggle"
        on:click={() => (showPasswordSection = !showPasswordSection)}
      >
        <Lock size={13} />
        <span>Ganti password</span>
        <Pencil size={12} class="profile-password-toggle-icon" />
      </button>

      {#if showPasswordSection}
        <div class="profile-password-section">
          <div class="profile-field-group">
            <label class="profile-label" for="pf-current">Password saat ini</label>
            <input id="pf-current" type="password" class="profile-input" bind:value={currentPassword} placeholder="Wajib diisi untuk ganti password" />
          </div>
          <div class="profile-field-group">
            <label class="profile-label" for="pf-new">Password baru</label>
            <input id="pf-new" type="password" class="profile-input" bind:value={newPassword} placeholder="••••••••" />
          </div>
          <div class="profile-field-group">
            <label class="profile-label" for="pf-confirm">Konfirmasi password baru</label>
            <input id="pf-confirm" type="password" class="profile-input" bind:value={confirmPassword} placeholder="••••••••" />
          </div>
        </div>
      {/if}

      {#if error}
        <div class="profile-error">{error}</div>
      {/if}

      <div class="profile-card-actions">
        <button class="profile-btn-primary" on:click={save} disabled={saving}>
          {#if savedFlash}
            <Check size={14} />&nbsp;Tersimpan
          {:else}
            {saving ? 'Menyimpan...' : 'Simpan perubahan'}
          {/if}
        </button>
        <button class="profile-btn-ghost" on:click={() => dispatch('close')}>Tutup</button>
      </div>
    </div>
  </div>
</div>

<style>
  .profile-card {
    width: 380px;
    max-width: calc(100vw - 32px);
    max-height: 88vh;
    overflow-y: auto;
    background: var(--content-bg, var(--face));
    border-radius: 16px;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.25), 0 2px 8px rgba(0, 0, 0, 0.08);
    position: relative;
    animation: profileCardIn 0.22s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes profileCardIn {
    from { opacity: 0; transform: translateY(8px) scale(0.98); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }

  .profile-card-close {
    position: absolute;
    top: 12px;
    right: 12px;
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.15);
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: background 0.15s ease;
    z-index: 1;
  }
  .profile-card-close:hover { background: rgba(255, 255, 255, 0.3); }

  .profile-card-hero {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 28px 24px 20px;
    background: linear-gradient(135deg, var(--win-blue), color-mix(in srgb, var(--win-blue) 70%, #000));
    border-radius: 16px 16px 0 0;
    color: #fff;
  }

  .profile-card-avatar {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.2);
    border: 2px solid rgba(255, 255, 255, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    font-weight: 700;
    letter-spacing: 0.5px;
    flex-shrink: 0;
  }

  .profile-card-name {
    font-size: 16px;
    font-weight: 700;
  }

  .profile-card-role {
    font-size: 12px;
    opacity: 0.85;
    text-transform: capitalize;
    margin-top: 2px;
  }

  .profile-card-body {
    padding: 20px 24px 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .profile-field-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .profile-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .profile-input {
    border: 1px solid var(--np-border, #d1d5db);
    border-radius: 8px;
    padding: 9px 12px;
    font-size: 13px;
    background: var(--content-alt, #fff);
    color: var(--text-primary);
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
    font-family: inherit;
  }

  .profile-input:focus {
    outline: none;
    border-color: var(--win-blue);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--win-blue) 18%, transparent);
  }

  .profile-input-short {
    width: 72px;
    text-transform: uppercase;
  }

  .profile-password-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    background: none;
    border: none;
    color: var(--win-blue);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    padding: 4px 0;
    width: fit-content;
  }
  .profile-password-toggle:hover { text-decoration: underline; }
  .profile-password-toggle :global(.profile-password-toggle-icon) { opacity: 0.6; }

  .profile-password-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 14px;
    background: var(--content-alt, #f8f9fa);
    border-radius: 10px;
    border: 1px solid var(--np-border, #e5e7eb);
    animation: profileSectionIn 0.18s ease;
  }

  @keyframes profileSectionIn {
    from { opacity: 0; max-height: 0; }
    to { opacity: 1; max-height: 300px; }
  }

  .profile-error {
    font-size: 12px;
    color: var(--win-red);
    background: color-mix(in srgb, var(--win-red) 10%, transparent);
    border-radius: 6px;
    padding: 8px 10px;
  }

  .profile-card-actions {
    display: flex;
    gap: 8px;
    margin-top: 4px;
  }

  .profile-btn-primary {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    background: var(--win-blue);
    color: #fff;
    border: none;
    border-radius: 8px;
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: filter 0.15s ease, transform 0.1s ease;
  }
  .profile-btn-primary:hover:not(:disabled) { filter: brightness(1.08); }
  .profile-btn-primary:active:not(:disabled) { transform: scale(0.98); }
  .profile-btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

  .profile-btn-ghost {
    background: transparent;
    color: var(--text-muted);
    border: 1px solid var(--np-border, #d1d5db);
    border-radius: 8px;
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s ease;
  }
  .profile-btn-ghost:hover { background: var(--content-alt, #f3f4f6); }
</style>
