<script lang="ts">
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/authStore';

  let email = '';
  let password = '';
  let error: string | null = null;
  let submitting = false;

  async function handleSubmit() {
    error = null;
    submitting = true;
    try {
      console.log('[login-page] Form submitted with email:', email);
      await auth.login(email, password);
      console.log('[login-page] Login successful, redirecting...');
      await goto('/');
    } catch (e) {
      const errMsg = (e as Error).message;
      console.error('[login-page] Login failed:', errMsg, e);
      error = errMsg;
    } finally {
      submitting = false;
    }
  }
</script>

<form class="login-window" on:submit|preventDefault={handleSubmit}>
  <div class="titlebar">
    <div class="titlebar-left">
      <span class="titlebar-icon">R</span>
      <span class="titlebar-title">R&amp;D Ops — Masuk</span>
    </div>
    <div class="titlebar-btns">
      <button class="titlebar-btn" type="button" tabindex="-1">–</button>
      <button class="titlebar-btn" type="button" tabindex="-1">✕</button>
    </div>
  </div>
  <div class="login-body">
    <p class="muted small" style="margin:0">Masuk untuk melanjutkan ke portal R&amp;D Ops.</p>

    <div class="login-field">
      <span class="muted small">Email</span>
      <input class="inline-input" type="email" bind:value={email} required autocomplete="username" />
    </div>

    <div class="login-field">
      <span class="muted small">Password</span>
      <input class="inline-input" type="password" bind:value={password} required autocomplete="current-password" />
    </div>

    {#if error}
      <p class="small" style="color:var(--win-red); margin:0">{error}</p>
    {/if}

    <button class="quick-btn quick-btn-done" type="submit" disabled={submitting} style="padding:6px 12px">
      {submitting ? 'Memproses...' : 'Masuk'}
    </button>
  </div>
</form>
