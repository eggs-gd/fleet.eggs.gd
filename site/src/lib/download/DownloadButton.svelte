<script lang="ts">
  import { onMount } from 'svelte';

  const REPO = 'eggs-gd/fleet.eggs.gd';
  const LATEST_PAGE = `https://github.com/${REPO}/releases/latest`;

  type Os = 'macos' | 'windows' | 'linux';
  type Arch = 'arm64' | 'amd64';
  type Platform = { os: Os; arch: Arch };
  type Build = Platform & { url: string };

  const OS_LABEL: Record<Os, string> = { macos: 'macOS', windows: 'Windows', linux: 'Linux' };

  function archLabel({ os, arch }: Platform): string {
    if (os === 'macos') return arch === 'arm64' ? 'Apple silicon' : 'Intel';
    return arch === 'arm64' ? 'ARM64' : 'x64';
  }

  function parseAsset(name: string, url: string): Build | null {
    const n = name.toLowerCase();
    if (/(sha256|checksum|\.sig$|\.sbom|\.txt$)/.test(n)) return null;
    const os: Os | null = /(macos|darwin|osx|\.dmg$)/.test(n)
      ? 'macos'
      : /(windows|win32|win64|\.msi$|\.exe$)/.test(n)
        ? 'windows'
        : /linux/.test(n)
          ? 'linux'
          : null;
    if (!os) return null;
    const arch: Arch = /(arm64|aarch64)/.test(n) ? 'arm64' : 'amd64';
    return { os, arch, url };
  }

  function detect(): Platform | null {
    const nav = navigator as Navigator & { userAgentData?: { platform?: string } };
    const hint = nav.userAgentData?.platform ?? '';
    const ua = navigator.userAgent;
    if (/(iPhone|iPad|Android)/i.test(ua)) return null;
    const source = `${hint} ${ua}`;
    if (/Mac/i.test(source)) return { os: 'macos', arch: 'arm64' };
    if (/Win/i.test(source)) return { os: 'windows', arch: /ARM/i.test(source) ? 'arm64' : 'amd64' };
    if (/(Linux|X11|CrOS)/i.test(source)) return { os: 'linux', arch: /(aarch64|arm)/i.test(source) ? 'arm64' : 'amd64' };
    return null;
  }

  let status = $state<'loading' | 'none' | 'ready'>('loading');
  let builds = $state<Build[]>([]);
  let detected = $state<Platform | null>(null);
  let open = $state(false);
  let root: HTMLElement;

  onMount(() => {
    detected = detect();
    const nav = navigator as Navigator & {
      userAgentData?: { getHighEntropyValues?: (h: string[]) => Promise<{ architecture?: string }> };
    };
    nav.userAgentData?.getHighEntropyValues?.(['architecture']).then((v) => {
      if (detected?.os === 'macos' && v.architecture === 'x86') detected = { os: 'macos', arch: 'amd64' };
    });

    fetch(`https://api.github.com/repos/${REPO}/releases/latest`, {
      headers: { Accept: 'application/vnd.github+json' }
    })
      .then((res) => (res.ok ? res.json() : null))
      .then((rel) => {
        const found = (rel?.assets ?? [])
          .map((a: { name: string; browser_download_url: string }) =>
            parseAsset(a.name, a.browser_download_url)
          )
          .filter(Boolean) as Build[];
        builds = found;
        status = found.length ? 'ready' : 'none';
      })
      .catch(() => {
        status = 'none';
      });

    const close = (e: Event) => {
      if (!root.contains(e.target as Node)) open = false;
    };
    const esc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') open = false;
    };
    document.addEventListener('click', close);
    document.addEventListener('keydown', esc);
    return () => {
      document.removeEventListener('click', close);
      document.removeEventListener('keydown', esc);
    };
  });

  const match = $derived(
    detected ? builds.find((b) => b.os === detected!.os && b.arch === detected!.arch) : undefined
  );
  const disabled = $derived(status === 'none');
  const href = $derived(match?.url ?? LATEST_PAGE);
  const label = $derived.by(() => {
    if (status === 'none') return 'Coming soon';
    if (match) return `Download for ${OS_LABEL[match.os]} · ${archLabel(match)}`;
    if (detected && status === 'ready') return `Download for ${OS_LABEL[detected.os]}`;
    return 'Download';
  });
</script>

<div class="dl" bind:this={root}>
  <div class="dl-split" class:disabled>
    <a
      class="primary-cta dl-main"
      {href}
      aria-disabled={disabled}
      tabindex={disabled ? -1 : undefined}
      onclick={(e) => disabled && e.preventDefault()}>{label}</a
    >
    <button
      type="button"
      class="primary-cta dl-caret"
      aria-label="Other platforms"
      aria-haspopup="true"
      aria-expanded={open}
      {disabled}
      onclick={() => (open = !open)}>▾</button
    >
  </div>

  {#if open && builds.length}
    <ul class="dl-menu">
      {#each builds as b (b.url)}
        <li>
          <a href={b.url}>{OS_LABEL[b.os]} <span>{archLabel(b)}</span></a>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .dl {
    position: relative;
    display: inline-block;
    text-align: left;
  }

  .dl-split {
    display: inline-flex;
  }

  .dl-main {
    border-radius: 4px 0 0 4px;
  }

  .dl-caret {
    min-width: 38px;
    padding: 0;
    border-left-color: #b7c0cd;
    border-radius: 0 4px 4px 0;
    cursor: pointer;
    font-family: inherit;
  }

  .dl-split.disabled {
    opacity: 0.5;
  }

  .dl-split.disabled .dl-main {
    cursor: not-allowed;
  }

  .dl-split.disabled .dl-caret {
    cursor: not-allowed;
  }

  .dl-menu {
    position: absolute;
    z-index: 5;
    top: calc(100% + 6px);
    left: 0;
    min-width: 100%;
    margin: 0;
    padding: 4px;
    list-style: none;
    border: 1px solid var(--line-strong);
    border-radius: 4px;
    background: var(--panel-strong);
  }

  .dl-menu a {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    border-radius: 3px;
    padding: 8px 10px;
    color: var(--text);
    font-size: 13px;
    white-space: nowrap;
  }

  .dl-menu a:hover {
    background: rgba(218, 226, 237, 0.08);
  }

  .dl-menu span {
    color: var(--faint);
  }
</style>
