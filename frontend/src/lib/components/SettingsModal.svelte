<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import { theme, THEMES } from '$lib/stores/themeStore';
  import type { ManagedUser, Role } from '$lib/types';

  const dispatch = createEventDispatcher<{ close: void }>();

  $: canManage = $auth.user?.roles.some((r) => r === 'admin' || r === 'spv') ?? false;

  let activeTab: 'users' | 'theme' = canManage ? 'users' : 'theme';

  let users: ManagedUser[] = [];
  let roles: Role[] = [];
  let loadingUsers = true;
  let usersError: string | null = null;

  async function loadUsers() {
    if (!canManage) return;
    loadingUsers = true;
    try {
      [users, roles] = await Promise.all([api.get<ManagedUser[]>('/users'), api.get<Role[]>('/roles')]);
    } catch (e) {
      usersError = (e as Error).message;
    } finally {
      loadingUsers = false;
    }
  }

  onMount(loadUsers);

  let newName = '';
  let newEmail = '';
  let newPassword = '';
  let newInitials = '';
  let newOrgTeam = 'R&D';
  let newRoles: Record<string, boolean> = {};
  let creating = false;
  let createError: string | null = null;

  async function createUser() {
    createError = null;
    creating = true;
    try {
      const roleCodes = Object.entries(newRoles)
        .filter(([, checked]) => checked)
        .map(([code]) => code);
      await api.post('/users', {
        display_name: newName,
        email: newEmail,
        password: newPassword,
        initials: newInitials,
        org_team: newOrgTeam,
        role_codes: roleCodes
      });
      newName = '';
      newEmail = '';
      newPassword = '';
      newInitials = '';
      newRoles = {};
      await loadUsers();
    } catch (e) {
      createError = (e as Error).message;
    } finally {
      creating = false;
    }
  }

  let themeSaving = false;
  async function pickTheme(key: (typeof THEMES)[number]['key']) {
    theme.set(key);
    themeSaving = true;
    try {
      await api.patch('/users/me', { theme_preference: key });
      await auth.refreshUser();
    } finally {
      themeSaving = false;
    }
  }
</script>

<svelte:window on:keydown={(e) => e.key === 'Escape' && dispatch('close')} />

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="overlay overlay-center" on:click={() => dispatch('close')}>
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div class="modal-box modal-box-wide" role="dialog" aria-modal="true" aria-label="Settings" on:click|stopPropagation>
    <div class="panel-header">
      <span class="titlebar-title" style="font-size:11px">Settings</span>
      <button class="icon-btn" on:click={() => dispatch('close')} aria-label="Tutup">✕</button>
    </div>
    <div class="settings-tabs">
      {#if canManage}
        <button class="role-filter-pill {activeTab === 'users' ? 'role-filter-pill-active' : ''}" on:click={() => (activeTab = 'users')}>
          Manajemen user
        </button>
      {/if}
      <button class="role-filter-pill {activeTab === 'theme' ? 'role-filter-pill-active' : ''}" on:click={() => (activeTab = 'theme')}>
        Tema aplikasi
      </button>
    </div>
    <div class="modal-body">
      {#if activeTab === 'users' && canManage}
        {#if loadingUsers}
          <p class="small muted">Memuat...</p>
        {:else if usersError}
          <p class="small" style="color:var(--win-red)">{usersError}</p>
        {:else}
          <table class="sheet-table">
            <thead>
              <tr><th>Nama</th><th>Email</th><th>Role</th></tr>
            </thead>
            <tbody>
              {#each users as u (u.id)}
                <tr>
                  <td>{u.display_name} ({u.initials})</td>
                  <td class="small">{u.email}</td>
                  <td class="small muted">{u.roles.join(', ')}</td>
                </tr>
              {/each}
            </tbody>
          </table>
          <div class="inline-form inline-form-daily">
            <input class="inline-input" placeholder="Nama tampilan" bind:value={newName} />
            <input class="inline-input" placeholder="Email" bind:value={newEmail} />
            <input class="inline-input" type="password" placeholder="Password awal" bind:value={newPassword} />
            <input class="inline-input" placeholder="Inisial" maxlength="2" bind:value={newInitials} />
            <input class="inline-input" placeholder="Tim/Org" bind:value={newOrgTeam} />
            <div class="role-filter-pills">
              {#each roles as role (role.code)}
                <label class="role-filter-pill" class:role-filter-pill-active={newRoles[role.code]}>
                  <input type="checkbox" bind:checked={newRoles[role.code]} style="margin-right:4px" />
                  {role.label}
                </label>
              {/each}
            </div>
            {#if createError}<span class="small" style="color:var(--win-red)">{createError}</span>{/if}
            <button class="quick-btn quick-btn-done" on:click={createUser} disabled={creating}>
              {creating ? 'Menyimpan...' : 'Buat user'}
            </button>
          </div>
        {/if}
      {:else}
        <div class="theme-grid">
          {#each THEMES as opt (opt.key)}
            <button class="theme-option {$theme === opt.key ? 'theme-option-active' : ''}" on:click={() => pickTheme(opt.key)}>
              <span class="theme-swatch-row">
                <span class="theme-swatch" style="background:{opt.titlebarA}" />
                <span class="theme-swatch" style="background:{opt.face}" />
                <span class="theme-swatch" style="background:{opt.winBlue}" />
              </span>
              <span class="small">{opt.label}</span>
            </button>
          {/each}
        </div>
        {#if themeSaving}<p class="small muted">Menyimpan...</p>{/if}
      {/if}
    </div>
  </div>
</div>
