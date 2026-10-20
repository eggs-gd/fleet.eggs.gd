<script lang="ts">
  import { onMount } from 'svelte';
  import { choose, consent, initConsent } from './consent';

  let ready = $state(false);

  onMount(() => {
    initConsent();
    ready = true;
  });
</script>

{#if ready && $consent === 'unset'}
  <aside class="consent" aria-label="Analytics cookies">
    <p>
      This site uses Google Analytics to count visits. Cookies are set only if you accept; without
      them it sends only cookieless pings.
    </p>
    <div class="consent-actions">
      <button type="button" onclick={() => choose('denied')}>Decline</button>
      <button type="button" onclick={() => choose('granted')}>Accept</button>
    </div>
  </aside>
{/if}

<style>
  .consent {
    position: fixed;
    z-index: 60;
    right: var(--space-s);
    bottom: var(--space-s);
    left: var(--space-s);
    max-width: 560px;
    margin-left: auto;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-s);
    padding: var(--space-s);
    border: 1px solid var(--line-strong);
    border-radius: 8px;
    background: var(--panel-strong);
    color: var(--text);
    font-size: var(--type-text-s);
  }

  p {
    flex: 1 1 260px;
    margin: 0;
    color: var(--muted);
  }

  .consent-actions {
    display: flex;
    gap: 8px;
  }

  button {
    min-height: 36px;
    padding: 0 16px;
    border: 1px solid var(--line-strong);
    border-radius: 6px;
    background: var(--panel);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }

  button:hover {
    border-color: var(--text);
  }

  button:focus-visible {
    outline: 2px solid var(--text);
    outline-offset: 2px;
  }
</style>
