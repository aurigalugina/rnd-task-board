<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/authStore';
  import type { AssignableUser, Board, BigTask, UserProgressSummary } from '$lib/types';
  import VerdictBadge from '$lib/components/VerdictBadge.svelte';
  import DualBar from '$lib/components/DualBar.svelte';
  import StatCard from '$lib/components/StatCard.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import DonutChart from '$lib/components/DonutChart.svelte';
  import GroupedBarChart from '$lib/components/GroupedBarChart.svelte';
  import { aggregateBoards, computeDashboardStats, truncateName, type BoardWithTasks } from '$lib/dashboardStats';

  let boardGroups: BoardWithTasks[] = [];
  let team: AssignableUser[] = [];
  let progressSummaries: UserProgressSummary[] = [];
  let loading = true;
  let error: string | null = null;

  // Dashboard mengagregasi LINTAS SEMUA board di client — tidak ada endpoint
  // backend baru buat ini, cukup fetch big-tasks tiap board lalu digabung.
  // POV Dashboard adalah per BOARD (=project), bukan per Big Task — lihat
  // docs/decision-log/decision-log-dashboard-board-level-aggregation-20260810.md.
  onMount(async () => {
    try {
      const [boards, users, summaries] = await Promise.all([
        api.get<Board[]>('/boards'),
        api.get<AssignableUser[]>('/users/assignable'),
        api.get<UserProgressSummary[]>('/users/progress-summary')
      ]);
      team = users;
      progressSummaries = summaries;
      boardGroups = await Promise.all(
        boards.map(async (b) => {
          const bigTasks = await api.get<BigTask[]>(`/boards/${b.id}/big-tasks`);
          return { boardId: b.id, boardName: b.name, bigTasks };
        })
      );
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  });

  $: isSuperUser = $auth.user?.access_level === 'super_user';
  $: progressByUserId = Object.fromEntries(progressSummaries.map((s) => [s.user_id, s]));

  $: stats = computeDashboardStats(aggregateBoards(boardGroups));
  $: ({
    total,
    notStarted,
    running,
    done,
    hold,
    won,
    lose,
    completionRate,
    activeBoards,
    loseBoards,
    nearestDeadline
  } = stats);

  // Browser modern resolve CSS custom property langsung di attribute SVG
  // (stroke="var(--x)"), jadi warna cukup dirujuk lewat token tema, ikut
  // berubah otomatis kalau tema diganti — tidak perlu resolve manual.
  $: statusChartData = [
    { name: 'Sedang berjalan', value: running, color: 'var(--win-green)' },
    { name: 'Belum berjalan', value: notStarted, color: 'var(--text-muted)' },
    { name: 'Sudah selesai', value: done, color: 'var(--win-blue)' },
    { name: 'Di hold', value: hold, color: 'var(--win-amber)' }
  ];
  $: hasilChartData = [
    { name: 'Won', value: won, color: 'var(--win-green)' },
    { name: 'Lose', value: lose, color: 'var(--win-red)' }
  ];
  $: progressChartData = activeBoards.map((b) => ({
    name: truncateName(b.boardName),
    actual: b.avgActualPct,
    expected: b.avgExpectedPct
  }));

  // VerdictBadge cuma kenal Verdict big task ('on_progress'|'win'|'lose') —
  // map dari BoardVerdict ('won'|'lose'|'neutral') tanpa ubah komponennya.
  function toBadgeVerdict(v: 'won' | 'lose' | 'neutral'): 'on_progress' | 'win' | 'lose' {
    return v === 'won' ? 'win' : v === 'lose' ? 'lose' : 'on_progress';
  }
</script>

<div class="dash-heading">
  <span class="dash-heading-title">Project tracking dashboard — R&amp;D</span>
  <span class="muted small">Live dari semua board dan big task</span>
</div>

{#if loading}
  <p>Memuat...</p>
{:else if error}
  <p class="small" style="color:var(--win-red)">Gagal memuat data: {error}</p>
{:else}
  <div class="stat-row stat-row-8">
    <StatCard label="Total project" value={total} />
    <StatCard label="Belum berjalan" value={notStarted} />
    <StatCard label="Sedang berjalan" value={running} />
    <StatCard label="Sudah selesai" value={done} />
    <StatCard label="Di hold" value={hold} />
    <StatCard label="Won" value={won} tone="good" />
    <StatCard label="Lose" value={lose} tone="warn" />
    <StatCard label="Completion rate" value="{completionRate}%" tone="accent" />
  </div>

  <div class="two-col">
    <div class="section">
      <div class="section-title">Status project</div>
      <div class="chart-row">
        <DonutChart data={statusChartData} />
        <div class="chart-legend">
          {#each statusChartData as d (d.name)}
            <div class="chart-legend-row">
              <span class="chart-legend-swatch" style="background:{d.color}" />
              <span class="small">{d.name}</span>
              <span class="mono small">{d.value}</span>
            </div>
          {/each}
        </div>
      </div>
    </div>
    <div class="section">
      <div class="section-title">Hasil project</div>
      <div class="chart-row">
        <DonutChart data={hasilChartData} />
        <div class="chart-legend">
          {#each hasilChartData as d (d.name)}
            <div class="chart-legend-row">
              <span class="chart-legend-swatch" style="background:{d.color}" />
              <span class="small">{d.name}</span>
              <span class="mono small">{d.value}</span>
            </div>
          {/each}
        </div>
      </div>
    </div>
  </div>

  <div class="section">
    <div class="section-title">Progress: actual vs expected (%)</div>
    <GroupedBarChart data={progressChartData} />
    <div class="attention-list" style="margin-top:10px">
      {#each activeBoards as b (b.boardId)}
        <div class="attention-row">
          <div class="attention-main">
            <span class="attention-name">{b.boardName}</span>
            <span class="muted small">{b.totalBigTasks} big task</span>
          </div>
          <div class="attention-bar">
            <DualBar expected={b.avgExpectedPct} actual={b.avgActualPct} />
          </div>
          <VerdictBadge verdict={toBadgeVerdict(b.verdict)} />
          <span class="mono small {b.daysLeft < 0 ? 'days-late' : 'muted'}">
            {b.daysLeft < 0 ? `telat ${-b.daysLeft}h` : `${b.daysLeft}h lagi`}
          </span>
        </div>
      {/each}
      {#if activeBoards.length === 0}
        <div class="empty-note">Tidak ada project aktif (semua sudah selesai, atau belum ada project).</div>
      {/if}
    </div>
  </div>

  <div class="two-col">
    <div class="section">
      <div class="section-title">Deadline terdekat</div>
      <table class="sheet-table">
        <thead><tr><th>Nama project</th><th>Sisa hari</th><th>Actual</th></tr></thead>
        <tbody>
          {#each nearestDeadline as b (b.boardId)}
            <tr class:row-overdue={b.daysLeft < 0}>
              <td>{b.boardName}</td>
              <td class="mono {b.daysLeft < 0 ? 'days-late' : ''}">{b.daysLeft}</td>
              <td class="mono">{b.avgActualPct}%</td>
            </tr>
          {/each}
          {#if nearestDeadline.length === 0}
            <tr><td colspan="3" class="empty-note">Tidak ada deadline mendatang.</td></tr>
          {/if}
        </tbody>
      </table>
    </div>
    <div class="section">
      <div class="section-title">Berstatus lose — perlu perhatian</div>
      <table class="sheet-table">
        <thead><tr><th>Nama project</th><th>Actual</th><th>Expected</th><th>Sisa hari</th></tr></thead>
        <tbody>
          {#each loseBoards as b (b.boardId)}
            <tr>
              <td>{b.boardName}</td>
              <td class="mono">{b.avgActualPct}%</td>
              <td class="mono">{b.avgExpectedPct}%</td>
              <td class="mono days-late">{b.daysLeft}</td>
            </tr>
          {/each}
          {#if loseBoards.length === 0}
            <tr><td colspan="4" class="empty-note">Tidak ada project berstatus lose saat ini.</td></tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>

  <div class="section">
    <div class="section-title">Tim</div>
    <div class="team-grid">
      {#each team as person (person.id)}
        {@const prog = progressByUserId[person.id]}
        {@const showMetrics = isSuperUser || person.id === $auth.user?.id}
        <div class="team-card">
          <div class="team-card-top">
            <Avatar initials={person.initials} size={32} title={person.display_name} />
            <div>
              <div class="team-name">{person.display_name}</div>
              <div class="muted small">{person.roles.join(', ')}</div>
            </div>
          </div>
          {#if showMetrics && prog}
            <div class="team-progress">
              <div class="team-stat-row">
                <span class="team-stat-pill team-stat-belum">{prog.belum} belum</span>
                <span class="team-stat-pill team-stat-onprogress">{prog.on_progress} on progress</span>
                <span class="team-stat-pill team-stat-selesai">{prog.selesai} selesai</span>
              </div>
              <div class="team-completion-label">
                <span class="muted small">Completion rate</span>
                <span class="mono small">{prog.completion_rate}%</span>
              </div>
              <div class="team-completion-bar">
                <div class="team-completion-fill" style="width:{prog.completion_rate}%"></div>
              </div>
            </div>
          {:else if showMetrics}
            <div class="team-progress muted small">Tidak ada day entry</div>
          {/if}
        </div>
      {/each}
    </div>
  </div>
{/if}
