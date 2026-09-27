<script>
  import SettingsField from './SettingsField.svelte';
  import { fieldHint } from './lib/settingsDraft.js';

  export let agents = {};
  export let draft = { agents: {} };
  export let onUpdate = () => {};
  export let onRecheck = () => {};
  export let dirty = false;

  let openId = '';

  function toggle(id) {
    openId = openId === id ? '' : id;
  }

  const statusTone = (agent) => {
    if (agent.enabled?.value === false) return 'muted';
    if (agent.status === 'ready') return 'ok';
    if (agent.status === 'non_canonical') return 'warn';
    if (agent.status === 'not_found' || agent.status === 'error') return 'fail';
    return 'muted';
  };

  const statusLabel = (agent) => {
    if (agent.enabled?.value === false) {
      const n = agent.active_sessions || 0;
      return n ? `Disabled · ${n} active session(s)` : 'Disabled';
    }
    return agent.status;
  };
</script>

<section class="settings-block" aria-label="Workers">
  <h3>Workers</h3>
  <p class="settings-lede">
    Registered executor providers only. Save writes overlay enabled/executable/routing. Recheck
    probes discovery and does not write core.local.yaml.
  </p>
  <div class="settings-row">
    <span class="settings-key">Discovery</span>
    <span class="settings-val">
      <button type="button" class="settings-btn" disabled={dirty} on:click={onRecheck}
        >Recheck</button
      >
    </span>
  </div>
  {#if dirty}
    <p class="settings-hint">Save or discard before Recheck.</p>
  {/if}
  {#each agents.providers || [] as agent (agent.id)}
    <article class="settings-item settings-item--click" class:is-open={openId === agent.id}>
      <button
        type="button"
        class="settings-item-head settings-item-head--btn"
        on:click={() => toggle(agent.id)}
      >
        <span class={`status-dot-inline is-${statusTone(agent)}`}></span>
        <strong>{agent.name}</strong>
        <span class="settings-pill">{statusLabel(agent)}</span>
        <span class="settings-item-meta"
          >{agent.effective_executable?.value || agent.expected || ''}</span
        >
      </button>
      {#if openId === agent.id}
        <div class="settings-item-body">
          <div class="settings-row">
            <span class="settings-key">Enabled</span>
            <span class="settings-val">
              <label class="settings-check">
                <input
                  type="checkbox"
                  checked={draft.agents?.[agent.id]?.enabled !== false}
                  on:change={(e) => onUpdate(agent.id, { enabled: e.target.checked })}
                />
                Launch new sessions
              </label>
            </span>
          </div>
          <p class="settings-hint">
            {fieldHint(agent.enabled)} · disable does not stop live sessions
          </p>
          <div class="settings-row">
            <span class="settings-key">Configured executable</span>
            <span class="settings-val">
              <input
                class="settings-input is-mono"
                type="text"
                value={draft.agents?.[agent.id]?.executable || ''}
                placeholder="absolute path, empty = PATH/bundled"
                on:input={(e) => onUpdate(agent.id, { executable: e.target.value })}
              />
            </span>
          </div>
          <p class="settings-hint">{fieldHint(agent.configured_executable)}</p>
          <SettingsField
            label="Effective executable"
            value={agent.effective_executable?.value || '—'}
            mono
            hint={fieldHint(agent.effective_executable)}
          />
          <SettingsField label="Detected version" value={agent.version || '—'} />
          <SettingsField label="Canonical" value={agent.canonical ? 'yes' : 'no'} />
          <SettingsField label="Expected" value={agent.expected || '—'} />
          <SettingsField label="Visibility" value={agent.visibility_class || '—'} />
          <SettingsField label="Live ready" value={agent.live_ready ? 'yes' : 'no'} />
          {#if agent.error}
            <p class="settings-warn">{agent.error}</p>
          {/if}
          {#if agent.status === 'non_canonical'}
            <p class="settings-warn">
              Non-canonical installation. Fleet will not silently use a ChatGPT.app bundle as the
              standalone CLI.
            </p>
          {/if}
          {#if agent.detected_executables?.length}
            <p class="settings-lede">Detected executables</p>
            {#each agent.detected_executables as item}
              <button
                type="button"
                class="settings-path settings-path-btn"
                on:click={() => onUpdate(agent.id, { executable: item.path })}
              >
                {item.source}: {item.path}
              </button>
            {/each}
          {/if}
          <p class="settings-lede">Preferred use</p>
          <textarea
            class="settings-input settings-textarea"
            rows="5"
            placeholder={agent.preferred_use || 'Fleet Best At'}
            value={draft.agents?.[agent.id]?.routingInstructions || ''}
            on:input={(e) => onUpdate(agent.id, { routingInstructions: e.target.value })}
          ></textarea>
          <p class="settings-hint">
            {fieldHint(agent.routing_instructions)} · empty Save keeps Fleet Best At
          </p>
        </div>
      {/if}
    </article>
  {/each}
</section>

{#if agents.notes?.length}
  <ul class="settings-notes">
    {#each agents.notes as note}<li>{note}</li>{/each}
  </ul>
{/if}
