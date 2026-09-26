<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { Size } from './size';

  let {
    id,
    level = 2,
    size = 'm',
    class: className = '',
    children
  }: {
    id?: string;
    level?: 1 | 2;
    size?: Size;
    class?: string;
    children: Snippet;
  } = $props();
</script>

{#if level === 1}
  <h1 {id} class="section-title size-{size} {className}">{@render children()}</h1>
{:else}
  <h2 {id} class="section-title size-{size} {className}">{@render children()}</h2>
{/if}
