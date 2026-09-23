export function draftFromSnapshot(snapshot) {
  const agents = {};
  for (const agent of snapshot?.agents?.providers || []) {
    agents[agent.id] = {
      enabled: agent.enabled?.value !== false,
      executable: agent.configured_executable?.value || '',
      routingInstructions:
        agent.routing_instructions?.source === 'core.local.yaml'
          ? agent.routing_instructions.value || ''
          : ''
    };
  }
  return {
    scanRoot: snapshot?.projects?.scan_root?.value || '',
    agents,
    manager: {
      agent: snapshot?.manager?.session ? snapshot.manager.provider || '' : '',
      threadId: snapshot?.manager?.session?.id || ''
    }
  };
}

export function patchFromDraft(draft, baseline) {
  const patch = {};
  if ((draft?.scanRoot || '') !== (baseline?.scanRoot || '')) {
    patch.scanRoot = draft.scanRoot;
  }
  const agents = {};
  for (const id of Object.keys(draft?.agents || {})) {
    const cur = draft.agents[id] || {};
    const prev = baseline?.agents?.[id] || {};
    const item = {};
    if (cur.enabled !== prev.enabled) item.enabled = cur.enabled;
    if ((cur.executable || '') !== (prev.executable || '')) item.executable = cur.executable || '';
    if ((cur.routingInstructions || '') !== (prev.routingInstructions || '')) {
      item.routingInstructions = cur.routingInstructions || '';
    }
    if (Object.keys(item).length) agents[id] = item;
  }
  if (Object.keys(agents).length) patch.agents = agents;
  const managerDraft = draft?.manager || {};
  const managerBase = baseline?.manager || {};
  if ((managerDraft.agent || '') !== (managerBase.agent || '') || (managerDraft.threadId || '') !== (managerBase.threadId || '')) {
    patch.manager = { agent: managerDraft.agent || '', threadId: managerDraft.threadId || '' };
  }
  return patch;
}

export function draftsEqual(a, b) {
  return JSON.stringify(a) === JSON.stringify(b);
}

export function fieldHint(field) {
  if (!field) return '';
  const parts = [];
  if (field.source) parts.push(field.source);
  if (field.overridden_by) parts.push(`overridden by ${field.overridden_by}`);
  if (field.warning) parts.push(field.warning);
  return parts.join(' · ');
}
