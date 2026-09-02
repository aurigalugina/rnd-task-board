<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { AssignableUser, TeamTodayEntry, TeamTodayUser } from '$lib/types';
  import Avatar from '$lib/components/Avatar.svelte';
  import DayProgressStatus from '$lib/components/DayProgressStatus.svelte';
  import { CalendarDays, Pencil, Check, X } from 'lucide-svelte';

  // "Apa yang sedang dikerjakan tim hari ini" -- POV per orang, transparan
  // (semua user terautentikasi bisa lihat semua orang, sama seperti
  // Dashboard). Lihat decision-log-team-today-menu-20260901.md.
  //
  // 2026-09-02: inline edit (Rencana/Realisasi/Progress/Blocker langsung di
  // sini, PATCH ke /day-entries/{id} yang sama dipakai DailyTaskPanel --
  // TIDAK ada endpoint/permission baru, cuma UX shortcut) + filter per user
  // (default diri sendiri) -- lihat decision-log-team-today-inline-edit-20260902.md.
  const today = new Date().toLocaleDateString('en-CA'); // en-CA = YYYY-MM-DD
  let date = today;
  let users: TeamTodayUser[] = [];
  let loading = true;
  let error: string | null = null;

  // Filter user -- default HANYA diri sendiri (permintaan user), '' = semua
  // orang. Daftar dropdown dari /users/assignable (sama sumbernya dengan
  // picker PIC/reviewer di tempat lain).
  let assignableUsers: AssignableUser[] = [];
  let selectedUserId = $auth.user?.id ?? '';
  $: filteredUsers = selectedUserId ? users.filter((u) => u.user_id === selectedUserId) : users;

  // Gate tombol edit -- sinkron dengan permission backend
  // (canEditDayEntry di dailytask/handler.go): pemilik entry SELALU boleh;
  // selain itu HANYA kalau viewer punya akses "lihat semua orang"
  // (super_user atau task_scope_visibility='team'). Dicek di CLIENT supaya
  // tombol pensil tidak muncul utk kasus yang pasti ditolak backend (403) --
  // pure UX, backend tetap validasi ulang (defense in depth).
  $: canSeeAllPeople =
    $auth.user?.access_level === 'super_user' || $auth.user?.task_scope_visibility === 'team';
  function canEditEntry(ownerUserId: string): boolean {
    return ownerUserId === $auth.user?.id || canSeeAllPeople;
  }

  $: isToday = date === today;

  async function load() {
    loading = true;
    error = null;
    try {
      users = await api.get<TeamTodayUser[]>(`/team-today?date=${date}`);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    assignableUsers = await api.get<AssignableUser[]>('/users/assignable').catch(() => []);
    await load();
  });
  $: if (date) load();

  function shiftDay(delta: number) {
    const d = new Date(date + 'T00:00:00');
    d.setDate(d.getDate() + delta);
    date = d.toLocaleDateString('en-CA');
  }

  function progressBadge(pct: number): { label: string; cls: string } {
    if (pct === 100) return { label: 'Selesai', cls: 'badge-good' };
    if (pct > 0) return { label: `${pct}%`, cls: 'badge-accent' };
    return { label: 'Belum', cls: 'badge-neutral' };
  }

  function formatDateLabel(iso: string): string {
    return new Date(iso + 'T00:00:00').toLocaleDateString('id-ID', {
      weekday: 'long',
      day: 'numeric',
      month: 'long',
      year: 'numeric'
    });
  }

  // --- Inline edit ---
  // Draft KEYED by day_entry_id -- cuma satu entry aktif diedit dalam satu
  // waktu (klik entry lain otomatis batal draft yang sedang jalan, tidak ada
  // multi-edit bersamaan yang bisa bikin bingung mana yang belum disimpan).
  let editingId: string | null = null;
  let draftPlanned = '';
  let draftRealisasi = '';
  let draftProgress = 0;
  let draftBlocker = '';
  let saving = false;
  let saveError = '';

  function startEdit(e: TeamTodayEntry) {
    editingId = e.day_entry_id;
    draftPlanned = e.planned_text;
    draftRealisasi = e.realisasi_text;
    draftProgress = e.progress_pct;
    draftBlocker = e.blocker_text;
    saveError = '';
  }

  function cancelEdit() {
    editingId = null;
    saveError = '';
  }

  async function saveEdit(e: TeamTodayEntry) {
    saving = true;
    saveError = '';
    try {
      await api.patch(`/day-entries/${e.day_entry_id}`, {
        planned_text: draftPlanned,
        realisasi_text: draftRealisasi,
        progress_pct: draftProgress,
        blocker_text: draftProgress === 100 ? '' : draftBlocker
      });
      // Update in-place -- hindari full reload (flicker + kehilangan posisi
      // scroll), cukup patch objek entry yang barusan disimpan.
      e.planned_text = draftPlanned;
      e.realisasi_text = draftRealisasi;
      e.progress_pct = draftProgress;
      e.blocker_text = draftProgress === 100 ? '' : draftBlocker;
      users = users;
      editingId = null;
    } catch (err) {
      saveError = (err as Error).message;
    } finally {
      saving = false;
    }
  }
