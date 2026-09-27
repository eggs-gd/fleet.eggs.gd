<script lang="ts">
  import type { TechnologyTagDisplay } from './lib/technologyDisplay';

  export let tag: TechnologyTagDisplay;
  export let variant = 'primary';
</script>

<span
  class="tech-tag {variant}"
  class:unknown={!tag.known}
  class:icon={Boolean(tag.svg || tag.glyph)}
  title={tag.title}
  aria-label={tag.title}
>
  {#if tag.svg}
    <svg
      class="tech-glyph-svg"
      viewBox="0 0 24 24"
      width="14"
      height="14"
      aria-hidden="true"
      focusable="false"
    >
      {@html tag.svg}
    </svg>
  {:else if tag.glyph}
    <span class="tech-glyph" aria-hidden="true">{tag.glyph}</span>
  {:else}
    <span class="tech-fallback">{tag.label}</span>
  {/if}
</span>

<style>
  .tech-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    width: 24px;
    height: 24px;
    min-width: 24px;
    min-height: 24px;
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-full);
    font-size: var(--text-2xs);
    font-weight: 700;
    line-height: 1.2;
    max-width: 100%;
    overflow: hidden;
    padding: 0;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tech-tag.icon {
    font-weight: 400;
  }

  .tech-glyph {
    display: block;
    font-size: var(--text-md);
    line-height: 1;
    transform: translateY(0.5px);
  }

  .tech-glyph-svg {
    display: block;
    width: 14px;
    height: 14px;
    flex: 0 0 14px;
  }

  .tech-fallback {
    display: block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tech-tag.unknown {
    width: auto;
    max-width: 72px;
    min-width: 24px;
    max-height: 24px;
    padding: 0 var(--space-sm);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: var(--text-2xs);
    text-transform: uppercase;
  }

  .tech-tag:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  .tech-tag.primary {
    border-color: var(--ok-border);
    background: var(--green-soft);
    color: var(--green-text);
  }

  .tech-tag.tooling {
    border-color: var(--warn-border);
    background: var(--amber-soft);
    color: var(--amber-text);
  }
</style>
