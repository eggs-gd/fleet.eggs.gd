<script>
  export let tag;
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
    <svg class="tech-glyph-svg" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true" focusable="false">
      {@html tag.svg}
    </svg>
  {:else if tag.glyph}
    <span class="tech-glyph" aria-hidden="true">{tag.glyph}</span>
  {:else}
    <span class="tech-fallback">{tag.label}</span>
  {/if}
</span>
