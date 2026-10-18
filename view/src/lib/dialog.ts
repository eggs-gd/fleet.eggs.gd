export function focusDialog(node: HTMLElement, onDismiss: () => void) {
  const focus = () => node.focus();
  focus();
  requestAnimationFrame(focus);
  const onKey = (event: KeyboardEvent) => {
    if (event.key !== 'Escape') return;
    event.preventDefault();
    onDismiss();
  };
  window.addEventListener('keydown', onKey);
  return {
    update(next: () => void) {
      onDismiss = next;
    },
    destroy() {
      window.removeEventListener('keydown', onKey);
    }
  };
}
