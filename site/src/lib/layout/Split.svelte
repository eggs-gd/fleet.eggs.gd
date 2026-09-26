<script lang="ts">
  import type { Snippet } from 'svelte';
  import { spaceVar, type Size } from './size';

  let {
    class: className = '',
    flat = false,
    columns = 'minmax(0, 1fr) minmax(0, 1fr)',
    gap = '32px',
    start,
    end
  }: {
    class?: string;
    flat?: boolean;
    columns?: string;
    gap?: Size | string;
    start: Snippet;
    end: Snippet;
  } = $props();
</script>

{#if flat}
  {@render start()}
  {@render end()}
{:else}
  <div class="split {className}" style="--split-columns: {columns}; --split-gap: {spaceVar(gap)}">
    <div class="split-start">
      {@render start()}
    </div>
    <div class="split-end">
      {@render end()}
    </div>
  </div>
{/if}
