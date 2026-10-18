<script lang="ts">
  import { onDestroy } from 'svelte';

  export let axis = 'x';
  export let onDrag: (delta: number) => void = () => {};

  let dragging = false;
  let lastX = 0;
  let lastY = 0;

  function unbind() {
    window.removeEventListener('pointermove', move);
    window.removeEventListener('pointerup', stop, true);
    window.removeEventListener('pointercancel', stop, true);
  }

  function start(event: PointerEvent) {
    if (event.button !== 0 || dragging) return;
    event.preventDefault();
    dragging = true;
    lastX = event.clientX;
    lastY = event.clientY;
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', stop, true);
    window.addEventListener('pointercancel', stop, true);
  }

  function move(event: PointerEvent) {
    if (!dragging) return;
    if (event.buttons === 0) {
      stop();
      return;
    }
    const dx = event.clientX - lastX;
    const dy = event.clientY - lastY;
    lastX = event.clientX;
    lastY = event.clientY;
    if (dx === 0 && dy === 0) return;
    onDrag(axis === 'y' ? dy : dx);
  }

  function stop() {
    if (!dragging) return;
    dragging = false;
    unbind();
  }

  function nudge(event: KeyboardEvent) {
    const step = event.shiftKey ? 48 : 24;
    const horizontal = axis !== 'y';
    if (horizontal && event.key === 'ArrowRight') {
      event.preventDefault();
      onDrag(step);
    } else if (horizontal && event.key === 'ArrowLeft') {
      event.preventDefault();
      onDrag(-step);
    } else if (!horizontal && event.key === 'ArrowDown') {
      event.preventDefault();
      onDrag(step);
    } else if (!horizontal && event.key === 'ArrowUp') {
      event.preventDefault();
      onDrag(-step);
    }
  }

  onDestroy(unbind);
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex a11y_no_noninteractive_element_interactions -->
<div
  class="split-gutter split-gutter-{axis}"
  class:is-dragging={dragging}
  role="separator"
  aria-orientation={axis === 'y' ? 'horizontal' : 'vertical'}
  aria-label={axis === 'y' ? 'Resize panels' : 'Resize columns'}
  tabindex="0"
  on:pointerdown={start}
  on:keydown={nudge}
>
  <span class="visually-hidden">{axis === 'y' ? 'Resize panels' : 'Resize columns'}</span>
</div>

<style>
  .split-gutter {
    flex: 0 0 auto;
    background: transparent;
    touch-action: none;
    user-select: none;
  }

  .split-gutter-x {
    width: 8px;
    cursor: col-resize;
  }

  .split-gutter-y {
    height: 8px;
    cursor: row-resize;
  }

  .split-gutter:hover,
  .split-gutter.is-dragging {
    background: var(--accent-soft);
  }

  @media (max-width: 900px) {
    .split-gutter {
      display: none;
    }
  }
</style>
