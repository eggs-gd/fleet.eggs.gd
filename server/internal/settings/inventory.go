package settings

func inventoryCatalog() []InventoryEntry {
	return []InventoryEntry{
		entry("core version", "cmd/core/main.go const version", false, false, true, "general", false),
		entry("listen addr", "core serve --addr / server.Config.Addr", false, true, true, "general", false),
		entry("core root", "core serve --root / server.Config.CoreRoot", false, true, true, "projects", false),
		entry("dry-run", "core serve --dry-run", false, true, true, "general", false),
		entry("session timeout", "core serve --session-timeout (idle attention, not a total cap)", false, true, true, "general", false),
		entry("theme", "browser localStorage core.theme via layoutPrefs.js", true, false, true, "general", true),
		entry("auto-refresh interval", "browser localStorage core.autoRefreshMs via layoutPrefs.js", true, false, true, "general", true),
		entry("split widths / accordion / scroll", "browser localStorage layoutPrefs.js", true, false, true, "general", true),
		entry("project scan roots", OverlayFileName+" scanRoots; serve watches them and rewrites _registry when repositories change", true, false, true, "projects", true),
		entry("task storage type", "core.config.yaml taskProvider.type (markdown|plane)", false, true, true, "projects", false),
		entry("plane token", "process env named by taskProvider.tokenEnv; .env is not auto-loaded by serve; never shown in Settings JSON", false, true, false, "projects", false),
		entry("agent executables", OverlayFileName+" agents.*.executable; env CORE_CODEX_BINARY wins when set", true, false, true, "agents", true),
		entry("agent enabled", OverlayFileName+" agents.*.enabled; disable does not kill live sessions", true, false, true, "agents", true),
		entry("agent routing/preferences", OverlayFileName+" routingInstructions overlaying Fleet/<agent>.md Best At", true, false, true, "agents", true),
		entry("manager STT/LLM", "manager.NewService wires UnconfiguredTranscriber + UnconfiguredClassifier", false, false, true, "manager", false),
		entry("manager session", "core.local.yaml manager identity; conversation stays in the provider app", false, false, true, "manager", false),
		entry("concurrency 1-1-1", "Fleet/LAUNCH_POLICY.md + execution session hub locks", false, false, true, "workflow", false),
		entry("operator status transitions", "tasklifecycle.StatusTransitionMap", false, false, true, "workflow", false),
		entry("HITL pause", "execution/host_pause.go; task stays doing, session waiting_input, slot held", false, false, true, "workflow", false),
		entry("finalizer outcome map", "taskflow.MapExecutionOutcomeToStatus (needs_input→blocked if published)", false, false, true, "workflow", false),
		entry("MCP servers", "Fleet MCP written into the Manager provider file at the data root", false, false, true, "integrations", false),
		entry("session/orphan classification", "execution.Status + RuntimeSession.IsActive (same Work Sessions source)", false, false, true, "diagnostics", false),
	}
}

func entry(setting, source string, mutable, restart, safe bool, section string, configurable bool) InventoryEntry {
	return InventoryEntry{
		Setting:         setting,
		CurrentSource:   source,
		RuntimeMutable:  mutable,
		RequiresRestart: restart,
		SafeForGUI:      safe,
		Section:         section,
		ConfigurableNow: configurable,
	}
}
