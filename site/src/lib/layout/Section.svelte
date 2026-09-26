<script lang="ts">
  import type { Snippet } from 'svelte';
  import './layout.css';

  let {
    id,
    class: className = '',
    bleed = false,
    'aria-labelledby': ariaLabelledby,
    tag,
    band,
    children,
    footer
  }: {
    id?: string;
    class?: string;
    bleed?: boolean;
    'aria-labelledby'?: string;
    tag?: Snippet;
    band?: Snippet;
    children: Snippet;
    footer?: Snippet;
  } = $props();
</script>

<section
  class="site-section {bleed ? 'bleed' : ''} {className}"
  {id}
  aria-labelledby={ariaLabelledby}
>
  {#if tag}
    {@render tag()}
  {/if}
  {#if band}
    {@render band()}
  {/if}
  {@render children()}
  {#if footer}
    {@render footer()}
  {/if}
</section>
