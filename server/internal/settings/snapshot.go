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
	Version            string
	Addr               string
	LaunchToken        string
	StartedAt          time.Time
	CoreRoot           string
	RuntimeRoot        string
	DryRun             bool
	SessionTimeout     time.Duration
	SessionTimeoutFlag bool
	// LaunchFlag names the flag this process was started with (--live or
	// --dry-run), or "" when the mode came from core.local.yaml or the default.
	LaunchFlag      string
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
	integrations.FleetMCP = InspectManagerMCP(in.CoreRoot, overlay.Manager.Agent, in.Addr, in.LaunchToken)
	sessions := classifySessions(in.RuntimeSessions, in.SessionGroups, in.OrphanedTasks)
	flow := workflowSnapshot()
	snap := Snapshot{
		GeneratedAt:  now.Format(time.RFC3339),
		General:      inspectGeneral(in, overlay, now),
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

func inspectGeneral(in Input, overlay Overlay, now time.Time) General {
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
		Version:              version,
		BuildHash:            gitHash(in.CoreRoot),
		RuntimeStatus:        status,
		LaunchMode:           launch,
		StartedAt:            started,
		Uptime:               uptime,
		Host:                 host,
		APIEndpoint:          endpoint,
		ListenAddr:           in.Addr,
		SessionTimeout:       in.SessionTimeout.String(),
		SessionTimeoutConfig: sessionTimeoutField(overlay, in.SessionTimeoutFlag),
		LaunchConfig:         launchField(overlay, in.LaunchFlag),
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

func launchField(overlay Overlay, flag string) Field {
	configured := strings.TrimSpace(overlay.Launch)
	field := Field{
		Value:    configured,
		Source:   SourceDefault,
		Writable: true,
		Warning:  "Live mode starts AI agents for ready tasks. They edit files in your repositories without asking each time. Dry-run only plans launches. A change applies after restart.",
	}
	if configured != "" {
		field.Source = SourceLocal
	}
	if flag != "" {
		field.OverriddenBy = "fleet serve " + flag
	}
	return field
}

func sessionTimeoutField(overlay Overlay, flagSet bool) Field {
	configured := strings.TrimSpace(overlay.SessionTimeout)
	field := Field{
		Value:    configured,
		Source:   SourceDefault,
		Writable: true,
		Warning:  "Idle attention threshold, not a total session cap. New sessions use the saved value after restart.",
	}
	if configured != "" {
		field.Source = SourceLocal
	}
	if flagSet {
		field.OverriddenBy = "fleet serve --session-timeout"
	}
	return field
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
		effective.OverriddenBy = "FLEET_CODEX_BINARY"
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
		VisibilityClass:      string(executionapi.NormalizeVisibility(row.VisibilityClass)),
		VisibilityLabel:      executionapi.VisibilityClass(row.VisibilityClass).Label(),
		VisibilitySummary:    executionapi.VisibilityClass(row.VisibilityClass).Summary(),
		LiveReady:            row.LiveReady && enabled,
		Capabilities:         row.Capabilities,
		SessionReuse:         row.SessionReuse,
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
		"Manager is a provider session whose working directory is the data root. Conversation stays in that provider's app.",
	}
	if overlay.Manager.Agent != "" && overlay.Manager.ThreadID != "" {
		provider = overlay.Manager.Agent
		session = &ManagerSession{
			ID:        overlay.Manager.ThreadID,
			Status:    "bound",
			StartedAt: overlay.Manager.BoundAt,
			Workspace: overlay.Manager.Workspace,
		}
		notes = append(notes, "Fleet remembers this session. It does not send chat into it.")
	} else {
		notes = append(notes, "No manager session is recorded.")
	}
	notes = append(notes, "Project-level managers are not implemented; this is not modeled as a hard singleton in the task domain.")
	return Manager{
		Role:               "Provider session for this data root. Conversation stays in the provider app.",
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

// ProviderAvailability says whether one supported agent is installed on this
// computer and usable.
type ProviderAvailability struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

// AvailableProviders probes the supported agents (claude, codex, cursor, gemini).
func AvailableProviders() []ProviderAvailability {
	var out []ProviderAvailability
	for _, row := range providers.DiscoverAgents() {
		out = append(out, ProviderAvailability{ID: row.ID, Name: row.Name, Available: row.Status == providers.AgentReady})
	}
	return out
}
