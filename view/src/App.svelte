<script lang="ts">
  import { onMount } from 'svelte';
  import Dialogs from './components/shell/Dialogs.svelte';
  import ManagerSetup from './components/shell/ManagerSetup.svelte';
  import Sidebar from './components/shell/Sidebar.svelte';
  import SplitGutter from './components/shell/SplitGutter.svelte';
  import { isSettingsNav } from './lib/settingsNav';
  import { applyTheme, watchSystemTheme } from './lib/theme';
  import { loadThemePref } from './lib/layoutPrefs';
  import SettingsRoute from './routes/SettingsRoute.svelte';
  import WorkRoute from './routes/WorkRoute.svelte';
  import { data, error, loadData, startPolling } from './stores/board';
  import { dragLeft, leftWidth, rightWidth, themePref } from './stores/layout';
  import { globalNav, startLocationSync } from './stores/location';
  import {
    loadManagerSetup,
    managerProviders,
    managerSetupNeeded,
    skipManagerSetup
  } from './stores/managerSetup';

  onMount(() => {
    const stopLocationSync = startLocationSync(window.location.search);
    applyTheme($themePref);
    loadData();
    loadManagerSetup();
    const stopPolling = startPolling();
    const stopWatchingTheme = watchSystemTheme(() => {
      if (loadThemePref() === 'system') applyTheme('system');
    });
    return () => {
      stopLocationSync();
      stopPolling();
      stopWatchingTheme();
    };
  });
</script>

<svelte:head>
  <title>Fleet</title>
</svelte:head>

{#if $managerSetupNeeded}
  <ManagerSetup
    providers={$managerProviders}
    onDone={() => managerSetupNeeded.set(false)}
    onSkip={skipManagerSetup}
  />
{/if}

<a class="skip-link" href="#main">Skip to content</a>
<main id="main" class="shell">
  {#if $error}
    <section class="error">
      <strong>Backoffice data is not available.</strong>
      <span>{$error}</span>
    </section>
  {:else if !$data}
    <section class="loading">Loading Fleet state…</section>
  {:else}
    <div class="app-shell" style="--left-w: {$leftWidth}px; --right-w: {$rightWidth}px">
      <Sidebar />
      <SplitGutter axis="x" onDrag={dragLeft} />

      {#if isSettingsNav($globalNav)}
        <SettingsRoute />
      {:else}
        <WorkRoute />
      {/if}
    </div>
  {/if}
</main>

<Dialogs />

<style>
  .skip-link {
    position: absolute;
    left: var(--space-md);
    top: var(--space-md);
    z-index: 40;
    transform: translateY(-200%);
    background: var(--surface);
    color: var(--text);
    padding: var(--space-sm) var(--space-lg);
  }

  .skip-link:focus {
    transform: none;
  }

  .shell {
    width: 100%;
    height: 100dvh;
    padding: calc(var(--space-lg) + env(safe-area-inset-top))
      calc(var(--space-xl) + env(safe-area-inset-right))
      calc(var(--space-lg) + env(safe-area-inset-bottom))
      calc(var(--space-xl) + env(safe-area-inset-left));
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .app-shell {
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
    flex-direction: row;
    min-width: 0;
    --left-w: 240px;
    --right-w: 300px;
  }

  .loading,
  .error {
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    background: var(--surface);
    padding: var(--space-xl);
  }

  .error {
    border-color: var(--error-border);
    background: var(--red-soft);
    color: var(--red-text);
  }

  .error span {
    display: block;
    margin-top: var(--space-sm);
  }

  @media (max-width: 1080px) {
    .app-shell {
      --left-w: 220px;
      --right-w: 260px;
    }
  }

  @media (max-width: 900px) {
    :global(html),
    :global(body) {
      overflow: auto;
    }

    .shell {
      height: auto;
      padding: var(--space-lg);
    }

    .app-shell {
      display: flex;
      flex-direction: column;
      height: auto;
    }
  }
</style>
