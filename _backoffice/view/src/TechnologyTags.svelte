<script>
  import { displayTechnologyGroups, displayTechnologyTags } from './lib/technologyDisplay.js';
  import TechnologyTag from './TechnologyTag.svelte';

  export let tags = [];
  /** Optional mixed-variant groups for a single overflow-aware row. */
  export let groups = null;
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
  <span class="tech-more" title={`+${shown.hidden} more`}>+{shown.hidden}{moreSuffix ? ` ${moreSuffix}` : ''}</span>
{/if}
