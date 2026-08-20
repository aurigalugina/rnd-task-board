<script lang="ts">
  import { onMount } from 'svelte';
  import { api, downloadBlob } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { AssignableUser, CheatSheetItem } from '$lib/types';
  import Avatar from './Avatar.svelte';
  import { FileText, Link2, NotebookPen, Download, Pencil, Trash2 } from 'lucide-svelte';

  // Edit & delete cheat sheet item -- super_user only, lihat
  // decision-log-boards-dashboard-enhancements-20260820.md.
  $: isSuperUser = $auth.user?.access_level === 'super_user';

  export let boardId: string;

  let items: CheatSheetItem[] = [];
  let assignableUsers: AssignableUser[] = [];
  let loading = true;
  let error: string | null = null;

  $: authorById = Object.fromEntries(assignableUsers.map((u) => [u.id, u]));

  // silent: dipakai buat refresh SETELAH nambah referensi -- kalau loading
  // di-toggle, {#if loading} bikin seluruh daftar referensi flash "Memuat"
  // tiap kali (kerasa kayak "refresh" walau SPA, dilaporkan user 2026-08-10).
  async function load({ silent = false }: { silent?: boolean } = {}) {
    if (!silent) loading = true;
    try {
      const [list, users] = await Promise.all([
        api.get<CheatSheetItem[]>(`/boards/${boardId}/cheat-sheet`),
        assignableUsers.length ? Promise.resolve(assignableUsers) : api.get<AssignableUser[]>('/users/assignable')
      ]);
      items = list;
      assignableUsers = users;
    } catch (e) {
      error = (e as Error).message;
    } finally {
      if (!silent) loading = false;
    }
  }

  onMount(() => load());

  let showAdd = false;
  let type: 'note' | 'url' | 'file' = 'note';
  let title = '';
  let value = '';
  let file: File | null = null;
  let saving = false;
  let saveError: string | null = null;
  // Non-null = form lagi dipakai buat EDIT item ini (bukan tambah baru) --
  // reuse form yang sama, submit-nya beda (PATCH vs POST).
  let editingId: string | null = null;

  function reset() {
    title = '';
    value = '';
    file = null;
    type = 'note';
    showAdd = false;
    editingId = null;
  }

  function startEdit(it: CheatSheetItem) {
    editingId = it.id;
    type = it.type;
    title = it.title;
    value = it.type === 'file' ? '' : it.value; // file: kosongin, isi lama tetap kepakai kalau gak diganti
    file = null;
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
      let finalValue = value.trim();
      if (type === 'file' && file) {
        const formData = new FormData();
        formData.append('file', file);
        const res = await api.upload<{ value: string }>('/uploads', formData);
        finalValue = res.value;
      } else if (type === 'file' && !file && editingId) {
        // Edit file tanpa ganti file baru -- pertahankan value (nama file) lama.
        finalValue = items.find((i) => i.id === editingId)?.value ?? '';
      } else if (type === 'file' && !file) {
        saveError = 'Pilih file dulu.';
        saving = false;
        return;
      }
      if (!finalValue) {
        saveError = 'Isi referensi wajib diisi.';
        saving = false;
        return;
      }
      if (editingId) {
        await api.patch(`/boards/${boardId}/cheat-sheet/${editingId}`, { type, title: title.trim(), value: finalValue });
      } else {
        await api.post(`/boards/${boardId}/cheat-sheet`, { type, title: title.trim(), value: finalValue });
      }
      reset();
      await load({ silent: true });
    } catch (e) {
      saveError = (e as Error).message;
    } finally {
      saving = false;
    }
  }

  async function deleteItem(it: CheatSheetItem) {
    if (!confirm(`Hapus referensi "${it.title}"?`)) return;
    try {
      await api.del(`/boards/${boardId}/cheat-sheet/${it.id}`);
      await load({ silent: true });
    } catch (e) {
      error = (e as Error).message;
    }
  }

  const typeIconMap = { file: FileText, url: Link2, note: NotebookPen };

  function originalFilename(value: string): string {
    return value.replace(/^[0-9a-f-]{36}_/, '');
  }

  async function download(it: CheatSheetItem) {
    try {
      const blob = await downloadBlob(`/uploads/${it.value}`);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = originalFilename(it.value);
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      error = (e as Error).message;
    }
  }
