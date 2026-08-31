<script lang="ts">
  import { createEventDispatcher, onMount, tick } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import { theme, THEMES } from '$lib/stores/themeStore';
  import type { AccessLevel, ManagedUser, NotificationSettings, ReferensiTim, ReferensiUserHR, Role } from '$lib/types';
  import { setupTokenApi, streamSetupToken, type AuthStatus } from '$lib/setupTokenClient';
  import { X, Bot, Smartphone, Settings as SettingsIcon, CircleCheck, CircleX, Clock, Check, KeyRound } from 'lucide-svelte';

  const dispatch = createEventDispatcher<{ close: void }>();

  $: isSuperUser = $auth.user?.access_level === 'super_user';

  $: canManage = $auth.user?.roles.some((r) => r === 'admin' || r === 'spv') ?? false;

  // Default 'users' TANPA nunggu canManage -- kalau di-init pakai
  // `canManage ? 'users' : 'theme'` di titik ini, canManage masih undefined
  // (reactive statement di atas belum jalan sekali pun saat baris ini
  // dieksekusi), jadi SELALU jatuh ke 'theme' walau user-nya admin/spv
  // (ketemu 2026-08-10 saat verifikasi visual fitur mapping HR). Template di
  // bawah sudah double-check `canManage` sendiri (`activeTab === 'users' &&
  // canManage`), jadi default polos di sini aman.
  let activeTab: 'users' | 'master' | 'notif' | 'chatauth' | 'theme' = 'users';

  let users: ManagedUser[] = [];
  let roles: Role[] = [];
  let tims: ReferensiTim[] = [];
  let usersHR: ReferensiUserHR[] = [];
  let loadingUsers = true;
  let usersError: string | null = null;

  async function loadUsers() {
    if (!canManage) return;
    loadingUsers = true;
    try {
      [users, roles, tims, usersHR] = await Promise.all([
        api.get<ManagedUser[]>('/users'),
        api.get<Role[]>('/roles'),
        api.get<ReferensiTim[]>('/referensi-tim'),
        api.get<ReferensiUserHR[]>('/referensi-user-hr')
      ]);
    } catch (e) {
      usersError = (e as Error).message;
    } finally {
      loadingUsers = false;
    }
  }

  onMount(loadUsers);

  function hrLabel(hrUserId: number | null): string {
    if (hrUserId === null) return '(belum di-mapping)';
    const hr = usersHR.find((u) => u.hr_user_id === hrUserId);
    return hr ? `${hr.nama_lengkap} (NIP ${hr.nip || '-'})` : `#${hrUserId}`;
  }

  let newName = '';
  let newEmail = '';
  let newPassword = '';
  let newInitials = '';
  let newOrgTeam = 'R&D';
  let newAccessLevel: AccessLevel = 'regular_user';
  let newHrUserId = '';
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
        role_codes: roleCodes,
        access_level: newAccessLevel,
        hr_user_id: newHrUserId ? Number(newHrUserId) : null
      });
      newName = '';
      newEmail = '';
      newPassword = '';
      newInitials = '';
      newRoles = {};
      newAccessLevel = 'regular_user';
      newHrUserId = '';
      await loadUsers();
    } catch (e) {
      createError = (e as Error).message;
    } finally {
      creating = false;
    }
  }

  // --- Tim CRUD ---
  let newTeamName = '';
  let addingTeam = false;
  let timError: string | null = null;
  let editingTimId: string | null = null;
  let editTimName = '';

  function startEditTim(t: ReferensiTim) {
    editingTimId = t.id;
    editTimName = t.name;
    timError = null;
  }

  async function addTeam() {
    if (!newTeamName.trim()) return;
    timError = null;
    addingTeam = true;
    try {
      await api.post<ReferensiTim>('/referensi-tim', { name: newTeamName.trim() });
      newTeamName = '';
      tims = await api.get<ReferensiTim[]>('/referensi-tim');
    } catch (e) {
      timError = (e as Error).message;
    } finally {
      addingTeam = false;
    }
  }

  async function saveTim() {
    if (!editingTimId || !editTimName.trim()) return;
    timError = null;
    try {
      await api.patch(`/referensi-tim/${editingTimId}`, { name: editTimName.trim() });
      editingTimId = null;
      tims = await api.get<ReferensiTim[]>('/referensi-tim');
    } catch (e) {
      timError = (e as Error).message;
    }
  }

  async function deleteTim(id: string, name: string) {
    if (!confirm(`Hapus tim "${name}"?`)) return;
    timError = null;
    try {
      await api.del(`/referensi-tim/${id}`);
      tims = await api.get<ReferensiTim[]>('/referensi-tim');
    } catch (e) {
      timError = (e as Error).message;
    }
  }

  // --- Role/Jabatan CRUD ---
  let newRoleCode = '';
  let newRoleLabel = '';
  let addingRole = false;
  let roleError: string | null = null;
  let editingRoleCode: string | null = null;
  let editRoleLabelVal = '';

  function startEditRole(r: Role) {
    editingRoleCode = r.code;
    editRoleLabelVal = r.label;
    roleError = null;
  }

  async function addRole() {
    if (!newRoleCode.trim() || !newRoleLabel.trim()) return;
    roleError = null;
    addingRole = true;
    try {
      await api.post<Role>('/roles', { code: newRoleCode.trim(), label: newRoleLabel.trim() });
      newRoleCode = '';
      newRoleLabel = '';
      roles = await api.get<Role[]>('/roles');
    } catch (e) {
      roleError = (e as Error).message;
    } finally {
      addingRole = false;
    }
  }

  async function saveRole() {
    if (!editingRoleCode || !editRoleLabelVal.trim()) return;
    roleError = null;
    try {
      await api.patch(`/roles/${editingRoleCode}`, { label: editRoleLabelVal.trim() });
      editingRoleCode = null;
      roles = await api.get<Role[]>('/roles');
    } catch (e) {
      roleError = (e as Error).message;
    }
  }

  async function deleteRole(code: string, label: string) {
    if (!confirm(`Hapus jabatan "${label}" (code: ${code})?`)) return;
    roleError = null;
    try {
      await api.del(`/roles/${code}`);
      roles = await api.get<Role[]>('/roles');
    } catch (e) {
      roleError = (e as Error).message;
    }
  }

  // Edit user existing -- SEBELUM ini tidak ada cara ubah org_team/access_level/
  // hr_user_id/roles user yang sudah dibuat (cuma bisa set saat create). Lihat
  // docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
  let editingUserId: string | null = null;
  let editOrgTeam = '';
  let editAccessLevel: AccessLevel = 'regular_user';
  let editTaskScopeVisibility: 'self' | 'team' = 'team'; // 2026-08-31
  let editHrUserId = '';
  let editRoles: Record<string, boolean> = {};
  let savingEdit = false;
  let editError: string | null = null;

  function startEdit(u: ManagedUser) {
    editingUserId = u.id;
    editOrgTeam = u.org_team;
    editAccessLevel = u.access_level;
    editTaskScopeVisibility = u.task_scope_visibility ?? 'team';
    editHrUserId = u.hr_user_id !== null ? String(u.hr_user_id) : '';
    editRoles = Object.fromEntries(roles.map((r) => [r.code, u.roles.includes(r.code)]));
    editError = null;
  }

  function cancelEdit() {
    editingUserId = null;
  }

  async function saveEdit() {
    if (!editingUserId) return;
    savingEdit = true;
    editError = null;
    try {
      const roleCodes = Object.entries(editRoles)
        .filter(([, checked]) => checked)
        .map(([code]) => code);
      await api.patch(`/users/${editingUserId}`, {
        org_team: editOrgTeam,
        access_level: editAccessLevel,
        task_scope_visibility: editTaskScopeVisibility,
        hr_user_id: editHrUserId ? Number(editHrUserId) : null,
        clear_hr_user_id: !editHrUserId,
        role_codes: roleCodes
      });
      editingUserId = null;
      await loadUsers();
    } catch (e) {
      editError = (e as Error).message;
    } finally {
      savingEdit = false;
    }
  }

  // --- Notifikasi ---
  const defaultNotifSettings: NotificationSettings = {
    telegram_chat_id: '', telegram_thread_id: '',
    deadline_threshold_days: 3, cooldown_hours: 24,
    notify_sign_off_ready: true, notify_verdict_lose: true, notify_deadline_soon: true,
  };
  let notifSettings: NotificationSettings = { ...defaultNotifSettings };
  let notifLoading = false;
  let notifSaving = false;
  let notifError: string | null = null;
  let notifSaved = false;
  let botToken = '';
  let botTokenSaving = false;
  let botTokenSaved = false;
  let botTokenError: string | null = null;

  async function loadNotifTab() {
    notifLoading = true;
    notifError = null;
    try {
      notifSettings = await api.get<NotificationSettings>('/notifications/settings');
      if (canManage) {
        const cfg = await api.get<{ key: string; value: string }>('/app-config/telegram_bot_token');
        botToken = cfg.value ?? '';
      }
    } catch (e) {
      notifError = (e as Error).message;
    } finally {
      notifLoading = false;
    }
  }

  async function saveNotifSettings() {
    notifSaving = true; notifError = null; notifSaved = false;
    try {
      await api.patch('/notifications/settings', notifSettings);
      notifSaved = true;
      setTimeout(() => (notifSaved = false), 2000);
    } catch (e) {
      notifError = (e as Error).message;
    } finally {
      notifSaving = false;
    }
  }

  async function saveBotToken() {
    botTokenSaving = true; botTokenError = null; botTokenSaved = false;
    try {
      await api.put('/app-config/telegram_bot_token', { value: botToken });
      botTokenSaved = true;
      setTimeout(() => (botTokenSaved = false), 2000);
    } catch (e) {
      botTokenError = (e as Error).message;
    } finally {
      botTokenSaving = false;
    }
  }

  $: if (activeTab === 'notif') loadNotifTab();

  // --- Login Claude (claude-chat-service, super_user only) ---
  // Provisioning ulang token OAuth akun Claude subscription BERSAMA (dipakai
  // semua sesi Change Request chat) -- sebelumnya cuma bisa lewat curl manual
  // ke claude-chat-service langsung di server. Sekarang di-handle service itu
  // sendiri (endpoint POST /auth/setup-token, SSE) dan dipicu dari sini lewat
  // backend proxy (digate super_user di chatproxy Go, lihat CLAUDE.md
  // "Login OAuth Claude di-handle service sendiri", 2026-08-21).
  let authStatus: AuthStatus | null = null;
  let authStatusLoading = false;
  let authStreaming = false;
  let authSessionId: string | null = null;
  let authOutput = '';
  let authDone: { success: true; capturedAt: string } | { success: false; reason: string } | null = null;
  let authStreamError: string | null = null;
  let authAbort: AbortController | null = null;
  let authInputValue = '';
  let authSendingInput = false;
  let authOutputBox: HTMLElement | undefined;

  async function loadAuthStatus() {
    authStatusLoading = true;
    try {
      authStatus = await setupTokenApi.status();
    } catch (e) {
      authStatus = null;
      authStreamError = (e as Error).message;
    } finally {
      authStatusLoading = false;
    }
  }

  $: if (activeTab === 'chatauth' && isSuperUser) loadAuthStatus();

  async function scrollAuthOutputToBottom() {
    await tick();
    authOutputBox?.scrollTo(0, authOutputBox.scrollHeight);
  }

  async function startClaudeLogin() {
    authOutput = '';
    authSessionId = null;
    authDone = null;
    authStreamError = null;
    authStreaming = true;
    authAbort = new AbortController();
    try {
      await streamSetupToken(async (evt) => {
        if (evt.type === 'session') {
          authSessionId = evt.sessionId;
        } else if (evt.type === 'output') {
          authOutput += evt.chunk;
          await scrollAuthOutputToBottom();
        } else if (evt.type === 'done') {
          authDone = evt;
          if (evt.success) await loadAuthStatus();
        }
      }, authAbort.signal);
    } catch (e) {
      authStreamError = (e as Error).message;
    } finally {
      authStreaming = false;
      authSessionId = null; // proses PTY sudah exit (sukses/gagal/dibatalkan) -- gak bisa nerima input lagi
    }
  }

  function cancelClaudeLogin() {
    authAbort?.abort();
  }

  async function sendClaudeLoginInput() {
    if (!authSessionId || !authInputValue.trim()) return;
    authSendingInput = true;
    try {
      await setupTokenApi.sendInput(authSessionId, authInputValue);
      authInputValue = '';
    } catch (e) {
      authStreamError = (e as Error).message;
    } finally {
      authSendingInput = false;
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
  <div
    class="modal-box modal-box-wide"
    class:modal-box-user-mgmt={activeTab === 'users' && canManage}
    class:modal-box-master={activeTab === 'master' && canManage}
    class:modal-box-chatauth={activeTab === 'chatauth' && isSuperUser}
    role="dialog"
    aria-modal="true"
    aria-label="Settings"
    on:click|stopPropagation
  >
    <div class="panel-header">
      <span class="titlebar-title" style="font-size:11px">Settings</span>
      <button class="icon-btn" on:click={() => dispatch('close')} aria-label="Tutup"><X size={13} /></button>
    </div>
    <div class="settings-tabs">
      {#if canManage}
        <button class="role-filter-pill {activeTab === 'users' ? 'role-filter-pill-active' : ''}" on:click={() => (activeTab = 'users')}>
          Manajemen user
        </button>
        <button class="role-filter-pill {activeTab === 'master' ? 'role-filter-pill-active' : ''}" on:click={() => (activeTab = 'master')}>
          Tim &amp; Jabatan
        </button>
      {/if}
      <button class="role-filter-pill {activeTab === 'notif' ? 'role-filter-pill-active' : ''}" on:click={() => (activeTab = 'notif')}>
        Notifikasi
      </button>
      {#if isSuperUser}
        <button class="role-filter-pill {activeTab === 'chatauth' ? 'role-filter-pill-active' : ''}" on:click={() => (activeTab = 'chatauth')}>
          Login Claude
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
          <div class="modal-body-scroll-x">
          <table class="sheet-table">
            <thead>
              <tr><th>Nama</th><th>Email</th><th>Role</th><th>Tim</th><th>Access</th><th>Scope</th><th>Mapping HR</th><th></th></tr>
            </thead>
            <tbody>
              {#each users as u (u.id)}
                <tr>
                  {#if editingUserId === u.id}
                    <td colspan="8">
                      <div class="inline-form" style="margin:0">
                        <strong class="small">{u.display_name} ({u.initials})</strong>
                        <select class="inline-input" bind:value={editOrgTeam}>
                          {#each tims as t (t.id)}
                            <option value={t.name}>{t.name}</option>
                          {/each}
                        </select>
                        <select class="inline-input" bind:value={editAccessLevel}>
                          <option value="regular_user">regular_user</option>
                          <option value="super_user">super_user</option>
                        </select>
                        <select class="inline-input" bind:value={editHrUserId}>
                          <option value="">(belum di-mapping)</option>
                          {#each usersHR as hr (hr.hr_user_id)}
                            <option value={hr.hr_user_id}>{hr.nama_lengkap} — {hr.email}</option>
                          {/each}
                        </select>
                        <select class="inline-input" bind:value={editTaskScopeVisibility}>
                          <option value="team">Lihat semua task tim</option>
                          <option value="self">Lihat hanya task sendiri</option>
                        </select>
                        <div class="role-filter-pills">
                          {#each roles as role (role.code)}
                            <label class="role-filter-pill" class:role-filter-pill-active={editRoles[role.code]}>
                              <input type="checkbox" bind:checked={editRoles[role.code]} style="margin-right:4px" />
                              {role.label}
                            </label>
                          {/each}
                        </div>
                        {#if editError}<span class="small" style="color:var(--win-red)">{editError}</span>{/if}
                        <div class="inline-form-actions">
                          <button class="quick-btn quick-btn-done" on:click={saveEdit} disabled={savingEdit}>
                            {savingEdit ? 'Menyimpan...' : 'Simpan'}
                          </button>
                          <button class="quick-btn" on:click={cancelEdit}>Batal</button>
                        </div>
                      </div>
                    </td>
                  {:else}
                    <td>{u.display_name} ({u.initials})</td>
                    <td class="small">{u.email}</td>
                    <td class="small muted">{u.roles.join(', ')}</td>
                    <td class="small muted">{u.org_team}</td>
                    <td class="small">
                      {#if u.access_level === 'super_user'}
                        <span class="badge badge-win">super_user</span>
                      {:else}
                        <span class="muted">regular_user</span>
                      {/if}
                    </td>
                    <td class="small muted">{u.task_scope_visibility ?? 'team'}</td>
                    <td class="small muted">{hrLabel(u.hr_user_id)}</td>
                    <td><button class="quick-btn" on:click={() => startEdit(u)}>Edit</button></td>
                  {/if}
                </tr>
              {/each}
            </tbody>
          </table>
          </div>

          <div class="inline-form inline-form-daily">
            <input class="inline-input" placeholder="Nama tampilan" bind:value={newName} />
            <input class="inline-input" placeholder="Email" bind:value={newEmail} />
            <input class="inline-input" type="password" placeholder="Password awal" bind:value={newPassword} />
            <input class="inline-input" placeholder="Inisial" maxlength="2" bind:value={newInitials} />
            <select class="inline-input" bind:value={newOrgTeam}>
              {#each tims as t (t.id)}
                <option value={t.name}>{t.name}</option>
              {/each}
            </select>
            <select class="inline-input" bind:value={newAccessLevel}>
              <option value="regular_user">regular_user</option>
              <option value="super_user">super_user</option>
            </select>
            <select class="inline-input" bind:value={newHrUserId}>
              <option value="">Mapping HR (opsional)</option>
              {#each usersHR as hr (hr.hr_user_id)}
                <option value={hr.hr_user_id}>{hr.nama_lengkap} — {hr.email}</option>
              {/each}
            </select>
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
      {:else if activeTab === 'master' && canManage}
        <!-- Tim / Organisasi -->
        <div class="ref-section">
          <div class="section-title">Tim / Organisasi</div>
          {#each tims as t (t.id)}
            {#if editingTimId === t.id}
              <div class="ref-row">
                <input class="inline-input" bind:value={editTimName} style="flex:1" />
                <button class="quick-btn quick-btn-done" on:click={saveTim}>Simpan</button>
                <button class="quick-btn" on:click={() => (editingTimId = null)}>Batal</button>
              </div>
            {:else}
              <div class="ref-row">
                <span class="ref-name">{t.name}</span>
                <button class="quick-btn" on:click={() => startEditTim(t)}>Ubah</button>
                <button class="quick-btn quick-btn-danger" on:click={() => deleteTim(t.id, t.name)}>Hapus</button>
              </div>
            {/if}
          {/each}
          <div class="ref-row ref-add-row">
            <input class="inline-input" placeholder="Nama tim baru" bind:value={newTeamName} style="flex:1" />
            <button class="quick-btn" on:click={addTeam} disabled={addingTeam}>
              {addingTeam ? 'Menyimpan...' : '+ Tambah'}
            </button>
          </div>
          {#if timError}<p class="small" style="color:var(--win-red);margin:0">{timError}</p>{/if}
        </div>

        <!-- Jabatan (Roles) -->
        <div class="ref-section">
          <div class="section-title">Jabatan</div>
          {#each roles as role (role.code)}
            {#if editingRoleCode === role.code}
              <div class="ref-row">
                <span class="ref-code">{role.code}</span>
                <input class="inline-input" bind:value={editRoleLabelVal} style="flex:1" />
                <button class="quick-btn quick-btn-done" on:click={saveRole}>Simpan</button>
                <button class="quick-btn" on:click={() => (editingRoleCode = null)}>Batal</button>
              </div>
            {:else}
              <div class="ref-row">
                <span class="ref-code">{role.code}</span>
                <span class="ref-name">{role.label}</span>
                <button class="quick-btn" on:click={() => startEditRole(role)}>Ubah</button>
                <button class="quick-btn quick-btn-danger" on:click={() => deleteRole(role.code, role.label)}>Hapus</button>
              </div>
            {/if}
          {/each}
          <div class="ref-row ref-add-row">
            <input class="inline-input" placeholder="Code (mis. pm)" bind:value={newRoleCode} style="flex:0 0 110px;min-width:0" />
            <input class="inline-input" placeholder="Label (mis. Project Manager)" bind:value={newRoleLabel} style="flex:1" />
            <button class="quick-btn" on:click={addRole} disabled={addingRole}>
              {addingRole ? 'Menyimpan...' : '+ Tambah'}
            </button>
          </div>
          {#if roleError}<p class="small" style="color:var(--win-red);margin:0">{roleError}</p>{/if}
        </div>
      {:else if activeTab === 'notif'}
        {#if notifLoading}
          <p class="small muted">Memuat...</p>
        {:else}
          {#if canManage}
            <!-- BOT TOKEN — admin only -->
            <div class="ref-section">
              <div class="section-title"><Bot size={13} />&nbsp;Konfigurasi Bot Telegram (Admin)</div>
              <p class="small muted" style="margin:0">Token bot global — dipakai untuk semua user. Buat bot baru via <b>@BotFather</b> di Telegram, lalu copy token-nya ke sini.</p>
              <div class="ref-row">
                <input class="inline-input" type="password" placeholder="1234567890:AAxxxxxxxx..." bind:value={botToken} style="flex:1;font-family:monospace" />
                <button class="quick-btn quick-btn-done" on:click={saveBotToken} disabled={botTokenSaving}>
                  {botTokenSaving ? 'Menyimpan...' : botTokenSaved ? 'Tersimpan' : 'Simpan'}
                </button>
              </div>
              {#if botTokenError}<p class="small" style="color:var(--win-red);margin:0">{botTokenError}</p>{/if}
            </div>
          {/if}

          <!-- TELEGRAM SETUP USER -->
          <div class="ref-section">
            <div class="section-title"><Smartphone size={13} />&nbsp;Setup Telegram</div>
            <div class="notif-guide">
              <b>Cara mendapatkan Chat ID:</b>
              <ol class="notif-guide-steps">
                <li>Buka Telegram, cari bot <b>@userinfobot</b></li>
                <li>Kirim pesan <code>/start</code> ke bot tersebut</li>
                <li>Bot akan membalas dengan ID kamu (angka seperti <code>123456789</code>)</li>
                <li>Untuk <b>group</b>: tambahkan bot alert ke group, forward sembarang pesan ke @userinfobot — ID group diawali <code>-</code></li>
                <li>Untuk <b>group topic</b>: isi juga Thread ID (klik kanan topik → Copy Link, angka terakhir di URL)</li>
              </ol>
            </div>
            <div class="ref-row">
              <span class="small muted" style="min-width:90px">Chat ID</span>
              <input class="inline-input" placeholder="mis. 123456789 atau -1001234567890" bind:value={notifSettings.telegram_chat_id} style="flex:1" />
            </div>
            <div class="ref-row">
              <span class="small muted" style="min-width:90px">Thread ID</span>
              <input class="inline-input" placeholder="(opsional, untuk group topic)" bind:value={notifSettings.telegram_thread_id} style="flex:1" />
            </div>
          </div>

          <!-- PREFERENSI ALERT -->
          <div class="ref-section">
            <div class="section-title"><SettingsIcon size={13} />&nbsp;Preferensi Alert</div>
            <label class="notif-toggle-row">
              <input type="checkbox" bind:checked={notifSettings.notify_sign_off_ready} />
              <CircleCheck size={13} style="color:var(--win-green);flex-shrink:0" />
              <span class="small">Sign-off siap — Big Task sudah 100%, menunggu SPV sign-off</span>
            </label>
            <label class="notif-toggle-row">
              <input type="checkbox" bind:checked={notifSettings.notify_verdict_lose} />
              <CircleX size={13} style="color:var(--win-red);flex-shrink:0" />
              <span class="small">Verdict Lose — deadline lewat, task belum selesai</span>
            </label>
            <label class="notif-toggle-row">
              <input type="checkbox" bind:checked={notifSettings.notify_deadline_soon} />
              <Clock size={13} style="color:var(--win-amber);flex-shrink:0" />
              <span class="small">Deadline dekat — dalam N hari ke depan</span>
            </label>
            <div class="ref-row" style="margin-top:4px">
              <span class="small muted" style="min-width:120px">Batas deadline (hari)</span>
              <input class="inline-input" type="number" min="1" max="30" bind:value={notifSettings.deadline_threshold_days} style="width:60px" />
              <span class="small muted">hari sebelum deadline</span>
            </div>
            <div class="ref-row">
              <span class="small muted" style="min-width:120px">Cooldown kirim ulang</span>
              <input class="inline-input" type="number" min="1" max="168" bind:value={notifSettings.cooldown_hours} style="width:60px" />
              <span class="small muted">jam (hindari spam notif yang sama)</span>
            </div>
          </div>

          {#if notifError}<p class="small" style="color:var(--win-red);margin:0">{notifError}</p>{/if}
          <button class="quick-btn quick-btn-done" on:click={saveNotifSettings} disabled={notifSaving} style="align-self:flex-start">
            {notifSaving ? 'Menyimpan...' : notifSaved ? 'Tersimpan' : 'Simpan pengaturan notifikasi'}
          </button>
        {/if}
      {:else if activeTab === 'chatauth' && isSuperUser}
        <div class="ref-section">
          <div class="section-title"><KeyRound size={13} />&nbsp;Login Claude (claude-chat-service)</div>
          <p class="small muted" style="margin:0">
            Akun Claude subscription BERSAMA yang dipakai semua sesi Change Request chat. Token OAuth-nya berumur
            panjang (~1 tahun), tapi kalau service melapor "OAuth access token has expired", provisioning ulang di sini —
            gak perlu SSH/curl manual ke server lagi.
          </p>

          {#if authStatusLoading}
            <p class="small muted">Memuat status...</p>
          {:else if authStatus}
            <div class="ref-row">
              {#if authStatus.loggedIn}
                <CircleCheck size={13} style="color:var(--win-green);flex-shrink:0" />
                <span class="small">Login aktif</span>
              {:else}
                <CircleX size={13} style="color:var(--win-red);flex-shrink:0" />
                <span class="small">Belum/tidak login{authStatus.error ? ` — ${authStatus.error}` : ''}</span>
              {/if}
            </div>
            {#if authStatus.tokenProvisioning.present}
              <p class="small muted" style="margin:0">
                Terakhir di-provisioning: <span class="mono">{authStatus.tokenProvisioning.capturedAt}</span>
              </p>
            {/if}
          {/if}

          {#if !authStreaming}
            <button class="quick-btn quick-btn-done" on:click={startClaudeLogin} style="align-self:flex-start">
              {authDone ? 'Login ulang' : 'Mulai login'}
            </button>
          {/if}

          {#if authStreaming || authOutput}
            <div class="setup-token-log" bind:this={authOutputBox}>{authOutput || 'Menunggu output...'}</div>
          {/if}

          {#if authStreaming}
            {#if authSessionId}
              <div class="ref-row">
                <input
                  class="inline-input mono"
                  style="flex:1"
                  placeholder="Paste authorization code di sini kalau diminta, lalu Enter"
                  bind:value={authInputValue}
                  on:keydown={(e) => e.key === 'Enter' && sendClaudeLoginInput()}
                  disabled={authSendingInput}
                />
                <button class="quick-btn" on:click={sendClaudeLoginInput} disabled={authSendingInput || !authInputValue.trim()}>
                  Kirim
                </button>
              </div>
            {/if}
            <button class="quick-btn quick-btn-danger" on:click={cancelClaudeLogin} style="align-self:flex-start">
              Batalkan
            </button>
          {/if}

          {#if authDone}
            {#if authDone.success}
              <p class="small" style="color:var(--win-green);margin:0">Login berhasil, token tersimpan.</p>
            {:else}
              <p class="small" style="color:var(--win-red);margin:0">Gagal: {authDone.reason}</p>
            {/if}
          {/if}
          {#if authStreamError}<p class="small" style="color:var(--win-red);margin:0">{authStreamError}</p>{/if}
        </div>
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
