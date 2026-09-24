<script>
  import Icon from './Icon.svelte';
  import { agentColorClass } from './lib/taskDisplay.js';

  export let projects = [];
  export let onApplied = async () => {};
  export let binding = null;
  export let onOpenBinding = () => {};

  const agents = ['claude', 'codex', 'cursor', 'gemini'];

  $: boundThreadPreview =
    binding?.threadId && binding.threadId.length > 12 ? `${binding.threadId.slice(0, 12)}…` : binding?.threadId || '';

  let text = '';
  let submitting = false;
  let recording = false;
  let error = '';
  let result = null;
  let mediaRecorder = null;
  let chunks = [];
  let textField;

  $: sortedProjects = [...projects].sort((a, b) => (a.title || a.id).localeCompare(b.title || b.id));

  async function submitText() {
    if (!text.trim() || submitting) return;
    submitting = true;
    error = '';
    result = null;
    try {
      const response = await fetch('/api/manager/text', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: text.trim(), source: 'text' })
      });
      const payload = await response.json();
      result = payload;
      if (!payload.ok) {
        error = payload.failure?.message || payload.failure?.code || 'Manager request failed';
        return;
      }
      text = '';
      await onApplied(payload);
    } catch (err) {
      error = err.message;
    } finally {
      submitting = false;
    }
  }

  async function toggleRecording() {
    error = '';
    if (recording) {
      mediaRecorder?.stop();
      return;
    }
    if (!navigator.mediaDevices?.getUserMedia) {
      error = 'Audio capture is not available in this browser.';
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      chunks = [];
      mediaRecorder = new MediaRecorder(stream);
      mediaRecorder.ondataavailable = (event) => {
        if (event.data?.size) chunks.push(event.data);
      };
      mediaRecorder.onstop = async () => {
        recording = false;
        stream.getTracks().forEach((track) => track.stop());
        const blob = new Blob(chunks, { type: mediaRecorder.mimeType || 'audio/webm' });
        await submitAudio(blob);
      };
      mediaRecorder.start();
      recording = true;
    } catch (err) {
      error = err.message;
    }
  }

  async function submitAudio(blob) {
    submitting = true;
    error = '';
    result = null;
    try {
      const body = new FormData();
      body.append('audio', blob, 'manager-capture.webm');
      body.append('source', 'voice');
      const response = await fetch('/api/manager/audio', { method: 'POST', body });
      const payload = await response.json();
      result = payload;
      if (!payload.ok) {
        error = payload.failure?.message || payload.failure?.code || 'Manager audio request failed';
        return;
      }
      await onApplied(payload);
    } catch (err) {
      error = err.message;
    } finally {
      submitting = false;
    }
  }

  function onKeydown(event) {
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      submitText();
    }
  }

  function insertMention(value) {
    const mention = `@${value} `;
    text = text ? `${text.trimEnd()} ${mention}` : mention;
    textField?.focus();
  }
</script>

<section class="manager-bar" aria-label="Core Manager">
  <div class="manager-bar-input">
    <span class="manager-bar-avatar"><Icon name="comment" size={16} /></span>
    <button
      type="button"
      class="manager-bar-route"
      on:click={onOpenBinding}
      title={binding?.bound
        ? `Routes to ${binding.agent}${binding.threadId ? ` · ${binding.threadId}` : ''} — open Settings > Manager`
        : 'Fast-path command parsing — no manager bound. Open Settings > Manager to bind one.'}
    >
      <span class={`agent-dot ${binding?.bound ? agentColorClass(binding.agent) : 'agent-other'}`}></span>
      {#if binding?.bound}
        {binding.agent}{boundThreadPreview ? ` · ${boundThreadPreview}` : ''}
      {:else}
        fast-path
      {/if}
    </button>
    <textarea
      bind:this={textField}
      rows="1"
      bind:value={text}
      on:keydown={onKeydown}
      placeholder="What do you want to do?"
      disabled={submitting}
    ></textarea>
    <button type="button" class="icon-button" class:recording disabled={submitting} on:click={toggleRecording} title="Record voice">
      <Icon name="mic" size={17} />
    </button>
    <button type="button" class="send-button" disabled={submitting || !text.trim()} on:click={submitText} title="Send">
      <Icon name="send" size={16} />
    </button>
  </div>

  <div class="manager-bar-chips">
    <div class="manager-bar-chip-group">
      <span class="manager-chip-label">@ Project</span>
      {#each sortedProjects.slice(0, 6) as project (project.id)}
        <button type="button" on:click={() => insertMention(project.title || project.id)}>
          {project.title || project.id}
        </button>
      {/each}
    </div>
    <div class="manager-bar-chip-group">
      <span class="manager-chip-label">@ Agent</span>
      {#each agents as agent (agent)}
        <button type="button" on:click={() => insertMention(agent)}>{agent}</button>
      {/each}
    </div>
  </div>

  {#if error}
    <p class="manager-error">{error}</p>
  {/if}
  {#if result}
    <details class="manager-result">
      <summary>{result.ok ? 'Manager applied' : 'Manager response'}</summary>
      <pre>{JSON.stringify(result, null, 2)}</pre>
    </details>
  {/if}
</section>