</script>

<div class="cheatsheet-section">
  <div class="section-title">Cheat sheet / referensi board</div>

  {#if loading}
    <p class="small muted">Memuat...</p>
  {:else}
    {#if error}<p class="small" style="color:var(--win-red)">{error}</p>{/if}
    <div class="cheatsheet-list">
      {#each items as it (it.id)}
        {@const author = authorById[it.author_id]}
        <div class="cheatsheet-row">
          <div class="cheatsheet-icon"><svelte:component this={typeIconMap[it.type]} size={14} /></div>
          <div class="cheatsheet-main">
            <div class="cheatsheet-title">{it.title}</div>
            {#if it.type === 'url'}
              <a class="cheatsheet-link" href={it.value} target="_blank" rel="noreferrer">{it.value}</a>
            {:else if it.type === 'file'}
              <button class="cheatsheet-link cheatsheet-download" on:click={() => download(it)}>
                <Download size={12} />&nbsp;{originalFilename(it.value)}
              </button>
            {:else}
              <span class="cheatsheet-note">{it.value}</span>
            {/if}
          </div>
          <div class="cheatsheet-meta">
            {#if author}
              <Avatar initials={author.initials} size={18} title={author.display_name} />
              <span class="small">{author.display_name}</span>
            {/if}
            <span class="mono small muted">{it.created_at.slice(0, 10)}</span>
            {#if isSuperUser}
              <button class="icon-btn" title="Edit" on:click={() => startEdit(it)}><Pencil size={11} /></button>
              <button class="icon-btn icon-btn-danger" title="Hapus" on:click={() => deleteItem(it)}><Trash2 size={11} /></button>
            {/if}
          </div>
        </div>
      {/each}
      {#if items.length === 0}
        <div class="empty-note">Belum ada referensi buat board ini.</div>
      {/if}
    </div>

    {#if showAdd}
      <div class="inline-form inline-form-daily">
        {#if editingId}<span class="small muted">Edit referensi</span>{/if}
        <div class="role-filter-pills">
          <button class="role-filter-pill {type === 'note' ? 'role-filter-pill-active' : ''}" on:click={() => (type = 'note')}>
            <NotebookPen size={12} />&nbsp;Catatan
          </button>
          <button class="role-filter-pill {type === 'url' ? 'role-filter-pill-active' : ''}" on:click={() => (type = 'url')}>
            <Link2 size={12} />&nbsp;URL
          </button>
          <button class="role-filter-pill {type === 'file' ? 'role-filter-pill-active' : ''}" on:click={() => (type = 'file')}>
            <FileText size={12} />&nbsp;File
          </button>
        </div>
        <input class="inline-input" placeholder="Judul, misal: Deploy Host to Host" bind:value={title} />
        {#if type === 'note'}
          <textarea class="inline-input inline-textarea" placeholder="Tulis keterangan lengkapnya..." bind:value />
        {:else if type === 'url'}
          <input class="inline-input" placeholder="https://..." bind:value />
        {:else}
          <input
            class="inline-input"
            type="file"
            on:change={(e) => (file = e.currentTarget.files?.[0] ?? null)}
          />
          {#if editingId}<span class="small muted">Kosongkan buat pertahankan file lama.</span>{/if}
        {/if}
        {#if saveError}<span class="small" style="color:var(--win-red)">{saveError}</span>{/if}
        <div class="inline-form-actions">
          <button class="quick-btn quick-btn-done" on:click={save} disabled={saving}>
            {saving ? 'Menyimpan...' : editingId ? 'Simpan perubahan' : 'Simpan'}
          </button>
          <button class="quick-btn" on:click={reset}>Batal</button>
        </div>
      </div>
    {:else}
      <button class="add-card-ghost" on:click={() => (showAdd = true)}>+ Tambah referensi</button>
    {/if}
  {/if}
</div>

<style>
  .cheatsheet-download {
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    font-family: inherit;
    font-size: inherit;
  }
</style>
