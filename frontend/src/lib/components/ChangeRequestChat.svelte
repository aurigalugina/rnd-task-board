<script lang="ts">
  import { createEventDispatcher, tick } from 'svelte';
  import { api } from '$lib/api/client';
  import { chatApi, type BrowseResult } from '$lib/chatClient';
  import { buildTranscript, isSupportedImageType } from '$lib/chatMessages';
  import { renderMarkdown } from '$lib/markdown';
  import { Folder, FolderUp, Wrench, X, Paperclip, NotebookPen } from 'lucide-svelte';
  import {
    chatSession,
    startSession,
    sendPrompt,
    compile,
    interrupt,
    closeSession,
    cancelSetup,
    setCwd
  } from '$lib/stores/chatSessionStore';

  const dispatch = createEventDispatcher<{ saved: void }>();

  // State sesi (session/ws/messages/busy/cost) HIDUP di store level modul supaya
  // bertahan saat pindah menu — komponen ini cuma view. Yang lokal cuma UI
  // ephemeral: input box, dir picker, flag saving.
  let input = '';
  let saving = false;
  let savedNote = false;

  // Lampiran gambar (screenshot) yang belum dikirim. url = data URL buat
  // preview; data = base64 mentah buat dikirim ke WS.
  let pending: { url: string; media_type: string; data: string }[] = [];
  let attachError = '';
  let fileInput: HTMLInputElement;
  let textareaEl: HTMLTextAreaElement;
  let messagesEl: HTMLDivElement;
  const TEXTAREA_MAX_H = 160;

  // Auto-scroll ngikutin bawah SELAMA streaming juga (bukan cuma pas ada pesan
  // baru): store bikin array messages baru tiap potongan teks masuk, jadi
  // reactive di bawah nge-fire tiap chunk. Guard "near bottom" diukur SEBELUM
  // tick (DOM belum ke-update) — jadi kalau user lagi scroll ke atas baca pesan
  // lama, kita TIDAK paksa loncat ke bawah.
  async function scrollToBottom() {
    if (!messagesEl) return;
    const nearBottom = messagesEl.scrollHeight - messagesEl.scrollTop - messagesEl.clientHeight < 80;
    await tick();
    if (messagesEl && nearBottom) messagesEl.scrollTop = messagesEl.scrollHeight;
  }
  // Depend ke referensi array messages (berubah tiap chunk streaming) + busy.
  $: {
    $chatSession.messages;
    $chatSession.busy;
    scrollToBottom();
  }
  const MAX_IMG_BYTES = 5 * 1024 * 1024;

  function addFile(file: File) {
    if (!isSupportedImageType(file.type)) {
      attachError = `Tipe ${file.type || '?'} tidak didukung (png/jpeg/gif/webp).`;
      return;
    }
    if (file.size > MAX_IMG_BYTES) {
      attachError = 'Gambar terlalu besar (maks 5MB).';
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      const url = String(reader.result);
      pending = [...pending, { url, media_type: file.type, data: url.slice(url.indexOf(',') + 1) }];
      attachError = '';
    };
    reader.readAsDataURL(file);
  }

  function onFileChange(e: Event) {
    const el = e.currentTarget as HTMLInputElement;
    for (const f of Array.from(el.files ?? [])) addFile(f);
    el.value = '';
  }

  function onPaste(e: ClipboardEvent) {
    for (const it of Array.from(e.clipboardData?.items ?? [])) {
      if (it.kind === 'file') {
        const f = it.getAsFile();
        if (f) addFile(f);
      }
    }
  }

  function removePending(i: number) {
    pending = pending.filter((_, idx) => idx !== i);
  }

  // Dir picker cuma fallback kalau CHAT_DEFAULT_CWD belum di-set backend.
  let showPicker = false;
  let browseState: BrowseResult | null = null;

  async function openPicker() {
    showPicker = true;
    try {
      browseState = await chatApi.browse($chatSession.cwd || undefined);
    } catch (e) {
      browseState = null;
      console.error(e);
    }
  }

  async function browseTo(path: string) {
    try {
      browseState = await chatApi.browse(path);
    } catch (e) {
      console.error(e);
    }
  }

  function usePickedDir() {
    if (browseState) setCwd(browseState.path);
    showPicker = false;
  }

  async function onSubmit() {
    if (($chatSession.busy) || (!input.trim() && pending.length === 0)) return;
    sendPrompt(
      input,
      pending.map((p) => ({ media_type: p.media_type, data: p.data }))
    );
    input = '';
    pending = [];
    await tick();
    autoGrow(); // reset tinggi textarea setelah dikosongkan
  }

  // Enter = kirim, Shift+Enter = baris baru (biar user gampang kontrol redaksi).
  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      onSubmit();
    }
  }

  // Tinggi textarea ngikutin isi (auto-grow) sampai batas TEXTAREA_MAX_H, lalu
  // baru muncul scroll — biar textarea gak melebar gak terkontrol.
  function autoGrow() {
    if (!textareaEl) return;
    textareaEl.style.height = 'auto';
    textareaEl.style.height = Math.min(textareaEl.scrollHeight, TEXTAREA_MAX_H) + 'px';
  }

  async function saveChangeRequest() {
    const transcript = buildTranscript($chatSession.messages);
    if (!transcript.trim()) return;
    saving = true;
    savedNote = false;
    try {
      // document_md = respons terakhir Claude abis "Susun change request"
      // (terpisah dari raw_conversation/transcript penuh) -- null kalau user
      // belum pernah klik itu, list-page cukup nampilin transcript-nya saja.
      await api.post('/change-requests', {
        raw_conversation: transcript,
        document_md: $chatSession.compiledDocument
      });
      savedNote = true;
      dispatch('saved'); // biar list di halaman ke-refresh; sesi TETAP hidup
    } catch (e) {
      console.error(e);
    } finally {
      saving = false;
    }
  }
