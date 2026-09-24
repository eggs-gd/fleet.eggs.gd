<script>
  import { onDestroy } from 'svelte';

  export let axis = 'x';
  export let onDrag = () => {};

  let dragging = false;
  let lastX = 0;
  let lastY = 0;

  function unbind() {
    window.removeEventListener('pointermove', move);
    window.removeEventListener('pointerup', stop, true);
    window.removeEventListener('pointercancel', stop, true);
  }

  function start(event) {
    if (event.button !== 0 || dragging) return;
    event.preventDefault();
    dragging = true;
    lastX = event.clientX;
    lastY = event.clientY;
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', stop, true);
    window.addEventListener('pointercancel', stop, true);
  }

  function move(event) {
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

  onDestroy(unbind);
</script>

<div
  class="split-gutter split-gutter-{axis}"
  class:is-dragging={dragging}
  role="separator"
  aria-orientation={axis === 'y' ? 'horizontal' : 'vertical'}
  on:pointerdown={start}
>
  <span class="visually-hidden">{axis === 'y' ? 'Resize panels' : 'Resize columns'}</span>
</div>
