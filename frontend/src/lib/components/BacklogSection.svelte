<script lang="ts">
  import { onMount } from 'svelte';
  import { api, downloadBlob } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { AssignableUser, BacklogItem, BigTask, DailyTask } from '$lib/types';
  import Avatar from './Avatar.svelte';
  import { Pencil, Trash2, Download, Upload, Rocket, X } from 'lucide-svelte';

  export let boardId: string;

  // can_manage_backlog: flag independen dari access_level/roles (permintaan
  // eksplisit user, "jangan terpaut sama role") -- super_user selalu boleh,
  // selain itu HARUS flag ini true. Lihat decision-log-board-backlog-20260902.md.
  $: canManage = $auth.user?.access_level === 'super_user' || ($auth.user?.can_manage_backlog ?? false);

  let items: BacklogItem[] = [];
  let bigTasks: BigTask[] = [];
  let assignableUsers: AssignableUser[] = [];
  let loading = true;
  let error: string | null = null;

  async function load({ silent = false }: { silent?: boolean } = {}) {
    if (!silent) loading = true;
    try {
      const [list, bts, users] = await Promise.all([
        api.get<BacklogItem[]>(`/boards/${boardId}/backlog-items`),
        api.get<BigTask[]>(`/boards/${boardId}/big-tasks`),
        api.get<AssignableUser[]>('/users/assignable')
      ]);
      items = list;
      bigTasks = bts;
      assignableUsers = users;
    } catch (e) {
      error = (e as Error).message;
    } finally {
      if (!silent) loading = false;
    }
  }

  onMount(() => load());

  // ---- Tambah/edit item backlog ----
  let showAdd = false;
  let title = '';
  let description = '';
  let saving = false;
  let saveError: string | null = null;
  let editingId: string | null = null;

  function reset() {
    title = '';
    description = '';
    showAdd = false;
    editingId = null;
    saveError = null;
  }

  function startEdit(it: BacklogItem) {
    editingId = it.id;
    title = it.title;
    description = it.description;
    saveError = null;
    showAdd = true;
  }

  async function save() {
    saveError = null;
    if (!title.trim()) {
      saveError = 'Judul wajib diisi.';
      return;
    }
    saving = true;
    try {
      if (editingId) {
        await api.patch(`/backlog-items/${editingId}`, { title: title.trim(), description: description.trim() });
      } else {
        await api.post(`/boards/${boardId}/backlog-items`, { title: title.trim(), description: description.trim() });
      }
      reset();
      await load({ silent: true });
    } catch (e) {
      saveError = (e as Error).message;
    } finally {
      saving = false;
    }
  }

  async function deleteItem(it: BacklogItem) {
    if (!confirm(`Hapus item backlog "${it.title}"? Daily Task yang sudah dibuat dari item ini TIDAK ikut terhapus.`)) return;
    try {
      await api.del(`/backlog-items/${it.id}`);
      await load({ silent: true });
    } catch (e) {
      error = (e as Error).message;
    }
  }

  // ---- Template & Import Excel ----
  let downloadingTemplate = false;
  let templateError: string | null = null;

  async function downloadTemplate() {
    templateError = null;
    downloadingTemplate = true;
    try {
      const blob = await downloadBlob(`/boards/${boardId}/backlog-items/template`);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'backlog-template.xlsx';
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (e) {
      templateError = (e as Error).message;
    } finally {
      downloadingTemplate = false;
    }
  }

  let importFile: HTMLInputElement;
  let importing = false;
  let importError: string | null = null;
  let importResult: { created: number; warnings: string[] } | null = null;

  async function doImport(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    importError = null;
    importResult = null;
    importing = true;
    try {
      const formData = new FormData();
      formData.append('file', file);
      importResult = await api.upload(`/boards/${boardId}/backlog-items/import`, formData);
      await load({ silent: true });
    } catch (e2) {
      importError = (e2 as Error).message;
    } finally {
      importing = false;
      if (importFile) importFile.value = '';
    }
  }

  // ---- Promote: jadikan Daily Task ----
  // Alur: pilih Big Task dulu -> baru muncul field PIC (dibatasi anggota Big
  // Task itu, sama seperti form create Daily Task biasa) + tanggal. Item
  // backlog TETAP ADA setelah promote (reusable, bukan sekali pakai) --
  // permintaan eksplisit user.
  let promoteItem: BacklogItem | null = null;
  let promoteBigTaskId = '';
  let promotePicUserId = '';
  let promoteStart = '';
  let promoteEnd = '';
  let promoting = false;
  let promoteError: string | null = null;

  $: minDate = $auth.user?.access_level === 'super_user' ? '' : new Date().toLocaleDateString('en-CA');
  $: promoteBigTask = bigTasks.find((b) => b.id === promoteBigTaskId) ?? null;
  $: promoteMembers = promoteBigTask
    ? assignableUsers.filter((u) => promoteBigTask?.member_user_ids.includes(u.id))
    : [];

  function openPromote(it: BacklogItem) {
    promoteItem = it;
    promoteBigTaskId = bigTasks[0]?.id ?? '';
    promotePicUserId = '';
    promoteStart = '';
    promoteEnd = '';
    promoteError = null;
  }

  async function submitPromote() {
    if (!promoteItem) return;
    promoteError = null;
    if (!promoteBigTaskId) {
      promoteError = 'Pilih Big Task dulu.';
      return;
    }
    if (!promotePicUserId) {
      promoteError = 'Pilih PIC dulu.';
      return;
    }
    if (!promoteStart || !promoteEnd) {
      promoteError = 'Rentang tanggal wajib diisi.';
      return;
    }
    promoting = true;
    try {
      await api.post<DailyTask>(`/big-tasks/${promoteBigTaskId}/daily-tasks`, {
        title: promoteItem.title,
        pic_user_id: promotePicUserId,
        start_date: promoteStart,
        end_date: promoteEnd,
        source_backlog_item_id: promoteItem.id
      });
      promoteItem = null;
      await load({ silent: true });
    } catch (e) {
      promoteError = (e as Error).message;
    } finally {
      promoting = false;
    }
  }
</script>

<div class="backlog-section">
  {#if loading}
    <p class="small muted">Memuat backlog...</p>
  {:else}
    {#if error}<p class="small" style="color:var(--win-red)">{error}</p>{/if}

    <div class="backlog-list">
      {#each items as it (it.id)}
        <div class="backlog-row">
          <div class="backlog-main">
            <div class="backlog-title-row">
              <span class="backlog-title">{it.title}</span>
              {#if it.promoted_count > 0}
                <span class="badge badge-accent" title="Sudah dipakai di {it.promoted_count} daily task">
                  {it.promoted_count}x dipakai
                </span>
              {/if}
            </div>
            {#if it.description}<p class="backlog-desc small muted">{it.description}</p>{/if}
          </div>
          <div class="backlog-meta">
            <span class="small muted">{it.created_by_name}</span>
            <span class="mono small muted">{it.created_at.slice(0, 10)}</span>
            <button class="quick-btn" title="Jadikan Daily Task" on:click={() => openPromote(it)}>
              <Rocket size={12} />&nbsp;Jadikan Daily Task
            </button>
            {#if canManage}
              <button class="icon-btn" title="Edit" on:click={() => startEdit(it)}><Pencil size={11} /></button>
              <button class="icon-btn icon-btn-danger" title="Hapus" on:click={() => deleteItem(it)}><Trash2 size={11} /></button>
            {/if}
          </div>
        </div>
      {/each}
      {#if items.length === 0}
        <div class="empty-note">Belum ada item backlog buat board ini.</div>
      {/if}
    </div>

    {#if canManage}
      <div class="backlog-toolbar">
        <button class="quick-btn" on:click={downloadTemplate} disabled={downloadingTemplate}>
          <Download size={12} />&nbsp;{downloadingTemplate ? 'Mengunduh...' : 'Unduh template Excel'}
        </button>
        <label class="quick-btn" style="cursor:pointer">
          <Upload size={12} />&nbsp;{importing ? 'Mengimpor...' : 'Import dari Excel'}
          <input
            bind:this={importFile}
            type="file"
            accept=".xlsx"
            style="display:none"
            on:change={doImport}
            disabled={importing}
          />
        </label>
      </div>
      {#if templateError}<p class="small" style="color:var(--win-red)">{templateError}</p>{/if}
      {#if importError}<p class="small" style="color:var(--win-red)">{importError}</p>{/if}
      {#if importResult}
        <p class="small" style="color:var(--win-green)">
          {importResult.created} item berhasil diimpor.
          {#if importResult.warnings.length > 0}({importResult.warnings.length} peringatan){/if}
        </p>
      {/if}

      {#if showAdd}
        <div class="inline-form inline-form-daily">
          {#if editingId}<span class="small muted">Edit item backlog</span>{/if}
          <input class="inline-input" placeholder="Judul (mis. Setup CI/CD pipeline)" bind:value={title} />
          <textarea class="inline-input inline-textarea" placeholder="Deskripsi (opsional)..." bind:value={description} />
          {#if saveError}<span class="small" style="color:var(--win-red)">{saveError}</span>{/if}
          <div class="inline-form-actions">
            <button class="quick-btn quick-btn-done" on:click={save} disabled={saving}>
              {saving ? 'Menyimpan...' : editingId ? 'Simpan perubahan' : 'Simpan'}
            </button>
            <button class="quick-btn" on:click={reset}>Batal</button>
          </div>
        </div>
      {:else}
        <button class="add-card-ghost" on:click={() => (showAdd = true)}>+ Tambah item backlog</button>
      {/if}
    {/if}
  {/if}
</div>

<!-- Modal promote: pilih Big Task + PIC + tanggal, sama seperti create Daily Task -->
{#if promoteItem}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="overlay overlay-center" on:click={() => !promoting && (promoteItem = null)}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal-box" role="dialog" aria-modal="true" on:click|stopPropagation style="width:420px;max-width:calc(100vw - 32px)">
      <div class="de-modal-header">
        <span class="de-modal-title">Jadikan Daily Task — {promoteItem.title}</span>
        <button class="de-close-btn icon-btn" aria-label="Tutup" on:click={() => (promoteItem = null)}><X size={12} /></button>
      </div>
      <div class="modal-body">
        <form on:submit|preventDefault={submitPromote}>
          <div class="panel-field">
            <label class="small muted" for="promote-bt">Big Task</label>
            <select id="promote-bt" class="inline-input" bind:value={promoteBigTaskId} required>
              {#each bigTasks as bt (bt.id)}
                <option value={bt.id}>{bt.name}</option>
              {/each}
            </select>
            {#if bigTasks.length === 0}
              <span class="small" style="color:var(--win-amber)">Belum ada Big Task di board ini -- buat dulu sebelum promote.</span>
            {/if}
          </div>
          <div class="panel-field">
            <label class="small muted" for="promote-pic">PIC</label>
            <select id="promote-pic" class="inline-input" bind:value={promotePicUserId} required disabled={!promoteBigTaskId}>
              <option value="">— Pilih PIC —</option>
              {#each promoteMembers as u (u.id)}
                <option value={u.id}>{u.display_name} — {u.roles.join('/')}</option>
              {/each}
            </select>
            {#if promoteBigTaskId && promoteMembers.length === 0}
              <span class="small" style="color:var(--win-amber)">Big Task ini belum punya anggota.</span>
            {/if}
          </div>
          <div class="panel-field">
            <span class="small muted">Rentang waktu</span>
            <div class="inline-form-dates" style="padding:0">
              <label class="small muted">
                Mulai
                <input class="inline-input" type="date" bind:value={promoteStart} min={minDate} required />
              </label>
              <label class="small muted">
                Selesai
                <input class="inline-input" type="date" bind:value={promoteEnd} min={minDate} required />
              </label>
            </div>
          </div>
          {#if promoteError}<p class="small" style="color:var(--win-red);margin:0">{promoteError}</p>{/if}
          <div class="inline-form-actions" style="padding-top:4px">
            <button class="quick-btn quick-btn-done" type="submit" disabled={promoting || bigTasks.length === 0}>
              {promoting ? 'Membuat...' : 'Buat Daily Task'}
            </button>
            <button class="quick-btn" type="button" on:click={() => (promoteItem = null)}>Batal</button>
          </div>
        </form>
      </div>
    </div>
  </div>
{/if}

<style>
  .backlog-row {
    display: flex; justify-content: space-between; align-items: flex-start;
    gap: 10px; padding: 8px 6px; border-bottom: 1px solid var(--content-border, #C3C8CC);
  }
  .backlog-row:last-child { border-bottom: none; }
  .backlog-main { flex: 1; min-width: 0; }
  .backlog-title-row { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
  .backlog-title { font-weight: bold; font-size: 11px; }
  .backlog-desc { margin: 2px 0 0; white-space: pre-wrap; word-break: break-word; }
  .backlog-meta { display: flex; align-items: center; gap: 6px; flex-shrink: 0; white-space: nowrap; }
  .backlog-toolbar { display: flex; gap: 8px; margin: 8px 0; }
</style>