</script>

<div class="section cr-chat">
  <div class="section-title">
    Ajukan usulan perubahan
    {#if $chatSession.step === 'chatting'}
      <button class="quick-btn cr-close" on:click={closeSession}>Akhiri sesi</button>
    {:else}
      <button class="quick-btn cr-close" on:click={cancelSetup}>Batal</button>
    {/if}
  </div>

  {#if $chatSession.error}
    <p class="small" style="color:var(--win-red)">{$chatSession.error}</p>
  {/if}

  {#if $chatSession.step === 'setup'}
    <div class="cr-setup">
      {#if $chatSession.configuredCwd}
        <span class="small muted">Direktori project (dari konfigurasi)</span>
        <div class="cr-cwd-fixed mono">{$chatSession.cwd}</div>
      {:else}
        <label class="small muted" for="cr-cwd">Direktori project</label>
        <p class="small" style="color:var(--win-amber)">
          CHAT_DEFAULT_CWD belum di-set di backend — pilih direktori manual di bawah.
        </p>
        <div class="cr-cwd-row">
          <input
            id="cr-cwd"
            class="inline-input"
            value={$chatSession.cwd}
            on:input={(e) => setCwd(e.currentTarget.value)}
            placeholder="/path/ke/repo" />
          <button class="quick-btn" on:click={openPicker}>Pilih…</button>
        </div>

        {#if showPicker && browseState}
          <div class="cr-picker">
            <div class="cr-picker-head">
              <span class="mono small">{browseState.path}</span>
              <button class="quick-btn" on:click={() => browseState && browseTo(browseState.parent)}
                ><FolderUp size={13} />&nbsp;Naik</button
              >
            </div>
            <div class="cr-picker-list">
              {#each browseState.directories as d (d.path)}
                <button class="cr-picker-item" on:click={() => browseTo(d.path)}><Folder size={13} />&nbsp;{d.name}</button>
              {/each}
              {#if browseState.directories.length === 0}
                <div class="empty-note">Tidak ada subfolder.</div>
              {/if}
            </div>
            <button class="sign-btn" on:click={usePickedDir}>Pakai direktori ini</button>
          </div>
        {/if}
      {/if}

      <button class="sign-btn" on:click={startSession} disabled={$chatSession.starting || !$chatSession.cwd}>
        {$chatSession.starting ? 'Memulai…' : 'Mulai percakapan'}
      </button>
      <p class="muted small">Mode read-only (plan) — assistant hanya membaca repo, tidak mengubah file.</p>
    </div>
  {:else if $chatSession.step === 'chatting'}
    <div class="cr-messages" bind:this={messagesEl}>
      {#each $chatSession.messages as m, i (i)}
        {#if m.role === 'tool'}
          <div class="cr-note cr-note-tool"><Wrench size={12} />&nbsp;{m.name}</div>
        {:else if m.role === 'system'}
          <div class="cr-note">{m.text}</div>
        {:else}
          <div class="cr-row cr-row-{m.role}">
            <div class="cr-bubble cr-bubble-{m.role}">
              {#if m.text}
                {#if m.role === 'assistant'}
                  <!-- Balasan Claude dirender sebagai markdown (sudah disanitasi
                       DOMPurify di renderMarkdown). Bubble user tetap plain text. -->
                  <div class="cr-bubble-text cr-md">{@html renderMarkdown(m.text)}</div>
                {:else}
                  <div class="cr-bubble-text">{m.text}</div>
                {/if}
              {/if}
              {#if m.role === 'user' && m.images}
                <div class="cr-msg-images">
                  {#each m.images as src, k (k)}
                    <img class="cr-thumb" {src} alt="lampiran" />
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        {/if}
      {/each}
      {#if $chatSession.busy}
        <div class="cr-row cr-row-assistant">
          <div class="cr-bubble cr-bubble-assistant cr-typing">Assistant sedang mengetik…</div>
        </div>
      {/if}
    </div>

    {#if pending.length > 0}
      <div class="cr-attachments">
        {#each pending as p, i (i)}
          <div class="cr-attach">
            <img src={p.url} alt="preview lampiran" />
            <button type="button" class="cr-attach-x" on:click={() => removePending(i)} aria-label="Hapus lampiran"><X size={11} /></button>
          </div>
        {/each}
      </div>
    {/if}
    {#if attachError}<span class="small" style="color:var(--win-red)">{attachError}</span>{/if}

    <form class="cr-input-row" on:submit|preventDefault={onSubmit}>
      <textarea
        class="cr-textarea"
        bind:this={textareaEl}
        bind:value={input}
        on:input={autoGrow}
        on:keydown={onKeydown}
        on:paste={onPaste}
        rows="1"
        placeholder="Tulis usulan… (Enter = kirim · Shift+Enter = baris baru · bisa tempel screenshot)"
        disabled={$chatSession.busy}></textarea>
      <input
        type="file"
        accept="image/png,image/jpeg,image/gif,image/webp"
        multiple
        bind:this={fileInput}
        on:change={onFileChange}
        style="display:none" />
      <div class="cr-input-btns">
        <button type="button" class="quick-btn" title="Lampirkan gambar" on:click={() => fileInput?.click()} disabled={$chatSession.busy}><Paperclip size={13} /></button>
        {#if $chatSession.busy}
          <button type="button" class="quick-btn" on:click={interrupt}>Stop</button>
        {:else}
          <button type="submit" class="sign-btn" disabled={!input.trim() && pending.length === 0}>Kirim</button>
        {/if}
      </div>
    </form>

    <div class="cr-actions">
      {#if savedNote}<span class="small" style="color:var(--win-green)">Tersimpan</span>{/if}
      {#if $chatSession.compiledDocument && !savedNote}
        <span class="small muted">Dokumen change request siap disimpan</span>
      {/if}
      <button class="quick-btn" on:click={compile} disabled={$chatSession.busy}><NotebookPen size={13} />&nbsp;Susun change request</button>
      <button class="sign-btn" on:click={saveChangeRequest} disabled={saving || $chatSession.busy}>
        {saving ? 'Menyimpan…' : 'Simpan sebagai change request'}
      </button>
    </div>
  {/if}
</div>

<style>
  .cr-chat {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: 760px;
  }
  .cr-close {
    margin-left: auto;
  }
  .cr-setup {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .cr-cwd-fixed {
    padding: 4px 6px;
    background: var(--content-alt);
    border: 1px solid var(--content-alt);
    word-break: break-all;
  }
  .cr-cwd-row {
    display: flex;
    gap: 6px;
  }
  .cr-cwd-row .inline-input {
    flex: 1;
  }
  .cr-picker {
    border: 1px solid var(--content-alt);
    padding: 6px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .cr-picker-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }
  .cr-picker-list {
    max-height: 180px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }
  .cr-picker-item {
    text-align: left;
    background: none;
    border: none;
    padding: 3px 4px;
    cursor: pointer;
    font: inherit;
    color: var(--text-primary);
  }
  .cr-picker-item:hover {
    background: var(--content-alt);
  }
  .cr-messages {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 46vh;
    overflow-y: auto;
    border: 1px solid var(--content-alt);
    padding: 10px;
    background: var(--content-bg);
  }
  /* Bubble chat: user rata kanan (aksen), assistant rata kiri (netral). */
  .cr-row {
    display: flex;
  }
  .cr-row-user {
    justify-content: flex-end;
  }
  .cr-row-assistant {
    justify-content: flex-start;
  }
  .cr-bubble {
    max-width: 78%;
    padding: 6px 9px;
    border-radius: 10px;
    line-height: 1.45;
  }
  .cr-bubble-text {
    white-space: pre-wrap;
    word-break: break-word;
  }
  .cr-bubble-user {
    background: var(--win-blue);
    color: #fff;
    border-bottom-right-radius: 2px;
  }
  .cr-bubble-assistant {
    background: var(--content-alt);
    color: var(--text-primary);
    border: 1px solid var(--content-alt);
    border-bottom-left-radius: 2px;
  }
  .cr-typing {
    color: var(--text-muted);
    font-style: italic;
  }
  /* .cr-md (markdown viewer) DIPINDAH ke app.css sebagai style global --
     dipakai bareng di sini (bubble assistant) DAN routes/change-requests
     (transcript + dokumen change_request.md), Svelte scoped style gak nembus
     lintas komponen. Lihat app.css kalau mau ubah tampilan markdown-nya. */
  /* Baris info non-percakapan (tool call / status) — di tengah, samar. */
  .cr-note {
    align-self: center;
    font-size: 10px;
    color: var(--text-muted);
    font-style: italic;
  }
  .cr-note-tool {
    font-family: monospace;
    font-style: normal;
  }
  .cr-msg-images {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 2px;
  }
  .cr-thumb {
    max-width: 160px;
    max-height: 120px;
    border: 1px solid var(--content-alt);
    object-fit: contain;
  }
  .cr-attachments {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .cr-attach {
    position: relative;
  }
  .cr-attach img {
    max-width: 72px;
    max-height: 72px;
    border: 1px solid var(--content-alt);
    object-fit: cover;
    display: block;
  }
  .cr-attach-x {
    position: absolute;
    top: -6px;
    right: -6px;
    width: 16px;
    height: 16px;
    line-height: 14px;
    padding: 0;
    font-size: 10px;
    border: 1px solid var(--content-alt);
    background: var(--face);
    color: var(--text-primary);
    cursor: pointer;
  }
  .cr-input-row {
    display: flex;
    gap: 6px;
    align-items: flex-end;
  }
  .cr-textarea {
    flex: 1;
    resize: none;
    overflow-y: auto;
    min-height: 30px;
    max-height: 160px;
    padding: 6px 8px;
    font: inherit;
    line-height: 1.4;
    color: var(--text-primary);
    background: var(--content-bg);
    border: 1px solid var(--content-alt);
    box-sizing: border-box;
  }
  .cr-input-btns {
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .cr-actions {
    display: flex;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
  }
</style>
