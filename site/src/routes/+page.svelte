<script lang="ts">
  import {
    DashboardSection,
    HeroSection,
    InstallSection,
    ManagerLoopSection,
    OrchestratorSection,
    WorkersSection
  } from '$lib/sections_v2';

  let showTop = $state(false);
</script>

<svelte:window onscroll={() => (showTop = window.scrollY > 480)} />

<svelte:head>
  <title>fleet.eggs.gd — one manager for coding-agent work</title>
  <meta
    name="description"
    content="Fleet sits between a manager and real coding agents: projects, tasks, workers, ownership, and execution state — locally."
  />
</svelte:head>

<div class="site-shell" id="top">
  <header class="site-header" aria-label="Primary navigation">
    <a class="brand" href="/" aria-label="fleet.eggs.gd home">
      <span class="brand-mark" aria-hidden="true">F</span>
      <span>
        <strong>fleet.eggs.gd</strong>
        <small>local agent work</small>
      </span>
    </a>

    <nav class="nav-links" aria-label="Site links">
      <a href="#product">Manager</a>
      <a href="#orchestrator">Execution</a>
      <a href="#workers">Workers</a>
      <a href="#dashboard">Dashboard</a>
      <a href="#install">Download</a>
    </nav>

    <div class="header-actions">
      <span class="early-badge">Early access</span>
      <a class="header-cta" href="#install">Download</a>
    </div>
  </header>

  <main>
    <HeroSection />
    <ManagerLoopSection />
    <OrchestratorSection />
    <WorkersSection />
    <DashboardSection />
    <InstallSection />
  </main>

  <footer class="site-footer">
    <span>fleet.eggs.gd</span>
  </footer>

  {#if showTop}
    <a class="to-top" href="#top" aria-label="Back to top">
      <svg viewBox="0 0 16 16" aria-hidden="true">
        <path d="M8 3.5 3 8.5M8 3.5l5 5M8 3.5V13" />
      </svg>
    </a>
  {/if}
</div>
