<script lang="ts">
  import {
    displayTechnologyGroups,
    displayTechnologyTags,
    type TechnologyGroup
  } from './lib/technologyDisplay';
  import TechnologyTag from './TechnologyTag.svelte';

  export let tags: string[] = [];
  /** Optional mixed-variant groups for a single overflow-aware row. */
  export let groups: TechnologyGroup[] | null = null;
  export let limit = 8;
  export let variant = 'primary';
  export let moreSuffix = '';

  $: shown = groups
    ? displayTechnologyGroups(groups, limit)
    : (() => {
        const base = displayTechnologyTags(tags, limit);
        return {
          visible: base.visible.map((tag) => ({ tag, variant })),
          hidden: base.hidden
        };
      })();
</script>

{#each shown.visible as item (`${item.variant}:${item.tag.name}`)}
  <TechnologyTag tag={item.tag} variant={item.variant} />
{/each}
{#if shown.hidden}
  <span class="tech-more" title={`+${shown.hidden} more`}
    >+{shown.hidden}{moreSuffix ? ` ${moreSuffix}` : ''}</span
  >
{/if}

<style>
  .tech-more {
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-full);
    font-size: var(--text-2xs);
    font-weight: 700;
    line-height: 1.2;
    max-width: 100%;
    overflow: hidden;
    padding: var(--space-2xs) var(--space-sm);
    text-overflow: ellipsis;
    white-space: nowrap;
    background: var(--surface-muted);
    color: var(--text-muted);
  }
</style>