</script>

<div class="week-nav">
  <button class="week-nav-arrow" on:click={() => shiftDay(-1)} aria-label="Hari sebelumnya">
    &lsaquo;
  </button>
  <div class="week-nav-range">
    <span class="week-nav-dates">{formatDateLabel(date)}</span>
    {#if isToday}<span class="week-nav-label muted small">Hari ini</span>{/if}
  </div>
  <button class="week-nav-arrow" on:click={() => shiftDay(1)} aria-label="Hari berikutnya">
    &rsaquo;
  </button>
</div>

<div class="team-today-toolbar">
  {#if !isToday}
    <button class="quick-btn" on:click={() => (date = today)}>
      <CalendarDays size={13} />&nbsp;Kembali ke hari ini
    </button>
  {/if}

  <div class="team-today-filter">
    <span class="small muted">Tampilkan:</span>
    <select class="inline-input" bind:value={selectedUserId} style="width:auto">
      <option value={$auth.user?.id ?? ''}>Saya ({$auth.user?.display_name})</option>
      <option value="">Semua orang</option>
      {#each assignableUsers.filter((u) => u.id !== $auth.user?.id) as u (u.id)}
        <option value={u.id}>{u.display_name}</option>
      {/each}
    </select>
  </div>
</div>

{#if loading}
  <p>Memuat...</p>
{:else if error}
  <p class="small" style="color:var(--win-red)">{error}</p>
{:else}
  <div class="team-today-list">
    {#each filteredUsers as u (u.user_id)}
      <div class="section team-today-card">
        <div class="team-today-head">
          <Avatar initials={u.initials} size={28} title={u.display_name} />
          <div>
            <div class="panel-row-name">{u.display_name}</div>
            <div class="muted small">{u.org_team}</div>
          </div>
          <div class="team-today-count muted small">
            {u.entries.length} task
          </div>
        </div>

        {#if u.entries.length === 0}
          <div class="empty-note">Belum ada update di tanggal ini.</div>
        {:else}
          <div class="team-today-entries">
            {#each u.entries as e (e.day_entry_id)}
              {@const pb = progressBadge(e.progress_pct)}
              {@const isEditing = editingId === e.day_entry_id}
              <div class="team-today-entry" class:team-today-entry-editing={isEditing}>
                <div class="team-today-entry-top">
                  <a class="board-link small" href="/boards?board={e.board_id}">{e.board_name}</a>
                  <span class="muted small">— {e.big_task_name} — {e.daily_task_title}</span>
                  {#if !isEditing}
                    <span class="badge {pb.cls}" style="margin-left:auto">{pb.label}</span>
                    {#if canEditEntry(u.user_id)}
                      <button
                        class="icon-btn team-today-edit-btn"
                        aria-label="Update task ini"
                        title="Update Rencana/Realisasi/Progress"
                        on:click={() => startEdit(e)}
                      >
                        <Pencil size={12} />
                      </button>
                    {/if}
                  {/if}
                </div>

                {#if isEditing}
                  <div class="team-today-entry-body">
                    <div class="panel-field" style="padding:0">
                      <label class="small muted" for="tt-planned-{e.day_entry_id}">Rencana</label>
                      <textarea
                        id="tt-planned-{e.day_entry_id}"
                        class="inline-input inline-textarea"
                        rows="2"
                        placeholder="Tulis rencana untuk hari ini..."
                        bind:value={draftPlanned}
                      />
                    </div>
                    <div class="panel-field" style="padding:0">
                      <label class="small muted" for="tt-realisasi-{e.day_entry_id}">Realisasi</label>
                      <textarea
                        id="tt-realisasi-{e.day_entry_id}"
                        class="inline-input inline-textarea"
                        rows="2"
                        placeholder="Apa yang benar-benar dikerjakan/dicapai hari ini..."
                        bind:value={draftRealisasi}
                      />
                    </div>
                    <div class="panel-field" style="padding:0">
                      <span class="small muted">Progress</span>
                      <DayProgressStatus
                        progressPct={draftProgress}
                        onChange={(next) => (draftProgress = next)}
                      />
                    </div>
                    {#if draftProgress !== 100}
                      <div class="panel-field" style="padding:0">
                        <label class="small muted" for="tt-blocker-{e.day_entry_id}">Blocker / catatan lanjutan</label>
                        <textarea
                          id="tt-blocker-{e.day_entry_id}"
                          class="inline-input inline-textarea"
                          rows="2"
                          placeholder="Ada hambatan? Rencana hari berikutnya?"
                          bind:value={draftBlocker}
                        />
                      </div>
                    {/if}
                    {#if saveError}<p class="small" style="color:var(--win-red);margin:0">{saveError}</p>{/if}
                    <div class="inline-form-actions">
                      <button class="quick-btn quick-btn-done" on:click={() => saveEdit(e)} disabled={saving}>
                        <Check size={12} />&nbsp;{saving ? 'Menyimpan...' : 'Simpan'}
                      </button>
                      <button class="quick-btn" on:click={cancelEdit} disabled={saving}>
                        <X size={12} />&nbsp;Batal
                      </button>
                    </div>
                  </div>
                {:else}
                  <div class="team-today-entry-body">
                    <div class="team-today-field">
                      <span class="small muted">Rencana</span>
                      <span class="small">{e.planned_text || '—'}</span>
                    </div>
                    <div class="team-today-field">
                      <span class="small muted">Realisasi</span>
                      <span class="small">{e.realisasi_text || '—'}</span>
                    </div>
                    {#if e.blocker_text}
                      <div class="team-today-field">
                        <span class="small" style="color:var(--win-red)">Blocker</span>
                        <span class="small">{e.blocker_text}</span>
                      </div>
                    {/if}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/each}
    {#if filteredUsers.length === 0}
      <p class="empty-note">Tidak ada user.</p>
    {/if}
  </div>
{/if}

<style>
  .team-today-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 10px;
  }

  .team-today-filter {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
  }

  .team-today-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .team-today-card {
    margin-bottom: 0;
  }

  .team-today-head {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .team-today-count {
    margin-left: auto;
  }

  .team-today-entries {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .team-today-entry {
    background: var(--content-alt);
    border: 1px solid var(--np-border, var(--win-border, #ccc));
    border-radius: 3px;
    padding: 8px 10px;
  }

  .team-today-entry-editing {
    border-color: var(--win-blue);
    box-shadow: 0 0 0 1px var(--win-blue);
  }

  .team-today-entry-top {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
    margin-bottom: 6px;
  }

  .team-today-edit-btn {
    flex-shrink: 0;
  }

  .team-today-entry-body {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .team-today-field {
    display: flex;
    gap: 6px;
  }

  .team-today-field > .muted {
    min-width: 64px;
    flex-shrink: 0;
  }
</style>
