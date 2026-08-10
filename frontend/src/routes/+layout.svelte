<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import '../app.css';
  import { auth } from '$lib/stores/authStore';
  import { theme } from '$lib/stores/themeStore';
  import { reviewQueue, loadReviewQueue } from '$lib/stores/reviewQueueStore';
  import ProfileModal from '$lib/components/ProfileModal.svelte';
  import SettingsModal from '$lib/components/SettingsModal.svelte';

  onMount(() => {
    auth.init();
  });

  // Review Queue (FR-NTF-01) dibatasi role spv di backend — cuma fetch kalau
  // user sekarang punya role itu, biar tidak nembak 403 percuma. Store dipakai
  // bareng routes/review-queue/+page.svelte supaya mark-reviewed dari halaman
  // itu langsung kepantul ke badge di sini juga (bukan cuma di halamannya).
  $: isSpv = $auth.user?.roles.includes('spv') ?? false;
  $: pendingReview = $reviewQueue.length;
  $: if (isSpv) loadReviewQueue();

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
    if (showNotif && isSpv) loadReviewQueue();
  }
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
            {#if isSpv && pendingReview > 0}<span class="review-badge">{pendingReview}</span>{/if}
          </a>
        </nav>
        <div class="topbar-right">
          <div class="notif-wrap">
            <button class="icon-only-btn" on:click={toggleNotif} aria-label="Notifikasi">
              🔔
              {#if isSpv && pendingReview > 0}<span class="notif-badge">{pendingReview}</span>{/if}
            </button>
            {#if showNotif}
              <div class="dropdown-panel">
                <div class="dropdown-title">Butuh atensi lo</div>
                {#if isSpv}
                  {#each $reviewQueue.slice(0, 5) as item (item.id)}
                    <div class="notif-item">
                      <span class="review-dot" />
                      <span class="small">{item.title}</span>
                    </div>
                  {/each}
                  {#if pendingReview === 0}
                    <div class="empty-note">Ga ada yang perlu direview. Rapi.</div>
                  {/if}
                  <a class="quick-btn" style="width:100%; margin-top:6px; box-sizing:border-box; text-decoration:none; text-align:center" href="/review-queue" on:click={() => (showNotif = false)}>
                    Buka Review Queue
                  </a>
                {:else}
                  <div class="empty-note">Review Queue cuma buat role SPV.</div>
                {/if}
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
