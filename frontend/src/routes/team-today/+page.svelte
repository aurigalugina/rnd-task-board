<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { TeamTodayUser } from '$lib/types';
  import Avatar from '$lib/components/Avatar.svelte';
  import { CalendarDays } from 'lucide-svelte';

  // "Apa yang sedang dikerjakan tim hari ini" -- POV per orang, transparan
  // (semua user terautentikasi bisa lihat semua orang, sama seperti
  // Dashboard). Lihat decision-log-team-today-menu-20260901.md.
  const today = new Date().toLocaleDateString('en-CA'); // en-CA = YYYY-MM-DD
  let date = today;
  let users: TeamTodayUser[] = [];
  let loading = true;
  let error: string | null = null;

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

  onMount(load);
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

{#if !isToday}
  <button class="quick-btn" style="margin-bottom:10px" on:click={() => (date = today)}>
    <CalendarDays size={13} />&nbsp;Kembali ke hari ini
  </button>
{/if}

{#if loading}
  <p>Memuat...</p>
{:else if error}
  <p class="small" style="color:var(--win-red)">{error}</p>
{:else}
  <div class="team-today-list">
    {#each users as u (u.user_id)}
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
              <div class="team-today-entry">
                <div class="team-today-entry-top">
                  <a class="board-link small" href="/boards?board={e.board_id}">{e.board_name}</a>
                  <span class="muted small">— {e.big_task_name} — {e.daily_task_title}</span>
                  <span class="badge {pb.cls}" style="margin-left:auto">{pb.label}</span>
                </div>
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
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/each}
    {#if users.length === 0}
      <p class="empty-note">Tidak ada user.</p>
    {/if}
  </div>
{/if}

<style>
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

  .team-today-entry-top {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
    margin-bottom: 6px;
  }

  .team-today-entry-body {
    display: flex;
    flex-direction: column;
    gap: 4px;
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
