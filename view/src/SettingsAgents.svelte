<script lang="ts">
  import SettingsField from './SettingsField.svelte';
  import { fieldHint } from './lib/settingsDraft';
  import {
    capabilityRows,
    launchesAutomatically,
    plainText,
    reuseSummary
  } from './lib/agentCapabilities';
  import type { AnyRecord, SettingsDraft, SettingsDraftAgent } from './lib/types';

  export let agents: AnyRecord = {};
  export let draft: SettingsDraft = { agents: {} };
  export let onUpdate: (id: string, patch: SettingsDraftAgent) => void = () => {};
  export let onRecheck: () => void = () => {};
  export let dirty = false;

  let openId = '';

  function toggle(id: string) {
    openId = openId === id ? '' : id;
  }

  const statusTone = (agent: AnyRecord) => {
    if (agent.enabled?.value === false) return 'muted';
    if (agent.status === 'ready') return 'ok';
    if (agent.status === 'non_canonical') return 'warn';
    if (agent.status === 'not_found' || agent.status === 'error') return 'fail';
    return 'muted';
  };

  const statusLabel = (agent: AnyRecord) => {
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
                  on:change={(e) => onUpdate(agent.id, { enabled: e.currentTarget.checked })}
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
                on:input={(e) => onUpdate(agent.id, { executable: e.currentTarget.value })}
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
          <p class="settings-lede">What Fleet can do with {agent.name}</p>
          <SettingsField
            label="Starts by itself"
            value={launchesAutomatically(agent) ? 'yes' : 'no'}
            tone={launchesAutomatically(agent) ? 'ok' : 'warn'}
            hint="Yes when the adapter is verified and enabled, and a person can reach the session."
          />
          <SettingsField
            label="How you reach a session"
            value={agent.visibility_label || agent.visibility_class || '—'}
            hint={plainText(agent.visibility_summary)}
          />
          {#each capabilityRows(agent) as row (row.key)}
            <SettingsField label={row.label} value={row.value} tone={row.tone} />
          {/each}
          <SettingsField
            label="Resume for the next task"
            value={reuseSummary(agent).label}
            hint={reuseSummary(agent).reason}
          />
          {#if agent.capabilities?.terminal_state_detection}
            <SettingsField
              label="Knows it finished by"
              value={agent.capabilities.terminal_state_detection}
            />
          {/if}
          {#if agent.capabilities?.notes}
            <p class="settings-hint">{plainText(agent.capabilities.notes)}</p>
          {/if}
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
            on:input={(e) => onUpdate(agent.id, { routingInstructions: e.currentTarget.value })}
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
