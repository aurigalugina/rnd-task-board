<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { ChangeRequest, CRStatus } from '$lib/types';
  import { crStatusLabel, crStatusTone, CR_TRIAGE_TARGETS } from '$lib/changeRequest';
  import ChangeRequestChat from '$lib/components/ChangeRequestChat.svelte';
  import { chatSession, beginSetup } from '$lib/stores/chatSessionStore';
  import { renderMarkdown } from '$lib/markdown';
  import { MessageSquare } from 'lucide-svelte';

  let items: ChangeRequest[] = [];
  let loading = true;
  let error: string | null = null;
  // Dua toggle independen per item -- "Lihat percakapan" (raw_conversation,
  // seluruh transcript chat) dan "Lihat dokumen" (document_md, hasil "Susun
  // change request" -- cuma ada tombolnya kalau field-nya terisi). Keduanya
  // dirender lewat renderMarkdown() + class global .cr-md (app.css), bukan
  // <pre> plain text lagi -- lihat keluhan user soal transcript kebaca mentah.
  let expanded: string | null = null;
  let expandedDoc: string | null = null;
  let triagingId: string | null = null;

  // Panel chat tampil selama sesi belum 'idle'. step HIDUP di store level modul,
  // jadi kalau user pindah menu lalu balik ke sini, panel + history muncul lagi
  // (sesi tidak mati sampai user klik "Akhiri sesi").
  $: showChat = $chatSession.step !== 'idle';

  // Triase hanya SPV & System Analyst (sa) -- lihat decision log & main.go.
  $: canTriage = ($auth.user?.roles.some((r) => r === 'spv' || r === 'sa')) ?? false;

  async function load({ silent = false } = {}) {
    if (!silent) loading = true;
    try {
      items = await api.get<ChangeRequest[]>('/change-requests');
      error = null;
    } catch (e) {
      error = (e as Error).message;
    } finally {
      if (!silent) loading = false;
    }
  }

  onMount(() => load());

  async function triage(cr: ChangeRequest, status: CRStatus) {
    triagingId = cr.id;
    try {
      await api.patch(`/change-requests/${cr.id}`, { status });
      await load({ silent: true });
    } catch (e) {
      error = (e as Error).message;
    } finally {
      triagingId = null;
    }
  }

  function onSaved() {
    // Sesi TIDAK ditutup di sini — user boleh lanjut chat / simpan lagi. Cukup
    // refresh daftar biar CR yang baru disimpan langsung muncul.
    load({ silent: true });
  }

  function fmtDate(iso: string): string {
    return new Date(iso).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' });
  }
</script>

<div class="section">
  <div class="section-title">
    Change request
    <span class="muted small">— usulan perubahan dari tim</span>
    {#if !showChat}
      <button class="sign-btn cr-new" on:click={beginSetup}>＋ Ajukan usulan</button>
    {/if}
  </div>

  {#if error}
    <p class="small" style="color:var(--win-red)">{error}</p>
  {/if}
</div>

{#if showChat}
  <ChangeRequestChat on:saved={onSaved} />
{/if}

<div class="section">
  {#if loading}
    <p class="small muted">Memuat...</p>
  {:else}
    <div class="cr-list">
      {#each items as cr (cr.id)}
        <div class="cr-item">
          <div class="cr-item-head">
            <span class="badge badge-{crStatusTone(cr.status)}">{crStatusLabel(cr.status)}</span>
            <span class="cr-item-by">{cr.submitted_by_name}</span>
            <span class="muted small">{fmtDate(cr.created_at)}</span>
            <div class="cr-item-actions">
              {#if cr.document_md}
                <button
                  class="quick-btn"
                  on:click={() => (expandedDoc = expandedDoc === cr.id ? null : cr.id)}>
                  {expandedDoc === cr.id ? 'Tutup dokumen' : 'Lihat dokumen'}
                </button>
              {/if}
              <button
                class="quick-btn"
                on:click={() => (expanded = expanded === cr.id ? null : cr.id)}>
                {expanded === cr.id ? 'Tutup' : 'Lihat percakapan'}
              </button>
            </div>
          </div>

          {#if cr.reviewed_by_name}
            <div class="muted small">
              Ditriase oleh {cr.reviewed_by_name}{cr.reviewed_at ? ` · ${fmtDate(cr.reviewed_at)}` : ''}
            </div>
          {/if}

          {#if expandedDoc === cr.id && cr.document_md}
            <div class="cr-doc-box">
              <div class="cr-md">{@html renderMarkdown(cr.document_md)}</div>
            </div>
          {/if}

          {#if expanded === cr.id}
            <div class="cr-doc-box">
              <div class="cr-md">{@html renderMarkdown(cr.raw_conversation)}</div>
            </div>
          {/if}

          {#if canTriage}
            <div class="cr-triage">
              <span class="muted small">Triase:</span>
              {#each CR_TRIAGE_TARGETS as target (target)}
                {#if target !== cr.status}
                  <button
                    class="quick-btn"
                    disabled={triagingId === cr.id}
                    on:click={() => triage(cr, target)}>{crStatusLabel(target)}</button>
                {/if}
              {/each}
            </div>
          {/if}
        </div>
      {/each}
      {#if items.length === 0}
        <div class="empty-state">
          <div class="empty-state-icon"><MessageSquare size={36} /></div>
          <div class="empty-state-text">Belum ada usulan perubahan. Klik "Ajukan usulan" buat mulai.</div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .cr-new {
    margin-left: auto;
  }
  .cr-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .cr-item {
    border: 1px solid var(--content-alt);
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    /* Flex item default min-width:auto -- tanpa ini, item melebar ngikutin
       baris terpanjang di dalam .cr-doc-box (dokumen/transcript) alih-alih
       teksnya yang wrap ke lebar container, bikin overflow ke kanan. */
    min-width: 0;
  }
  .cr-item-head {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .cr-item-by {
    font-weight: bold;
  }
  .cr-item-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
  }
  .cr-doc-box {
    background: var(--content-bg);
    border: 1px solid var(--content-alt);
    padding: 8px;
    max-height: 320px;
    overflow-y: auto;
    overflow-x: auto;
    font-size: 11px;
  }
  .cr-triage {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }
</style>
