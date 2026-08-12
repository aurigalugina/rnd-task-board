<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import '../app.css';
  import { auth } from '$lib/stores/authStore';
  import { theme } from '$lib/stores/themeStore';
  import { reviewQueue, loadReviewQueue } from '$lib/stores/reviewQueueStore';
  import { alerts, startAlertPolling, stopAlertPolling, loadAlerts } from '$lib/stores/notificationStore';
  import { chatSession } from '$lib/stores/chatSessionStore';
  import ProfileModal from '$lib/components/ProfileModal.svelte';
  import SettingsModal from '$lib/components/SettingsModal.svelte';

  onMount(() => {
    auth.init();
  });

  // Review Queue (FR-NTF-01) sekarang di-filter PER USER di backend (reviewer
  // yang di-assign ke Big Task, atau fallback spv kalau Big Task-nya belum
  // di-assign reviewer sama sekali) -- bukan hardcode role spv lagi, lihat
  // decision-log-bigtask-reviewer-assignment-20260810.md. Jadi fetch buat
  // SEMUA user yang sudah login, bukan cuma yang ber-role spv. Store dipakai
  // bareng routes/review-queue/+page.svelte supaya mark-reviewed dari halaman
  // itu langsung kepantul ke badge di sini juga (bukan cuma di halamannya).
  $: pendingReview = $reviewQueue.length;
  $: if ($auth.status === 'authenticated') {
    loadReviewQueue();
    startAlertPolling();
  }
  $: if ($auth.status === 'unauthenticated') stopAlertPolling();

  $: totalBadge = pendingReview + $alerts.length;

  // Board Archive (super_user only) -- lihat decision-log-board-archive-20260812.md.
  $: isSuperUser = $auth.user?.access_level === 'super_user';

  $: isLoginPage = $page.url.pathname === '/login';
  $: if ($auth.status === 'unauthenticated' && !isLoginPage) {
    goto('/login');
  }

  // Tutup dropdown notif/user-menu tiap kali route berubah — tanpa ini, klik
  // link nav biasa (bukan tombol "Buka Review Queue" yang punya handler close
  // sendiri) ninggalin dropdown kebuka nutupin halaman tujuan.
  $: if ($page.url.pathname) {
    showNotif = false;
    showUserMenu = false;
  }

  // Sinkronkan tema dari profil user ke <html data-theme> — CSS custom
  // properties di app.css didefinisikan lewat selector :root[data-theme=...].
  $: if (browser) {
    document.documentElement.setAttribute('data-theme', $auth.user?.theme_preference || $theme);
  }
  $: if ($auth.user?.theme_preference) {
    theme.set($auth.user.theme_preference as any);
  }

  function handleLogout() {
    auth.logout();
  }

  let showNotif = false;
  let showUserMenu = false;
  let showProfile = false;
  let showSettings = false;

  function toggleNotif() {
    showNotif = !showNotif;
    showUserMenu = false;
    if (showNotif) { loadReviewQueue(); loadAlerts(); }
  }

  const alertLabel: Record<string, string> = {
    sign_off_ready: '✅ Sign-off siap',
    verdict_lose:   '⛔ Verdict Lose',
    deadline_soon:  '⏰ Deadline dekat',
  };
  function toggleUserMenu() {
    showUserMenu = !showUserMenu;
    showNotif = false;
  }
</script>

