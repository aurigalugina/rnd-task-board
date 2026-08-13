<script lang="ts">
  import { auth } from '$lib/stores/authStore';
  import { api, downloadBlob } from '$lib/api/client';
  import { Database, Download, Upload, AlertTriangle } from 'lucide-svelte';

  $: isSuperUser = $auth.user?.access_level === 'super_user';

  // Export
  let exporting = false;
  let exportError: string | null = null;

  async function doExport() {
    exportError = null;
    exporting = true;
    try {
      const blob = await downloadBlob('/admin/export');
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `rndops-export-${new Date().toISOString().slice(0, 10)}.xlsx`;
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (e) {
      exportError = (e as Error).message;
    } finally {
      exporting = false;
    }
  }

  // Import
  let fileInput: HTMLInputElement;
  let importFile: File | null = null;
  let importing = false;
  let importError: string | null = null;
  let importResult: {
    boards_created: number;
    big_tasks_created: number;
    daily_tasks_created: number;
    day_entries_created: number;
    warnings: string[];
  } | null = null;

  function handleFileChange(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0];
    importFile = f ?? null;
    importError = null;
    importResult = null;
  }

  async function doImport() {
    if (!importFile) return;
    importError = null;
    importResult = null;
    importing = true;
    try {
      const formData = new FormData();
      formData.append('file', importFile);
      importResult = await api.upload('/admin/import', formData);
    } catch (e) {
      importError = (e as Error).message;
    } finally {
      importing = false;
    }
  }
</script>

{#if !isSuperUser}
  <div class="section" style="max-width:500px">
    <div class="section-title">Import / Export data</div>
    <p class="small muted">Fitur ini hanya tersedia untuk super user.</p>
  </div>
{:else}
  <div class="section" style="max-width:600px">
    <div class="section-title"><Database size={14} style="vertical-align:-2px" />&nbsp;Import / Export data</div>
    <p class="small muted" style="margin-bottom:16px">
      Export data project aktif (Board, Big Task, Daily Task, Day Entry) ke file Excel (.xlsx), lalu bisa
      diimpor kembali ke environment lain. Komentar dan riwayat sign-off tidak disertakan.
    </p>

    <!-- Export -->
    <div class="panel" style="margin-bottom:16px">
      <div class="panel-header">
        <span style="font-size:11px;font-weight:bold">Export data</span>
      </div>
      <div class="modal-body">
        <p class="small muted">
          Download semua board aktif (non-archived) beserta seluruh big task, daily task, dan day entry
          sebagai file Excel (.xlsx). File ini bisa dibuka di Excel/LibreOffice dan dipakai untuk
          backup atau migrasi ke deployment baru.
        </p>
        {#if exportError}<p class="small" style="color:var(--win-red)">{exportError}</p>{/if}
        <button class="quick-btn quick-btn-done" on:click={doExport} disabled={exporting}>
          <Download size={12} />&nbsp;{exporting ? 'Mengunduh...' : 'Download export.xlsx'}
        </button>
      </div>
    </div>

    <!-- Import -->
    <div class="panel">
      <div class="panel-header">
        <span style="font-size:11px;font-weight:bold">Import data</span>
      </div>
      <div class="modal-body">
        <div class="import-warning">
          <AlertTriangle size={13} style="flex-shrink:0;margin-top:1px" />
          <div class="small">
            Import akan <strong>menambahkan</strong> data dari file ke database yang ada — bukan menggantikan.
            Pastikan environment ini benar-benar kosong atau kamu memang ingin menambah data.
            User yang ada di file tapi tidak ditemukan di environment ini akan di-skip (daily task tanpa PIC
            tidak diimpor; anggota big task yang tidak ditemukan diabaikan).
          </div>
        </div>

        <div class="panel-field" style="margin-top:12px">
          <label class="small muted" for="import-file">Pilih file export Excel (.xlsx)</label>
          <input
            id="import-file"
            type="file"
            accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            class="inline-input"
            style="padding:4px"
            bind:this={fileInput}
            on:change={handleFileChange}
          />
        </div>

        {#if importError}<p class="small" style="color:var(--win-red);margin-top:8px">{importError}</p>{/if}

        {#if importResult}
          <div class="import-result">
            <div class="small" style="font-weight:bold;margin-bottom:6px">Import berhasil</div>
            <div class="import-stat-grid">
              <span class="small muted">Board dibuat:</span><span class="small mono">{importResult.boards_created}</span>
              <span class="small muted">Big task:</span><span class="small mono">{importResult.big_tasks_created}</span>
              <span class="small muted">Daily task:</span><span class="small mono">{importResult.daily_tasks_created}</span>
              <span class="small muted">Day entry:</span><span class="small mono">{importResult.day_entries_created}</span>
            </div>
            {#if importResult.warnings.length > 0}
              <div class="small muted" style="margin-top:8px;font-weight:bold">Peringatan ({importResult.warnings.length}):</div>
              <ul class="import-warn-list">
                {#each importResult.warnings as w}
                  <li class="small">{w}</li>
                {/each}
              </ul>
            {/if}
          </div>
        {/if}

        <div style="margin-top:12px">
          <button class="quick-btn quick-btn-done" on:click={doImport} disabled={!importFile || importing}>
            <Upload size={12} />&nbsp;{importing ? 'Mengimpor...' : 'Import sekarang'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .import-warning {
    display: flex;
    gap: 8px;
    align-items: flex-start;
    padding: 8px 10px;
    background: color-mix(in srgb, var(--win-amber) 12%, var(--content-bg));
    border: 1px solid color-mix(in srgb, var(--win-amber) 40%, transparent);
    border-radius: 2px;
    color: var(--text-primary);
  }
  .import-result {
    margin-top: 12px;
    padding: 10px;
    background: color-mix(in srgb, var(--win-green) 10%, var(--content-bg));
    border: 1px solid color-mix(in srgb, var(--win-green) 40%, transparent);
    border-radius: 2px;
  }
  .import-stat-grid {
    display: grid;
    grid-template-columns: auto auto;
    gap: 2px 16px;
    width: fit-content;
  }
  .import-warn-list {
    margin: 4px 0 0 16px;
    padding: 0;
    color: var(--win-amber);
  }
</style>
