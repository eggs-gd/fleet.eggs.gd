<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { Size } from './size';

  let {
    size = 'm',
    class: className = '',
    children
  }: {
    size?: Size;
    class?: string;
    children: Snippet;
  } = $props();
</script>

<p class="section-subtitle size-{size} {className}">{@render children()}</p>
