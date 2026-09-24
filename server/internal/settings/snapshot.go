package settings

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"
	"github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// Input is live process + board/runtime state used to build Snapshot.
type Input struct {
	Version         string
	Addr            string
	StartedAt       time.Time
	CoreRoot        string
	RuntimeRoot     string
	DryRun          bool
	SessionTimeout  time.Duration
	Workspaces      []board.Workspace
	Projects        []board.Project
	Registry        board.RegistryInfo
	Tasks           []tasklifecycle.Task
	RuntimeSessions []executionapi.RuntimeSession
	SessionGroups   []execution.SessionGroup
	OrphanedTasks   []execution.OrphanedTask
	Scan            ScanStatus
}

// Build returns the Settings inspect model with overlay merged for display.
func Build(in Input) Snapshot {
	now := time.Now()
	overlay, overlayErr := LoadOverlay(in.CoreRoot)
	ApplyOverlay(overlay)
	projects := inspectProjects(in.CoreRoot, in.RuntimeRoot, in.Workspaces, in.Projects, in.Registry, overlay, in.Scan)
	if overlayErr != nil {
		projects.Notes = append(projects.Notes, "Failed to load "+OverlayFileName+": "+overlayErr.Error())
	}
	agents := inspectAgents(in.CoreRoot, overlay, in.RuntimeSessions)
	manager := inspectManager(in.CoreRoot, overlay)
	integrations := loadMCP(in.CoreRoot)
	sessions := classifySessions(in.RuntimeSessions, in.SessionGroups, in.OrphanedTasks)
	flow := workflowSnapshot()
	snap := Snapshot{
		GeneratedAt:  now.Format(time.RFC3339),
		General:      inspectGeneral(in, now),
		Projects:     projects,
		Agents:       agents,
		Manager:      manager,
		Workflow:     flow,
		Integrations: integrations,
		Inventory:    inventoryCatalog(),
	}
	snap.Diagnostics = inspectDiagnostics(projects, agents, manager, integrations, sessions)
	return snap
}

func inspectGeneral(in Input, now time.Time) General {
	launch := "live"
	status := "ok"
	if in.DryRun {
		launch = "dry-run"
		status = "dry-run"
	}
	host, _ := os.Hostname()
	endpoint := ""
	if strings.TrimSpace(in.Addr) != "" {
		endpoint = "http://" + in.Addr + "/"
	}
	uptime := ""
	started := ""
	if !in.StartedAt.IsZero() {
		started = in.StartedAt.Format(time.RFC3339)
		uptime = now.Sub(in.StartedAt).Truncate(time.Second).String()
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		version = "unknown"
	}
	return General{
		Version:        version,
		BuildHash:      gitHash(in.CoreRoot),
		RuntimeStatus:  status,
		LaunchMode:     launch,
		StartedAt:      started,
		Uptime:         uptime,
		Host:           host,
		APIEndpoint:    endpoint,
		ListenAddr:     in.Addr,
		SessionTimeout: in.SessionTimeout.String(),
		AutoRefresh: ClientPref{
			Value:         "browser localStorage core.autoRefreshMs",
			Source:        "layoutPrefs.js",
			Stored:        "localStorage",
			EditableInGUI: true,
		},
		Theme: ClientPref{
			Value:         "system|light|dark",
			Source:        "browser localStorage key core.theme (layoutPrefs.js / ThemeSwitch)",
			Stored:        "localStorage",
			EditableInGUI: true,
		},
		Startup: []StartupFact{
			{ID: "hydrate", Label: "Hydrate Work/ and _registry", Value: "always", Source: "Dashboard.Bootstrap → taskprovider.Hydrate", Configurable: false},
			{ID: "reconcile", Label: "Reconcile persisted sessions", Value: "always", Source: "execution.ReconcilePersistedSessions", Configurable: false},
			{ID: "orphans", Label: "Detect startup orphans", Value: "always", Source: "execution.DetectStartupOrphans", Configurable: false},
			{ID: "index", Label: "Rebuild Work/INDEX.md", Value: "markdown provider", Source: "markdown.RebuildDerivedViews", Configurable: false},
			{ID: "launch", Label: "Launch pending pickup tasks", Value: launchValue(in.DryRun), Source: "App.Run → LaunchPendingTasks when not dry-run", Configurable: false},
			{ID: "sniff", Label: "Project tree scan", Value: "while serve is running", Source: "settings.Scanner.Watch → projectscan.Scan", Configurable: true},
		},
		Notes: []string{
			"Startup on/off toggles from the product spec are not implemented; facts above are observed bootstrap behaviour.",
			"Theme and auto-refresh are browser preferences. They apply immediately and are not written to core.local.yaml.",
			"Resume of resumable sessions is classified at bootstrap (attach vs orphaned-but-resumable), not an ask/off/on setting.",
		},
	}
}

