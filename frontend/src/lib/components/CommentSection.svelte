<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import { applyMention, mentionQuery, renderCommentHtml } from '$lib/comments';
  import type { AssignableUser, Comment } from '$lib/types';

  export let bigTaskId: string;
  // { id, title } daily task di bawah Big Task ini — buat pill filter/scope,
  // dikirim dari BigTaskList (DailyTaskPanel yang fetch daftar aslinya).
  export let dailyTasks: { id: string; title: string }[] = [];
  // Dipicu tombol "Komentar" di DailyTaskPanel: jumpScope = id daily task
  // tujuan, jumpToken increment tiap klik (biar reactive block ke-trigger
  // walau jumpScope-nya sama seperti sebelumnya).
  export let jumpScope: string | null = null;
  export let jumpToken = 0;

  let comments: Comment[] = [];
  let assignableUsers: AssignableUser[] = [];
  let loading = true;
  let error: string | null = null;

  let filterScope: 'all' | 'general' | string = 'all';
  let composeScope: 'general' | string = 'general';
  let text = '';
  let showMentions = false;
  let sending = false;
  let sectionEl: HTMLDivElement;
  let inputEl: HTMLTextAreaElement;

  $: authorNameById = Object.fromEntries(assignableUsers.map((u) => [u.id, u.display_name]));
  $: mentionNames = assignableUsers.map((u) => u.display_name);
  $: query = mentionQuery(text);
  $: mentionSuggestions =
    query !== null ? assignableUsers.filter((u) => u.display_name.toLowerCase().startsWith(query.toLowerCase())) : [];

  // silent: dipakai buat refresh SETELAH kirim komentar -- kalau loading
  // di-toggle, {#if loading} bikin seluruh daftar komentar flash "Memuat"
  // tiap kali (kerasa kayak "refresh" walau SPA, dilaporkan user 2026-08-10).
  async function load({ silent = false }: { silent?: boolean } = {}) {
    if (!silent) loading = true;
    try {
      const [c, users] = await Promise.all([
        api.get<Comment[]>(`/big-tasks/${bigTaskId}/comments?scope=all`),
        assignableUsers.length ? Promise.resolve(assignableUsers) : api.get<AssignableUser[]>('/users/assignable')
      ]);
      comments = c;
      assignableUsers = users;
    } catch (e) {
      error = (e as Error).message;
    } finally {
      if (!silent) loading = false;
    }
  }

  onMount(() => load());

  $: if (jumpToken > 0) jumpTo(jumpScope);

  async function jumpTo(scope: string | null) {
    if (!scope) return;
    filterScope = scope;
    composeScope = scope;
    await tick();
    sectionEl?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    inputEl?.focus();
  }

  function scopeLabel(dailyTaskId: string | null): string {
    if (!dailyTaskId) return 'Umum';
    return dailyTasks.find((d) => d.id === dailyTaskId)?.title ?? 'Umum';
  }

  $: scopedComments = comments
    .filter((c) => {
      if (filterScope === 'all') return true;
      if (filterScope === 'general') return !c.daily_task_id;
      return c.daily_task_id === filterScope;
    })
    .slice()
    .sort((a, b) => (a.created_at < b.created_at ? 1 : -1));

  function pickMention(name: string) {
    text = applyMention(text, name);
    inputEl?.focus();
  }

  async function send() {
    if (!text.trim()) return;
    sending = true;
    try {
      await api.post(`/big-tasks/${bigTaskId}/comments`, {
        daily_task_id: composeScope === 'general' ? null : composeScope,
        body: text.trim()
      });
      text = '';
      await load({ silent: true });
    } catch (e) {
      error = (e as Error).message;
    } finally {
      sending = false;
    }
  }
</script>

<div class="comment-section" bind:this={sectionEl}>
  <div class="section-title">Komentar</div>

  <div class="comment-filter-row">
    <button class="role-filter-pill {filterScope === 'all' ? 'role-filter-pill-active' : ''}" on:click={() => (filterScope = 'all')}>
      Semua
    </button>
    <button class="role-filter-pill {filterScope === 'general' ? 'role-filter-pill-active' : ''}" on:click={() => (filterScope = 'general')}>
      Umum
    </button>
    {#each dailyTasks as dt (dt.id)}
      <button class="role-filter-pill {filterScope === dt.id ? 'role-filter-pill-active' : ''}" on:click={() => (filterScope = dt.id)}>
        {dt.title}
      </button>
    {/each}
  </div>

  {#if loading}
    <p class="small muted">Memuat komentar...</p>
  {:else}
    {#if error}<p class="small" style="color:var(--win-red)">{error}</p>{/if}
    <div class="comment-list">
      {#each scopedComments as c (c.id)}
        <div class="comment-row">
          <div class="avatar" style="width:22px;height:22px;font-size:9px">
            {authorNameById[c.author_id]?.slice(0, 2).toUpperCase() ?? '?'}
          </div>
          <div class="comment-body">
            <div class="comment-meta">
              <span class="comment-author">{authorNameById[c.author_id] ?? c.author_id}</span>
              <span class="comment-scope-tag">{scopeLabel(c.daily_task_id)}</span>
              <span class="mono small muted">{c.created_at.slice(0, 10)}</span>
            </div>
            <div class="comment-text">{@html renderCommentHtml(c.body, mentionNames)}</div>
          </div>
        </div>
      {/each}
      {#if scopedComments.length === 0}
        <div class="empty-note">Belum ada komentar di scope ini.</div>
      {/if}
    </div>

    <div class="comment-composer">
      <div class="comment-composer-row">
        <select class="inline-input" bind:value={composeScope}>
          <option value="general">Umum (level big task)</option>
          {#each dailyTasks as dt (dt.id)}
            <option value={dt.id}>{dt.title}</option>
          {/each}
        </select>
      </div>
      <div class="comment-composer-row comment-input-wrap">
        <div class="comment-input-relative">
          <textarea
            bind:this={inputEl}
            class="inline-input comment-textarea"
            rows="3"
            placeholder="Tulis komentar... ketik @ buat mention&#10;(Enter = kirim, Shift+Enter = baris baru)"
            bind:value={text}
            on:focus={() => (showMentions = true)}
            on:blur={() => setTimeout(() => (showMentions = false), 150)}
            on:keydown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); } }}
          />
          {#if showMentions && query !== null && mentionSuggestions.length > 0}
            <div class="mention-popup">
              {#each mentionSuggestions as u (u.id)}
                <!-- svelte-ignore a11y-no-static-element-interactions -->
                <button class="mention-option" on:mousedown|preventDefault={() => pickMention(u.display_name)}>
                  <div class="avatar" style="width:18px;height:18px;font-size:8px">{u.initials}</div>
                  <span>{u.display_name}</span>
                  <span class="muted small">{u.roles.join('/')}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
        <button class="quick-btn quick-btn-done" on:click={send} disabled={sending}>
          {sending ? 'Mengirim...' : 'Kirim'}
        </button>
      </div>
    </div>
  {/if}
</div>
