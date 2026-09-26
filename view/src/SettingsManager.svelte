<script>
  import SettingsField from './SettingsField.svelte';
  import { timeAgo } from './lib/taskDisplay.js';

  export let manager = {};
  export let onReload = async () => {};

  let threads = [];
  let threadsLoading = false;
  let threadsNote = '';
  let adoptError = '';
  let adopting = '';

  $: session = manager?.session || null;
  $: agent = session ? manager.provider : '';
  $: workspace = session?.workspace || '';

  async function loadThreads(nextAgent, nextWorkspace) {
    threads = [];
    threadsNote = '';
    if (!nextAgent || !nextWorkspace) return;
    threadsLoading = true;
    try {
      const response = await fetch(
        `/api/manager/threads?agent=${encodeURIComponent(nextAgent)}&cwd=${encodeURIComponent(nextWorkspace)}`
      );
      const body = await response.text();
      if (!response.ok) {
        if (body.includes('not implemented')) {
          threadsNote = 'This provider does not list session roots.';
          return;
        }
        throw new Error(body || `Failed to list sessions (${response.status})`);
      }
      const listed = body ? JSON.parse(body) : [];
      threads = listed.filter((thread) => thread.cwd);
      if (!threads.length) threadsNote = 'No other sessions use this data root.';
    } catch (err) {
      threadsNote = err.message || String(err);
    } finally {
      threadsLoading = false;
    }
  }

  $: loadThreads(agent, workspace);

  async function adopt(threadId) {
    adopting = threadId;
    adoptError = '';
    try {
      const response = await fetch('/api/manager/adopt', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent, threadId })
      });
      if (!response.ok) throw new Error(await response.text());
      await onReload();
    } catch (err) {
      adoptError = err.message || String(err);
    } finally {
      adopting = '';
    }
  }
</script>

<section class="settings-block" aria-label="Manager">
  <h3>Manager</h3>
  <p class="settings-lede">Manager is a session in the provider app. Fleet remembers it and does not send chat into it.</p>
  <SettingsField label="Role" value={manager.role || '—'} />
  <SettingsField label="Provider" value={session ? manager.provider : 'none'} />
  <SettingsField label="Session" value={session?.id || 'none'} mono />
  <SettingsField label="Workspace" value={session?.workspace || '—'} mono />

  {#if session}
    <p class="settings-lede">Sessions whose root is this workspace</p>
    {#if threadsLoading}
      <p class="settings-hint">Loading sessions…</p>
    {:else if threads.length}
      <ul class="settings-notes">
        {#each threads as thread (thread.id)}
          <li>
            {thread.name || '(untitled)'} — {timeAgo(thread.updatedAt) || thread.id}
            {#if thread.id !== session.id}
              <button type="button" class="settings-btn" disabled={adopting !== ''} on:click={() => adopt(thread.id)}>
                {adopting === thread.id ? 'Saving…' : 'Use as Manager'}
              </button>
            {/if}
          </li>
        {/each}
      </ul>
    {:else if threadsNote}
      <p class="settings-hint">{threadsNote}</p>
    {/if}
    {#if adoptError}
      <p class="settings-error">{adoptError}</p>
    {/if}
  {/if}
</section>

{#if manager.notes?.length}
  <ul class="settings-notes">
    {#each manager.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
