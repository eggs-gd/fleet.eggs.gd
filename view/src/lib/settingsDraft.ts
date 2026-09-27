import type {
  SettingsSnapshot,
  SettingsDraft,
  SettingsDraftAgent,
  SettingsFieldMeta
} from './types.ts';

export function draftFromSnapshot(snapshot?: SettingsSnapshot | null): SettingsDraft {
  const agents: SettingsDraft['agents'] = {};
  for (const agent of snapshot?.agents?.providers || []) {
    if (!agent.id) continue;
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
    scanRoots: Array.isArray(snapshot?.projects?.scan_roots)
      ? [...(snapshot!.projects!.scan_roots as string[])]
      : [],
    sessionTimeout: snapshot?.general?.session_timeout_config?.value || '',
    launch: snapshot?.general?.launch_config?.value || '',
    agents,
    manager: {
      agent: snapshot?.manager?.session ? snapshot.manager.provider || '' : '',
      threadId: snapshot?.manager?.session?.id || ''
    }
  };
}

export function patchFromDraft(
  draft?: SettingsDraft | null,
  baseline?: SettingsDraft | null
): Record<string, unknown> {
  const patch: Record<string, unknown> = {};
  if (JSON.stringify(draft?.scanRoots || []) !== JSON.stringify(baseline?.scanRoots || [])) {
    patch.scanRoots = draft?.scanRoots || [];
  }
  if ((draft?.sessionTimeout || '') !== (baseline?.sessionTimeout || '')) {
    patch.sessionTimeout = draft?.sessionTimeout || '';
  }
  if ((draft?.launch || '') !== (baseline?.launch || '')) {
    patch.launch = draft?.launch || '';
  }
  const agents: SettingsDraft['agents'] = {};
  for (const id of Object.keys(draft?.agents || {})) {
    const cur = draft?.agents?.[id] || {};
    const prev = baseline?.agents?.[id] || {};
    const item: SettingsDraftAgent = {};
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
  if (
    (managerDraft.agent || '') !== (managerBase.agent || '') ||
    (managerDraft.threadId || '') !== (managerBase.threadId || '')
  ) {
    patch.manager = { agent: managerDraft.agent || '', threadId: managerDraft.threadId || '' };
  }
  return patch;
}

export function draftsEqual(a: unknown, b: unknown): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

export function fieldHint(field?: SettingsFieldMeta | null): string {
  if (!field) return '';
  const parts: string[] = [];
  if (field.source) parts.push(field.source);
  if (field.overridden_by) parts.push(`overridden by ${field.overridden_by}`);
  if (field.warning) parts.push(field.warning);
  return parts.join(' · ');
}