{#if $auth.status === 'idle' || $auth.status === 'loading'}
  <p class="boot-hint">Memuat sesi...</p>
{:else if $auth.status === 'unauthenticated' && !isLoginPage}
  <!-- Menunggu goto('/login') selesai — sengaja tidak merender <slot/> di sini
       supaya halaman terautentikasi tidak sempat mount sebelum redirect kelar. -->
  <p class="boot-hint">Mengalihkan ke halaman login...</p>
{:else}
  <!-- <slot/> selalu di posisi DOM yang sama (dibungkus div yang sama) baik untuk
       halaman login maupun terautentikasi — cuma class & sibling (titlebar/topbar)
       yang toggle. Lihat gotcha di CLAUDE.md soal double-mount kalau slot pindah
       cabang struktural. -->
  <div class="app" class:login-mode={isLoginPage}>
    {#if !isLoginPage}
      <div class="titlebar">
        <div class="titlebar-left">
          <span class="titlebar-icon">R</span>
          <span class="titlebar-title">R&amp;D Ops — PT USSI Pinbuk Prima Software</span>
        </div>
        <div class="titlebar-btns">
          <button class="titlebar-btn" tabindex="-1">–</button>
          <button class="titlebar-btn" tabindex="-1">▢</button>
          <button class="titlebar-btn" tabindex="-1">✕</button>
        </div>
      </div>

      <div class="topbar">
        <nav class="tabs">
          <a class="tab-btn {$page.url.pathname === '/' ? 'tab-btn-active' : ''}" href="/">Dashboard</a>
          <a class="tab-btn {$page.url.pathname.startsWith('/boards') ? 'tab-btn-active' : ''}" href="/boards">Boards</a>
          <a class="tab-btn {$page.url.pathname === '/weekly-plan' ? 'tab-btn-active' : ''}" href="/weekly-plan">My Weekly Plan</a>
          <a class="tab-btn {$page.url.pathname === '/review-queue' ? 'tab-btn-active' : ''}" href="/review-queue">
            Review queue
            {#if pendingReview > 0}<span class="review-badge">{pendingReview}</span>{/if}
          </a>
          <a
            class="tab-btn {$page.url.pathname === '/change-requests' ? 'tab-btn-active' : ''}"
            href="/change-requests">
            Change request
            {#if $chatSession.step === 'chatting'}<span class="chat-live-dot" title="Sesi chat masih aktif"></span>{/if}
          </a>
        </nav>
        <div class="topbar-right">
          <div class="notif-wrap">
            <button class="icon-only-btn" on:click={toggleNotif} aria-label="Notifikasi">
              🔔
              {#if totalBadge > 0}<span class="notif-badge">{totalBadge}</span>{/if}
            </button>
            {#if showNotif}
              <div class="dropdown-panel notif-dropdown">
                {#if $alerts.length > 0}
                  <div class="dropdown-title">Alert</div>
                  {#each $alerts as a (a.big_task_id + a.type)}
                    <div class="notif-item notif-item-alert">
                      <span class="small"><b>{alertLabel[a.type]}</b></span>
                      <span class="small muted">{a.big_task_name}</span>
                      <span class="small muted">{a.board_name}</span>
                      {#if a.type === 'deadline_soon'}
                        <span class="small" style="color:var(--win-amber)">{a.days_left === 0 ? 'Hari ini!' : a.days_left === 1 ? 'Besok' : `${a.days_left} hari lagi`} — {a.actual_pct}% vs {a.expected_pct}%</span>
                      {:else if a.type === 'verdict_lose'}
                        <span class="small" style="color:var(--win-red)">Deadline {-a.days_left} hari lalu — {a.actual_pct}%</span>
                      {:else}
                        <span class="small" style="color:var(--win-green)">{a.actual_pct}% — siap di-sign</span>
                      {/if}
                    </div>
                  {/each}
                  <div class="dropdown-divider" />
                {/if}
                <div class="dropdown-title">Perlu ditinjau</div>
                {#each $reviewQueue.slice(0, 5) as item (item.id)}
                  <div class="notif-item">
                    <span class="review-dot" />
                    <span class="small">{item.title}</span>
                  </div>
                {/each}
                {#if pendingReview === 0}
                  <div class="empty-note">Tidak ada yang perlu ditinjau.</div>
                {/if}
                {#if $alerts.length === 0 && pendingReview === 0}
                  <div class="empty-note">Semua aman 👍</div>
                {/if}
                <a class="quick-btn" style="width:100%;margin-top:6px;box-sizing:border-box;text-decoration:none;text-align:center" href="/review-queue" on:click={() => (showNotif = false)}>
                  Buka Review Queue
                </a>
              </div>
            {/if}
          </div>
          <div class="user-menu-wrap">
            <button class="user-menu-trigger" on:click={toggleUserMenu}>
              <div class="avatar" style="width:26px;height:26px;font-size:10px">{$auth.user?.initials ?? ''}</div>
            </button>
            {#if showUserMenu}
              <div class="dropdown-panel">
                <div class="user-menu-head">
                  <div class="avatar" style="width:30px;height:30px;font-size:12px">{$auth.user?.initials ?? ''}</div>
                  <div>
                    <div class="panel-row-name">{$auth.user?.display_name ?? ''}</div>
                    <div class="muted small">{$auth.user?.roles.join(', ') ?? ''}</div>
                  </div>
                </div>
                <div class="dropdown-divider" />
                <button class="dropdown-item" on:click={() => { showProfile = true; showUserMenu = false; }}>
                  👤 My Profile
                </button>
                {#if isSuperUser}
                  <a class="dropdown-item" href="/boards/archive" on:click={() => (showUserMenu = false)}>
                    🗄 Board Archive
                  </a>
                {/if}
                <button class="dropdown-item" on:click={() => { showSettings = true; showUserMenu = false; }}>
                  ⚙️ Settings
                </button>
                <div class="dropdown-divider" />
                <button class="dropdown-item dropdown-item-danger" on:click={handleLogout}>
                  ↪ Sign out
                </button>
              </div>
            {/if}
          </div>
        </div>
      </div>
    {/if}

    <div class={isLoginPage ? 'login-content' : 'content'}>
      <slot />
    </div>
  </div>

  {#if !isLoginPage && showProfile}
    <ProfileModal on:close={() => (showProfile = false)} />
  {/if}
  {#if !isLoginPage && showSettings}
    <SettingsModal on:close={() => (showSettings = false)} />
  {/if}
{/if}

<style>
  .boot-hint { padding: 24px; color: #616161; font-size: 13px; }
</style>