func launchValue(dryRun bool) string {
	if dryRun {
		return "skipped (dry-run)"
	}
	return "on when live"
}

func gitHash(root string) string {
	if strings.TrimSpace(root) == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", root, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func inspectAgents(root string, overlay Overlay, sessions []executionapi.RuntimeSession) Agents {
	discovered := providers.DiscoverAgents()
	out := Agents{
		Notes: []string{
			"Only providers registered in execution/providers.NewRegistry() are listed (claude, codex, cursor, gemini).",
			"Preferred use is overlay routingInstructions, else Fleet/<agent>.md “Best At”. Fleet files are not rewritten.",
			"Disable blocks new launches only. Live sessions stay running.",
			"Recheck probes PATH/env/bundled and does not write core.local.yaml.",
			"Merge order: defaults < core.config.yaml < core.local.yaml < environment.",
		},
	}
	for _, row := range discovered {
		out.Providers = append(out.Providers, composeAgent(root, overlay, row, sessions))
	}
	return out
}

func composeAgent(root string, overlay Overlay, row providers.AgentDiscovery, sessions []executionapi.RuntimeSession) Agent {
	local := overlay.Agent(row.ID)
	enabled := true
	enabledSource := SourceDefault
	if local.Enabled != nil {
		enabled = *local.Enabled
		enabledSource = SourceLocal
	}
	fleetUse, _ := fleetPreferredUse(root, row.ID)
	routing := Field{Value: fleetUse, Source: SourceFleet, Writable: true}
	if strings.TrimSpace(local.RoutingInstructions) != "" {
		routing = Field{Value: local.RoutingInstructions, Source: SourceLocal, Writable: true}
	}
	configured := Field{Value: local.Executable, Source: SourceDefault, Writable: true}
	if strings.TrimSpace(local.Executable) != "" {
		configured.Source = SourceLocal
	}
	effective := Field{
		Value:    row.EffectiveExecutable,
		Source:   firstNonEmpty(row.EffectiveSource, row.DiscoverySource),
		Writable: false,
	}
	if effective.Source == SourceEnv {
		effective.OverriddenBy = "CORE_CODEX_BINARY"
	}
	return Agent{
		ID:                   row.ID,
		Name:                 row.Name,
		Status:               row.Status,
		Enabled:              BoolField{Value: enabled, Source: enabledSource, Writable: true},
		ConfiguredExecutable: configured,
		DetectedExecutables:  row.DetectedExecutables,
		EffectiveExecutable:  effective,
		Canonical:            row.Canonical,
		Expected:             row.Expected,
		Version:              row.Version,
		Error:                row.Error,
		VisibilityClass:      row.VisibilityClass,
		LiveReady:            row.LiveReady && enabled,
		RoutingInstructions:  routing,
		PreferredUse:         routing.Value,
		ActiveSessions:       countActiveAgentSessions(sessions, row.ID),
	}
}

func countActiveAgentSessions(sessions []executionapi.RuntimeSession, agent string) int {
	n := 0
	for _, session := range sessions {
		if !session.IsActive() {
			continue
		}
		if strings.EqualFold(session.Agent, agent) {
			n++
		}
	}
	return n
}

func inspectManager(root string, overlay Overlay) Manager {
	instructions, src := fleetRouting(root)
	provider := "none (HTTP Manager API)"
	var session *ManagerSession
	notes := []string{
		"manager.NewService wires UnconfiguredTranscriber and UnconfiguredClassifier.",
		"Manager Bar posts text to /api/manager/text.",
	}
	if overlay.Manager.Agent != "" && overlay.Manager.ThreadID != "" {
		provider = overlay.Manager.Agent
		session = &ManagerSession{
			ID:        overlay.Manager.ThreadID,
			Status:    "bound",
			StartedAt: overlay.Manager.BoundAt,
		}
		notes = append(notes, "Manager Bar messages route directly into this bound session instead of fast-path command parsing.")
	} else {
		notes = append(notes, "No manager session is bound; Manager Bar text goes through fast-path command parsing only.")
	}
	notes = append(notes, "Project-level managers are not implemented; this is not modeled as a hard singleton in the task domain.")
	return Manager{
		Role:               "Local command interpreter for the Manager Bar (not an executor provider)",
		Provider:           provider,
		Session:            session,
		STT:                "unconfigured",
		Classifier:         "unconfigured",
		FastPath:           true,
		Endpoints:          []string{"/api/manager/schema", "/api/manager/vocabulary", "/api/manager/text"},
		Instructions:       instructions,
		InstructionsSource: src,
		Notes:              notes,
	}
}
