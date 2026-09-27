package settings

import "github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"

const (
	SourceDefault = "default"
	SourceConfig  = "core.config.yaml"
	SourceLocal   = "core.local.yaml"
	SourceEnv     = "env"
	SourcePath    = "path"
	SourceBundled = "bundled"
	SourceFleet   = "fleet"
)

type Snapshot struct {
	GeneratedAt  string           `json:"generated_at"`
	General      General          `json:"general"`
	Projects     Projects         `json:"projects"`
	Agents       Agents           `json:"agents"`
	Manager      Manager          `json:"manager"`
	Workflow     Workflow         `json:"workflow"`
	Integrations Integrations     `json:"integrations"`
	Diagnostics  Diagnostics      `json:"diagnostics"`
	Inventory    []InventoryEntry `json:"inventory"`
}

type Field struct {
	Value        string `json:"value"`
	Source       string `json:"source"`
	Writable     bool   `json:"writable"`
	OverriddenBy string `json:"overridden_by,omitempty"`
	Warning      string `json:"warning,omitempty"`
}

type BoolField struct {
	Value        bool   `json:"value"`
	Source       string `json:"source"`
	Writable     bool   `json:"writable"`
	OverriddenBy string `json:"overridden_by,omitempty"`
}

type General struct {
	Version              string        `json:"version"`
	BuildHash            string        `json:"build_hash,omitempty"`
	RuntimeStatus        string        `json:"runtime_status"`
	LaunchMode           string        `json:"launch_mode"`
	StartedAt            string        `json:"started_at,omitempty"`
	Uptime               string        `json:"uptime,omitempty"`
	Host                 string        `json:"host,omitempty"`
	APIEndpoint          string        `json:"api_endpoint,omitempty"`
	ListenAddr           string        `json:"listen_addr,omitempty"`
	SessionTimeout       string        `json:"session_timeout"`
	SessionTimeoutConfig Field         `json:"session_timeout_config"`
	LaunchConfig         Field         `json:"launch_config"`
	AutoRefresh          ClientPref    `json:"auto_refresh"`
	Theme                ClientPref    `json:"theme"`
	Startup              []StartupFact `json:"startup"`
	Notes                []string      `json:"notes"`
}

type ClientPref struct {
	Value         string `json:"value"`
	Source        string `json:"source"`
	Stored        string `json:"stored"`
	EditableInGUI bool   `json:"editable_in_gui"`
}

type StartupFact struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Value        string `json:"value"`
	Source       string `json:"source"`
	Configurable bool   `json:"configurable"`
}

type Projects struct {
	MultiRootSupported bool          `json:"multi_root_supported"`
	ScanRoots          []string      `json:"scan_roots"`
	ScanRootsWritable  bool          `json:"scan_roots_writable"`
	ScanRootsSource    string        `json:"scan_roots_source,omitempty"`
	Scan               ScanStatus    `json:"scan"`
	Roots              []ProjectRoot `json:"roots"`
	DataPaths          []DataPath    `json:"data_paths"`
	TaskBackend        TaskBackend   `json:"task_backend"`
	Notes              []string      `json:"notes"`
}

type ProjectRoot struct {
	Kind            string `json:"kind"`
	Path            string `json:"path"`
	Reachable       bool   `json:"reachable"`
	RepositoryCount int    `json:"repository_count,omitempty"`
	WorkspaceCount  int    `json:"workspace_count,omitempty"`
	ProjectCount    int    `json:"project_count,omitempty"`
	LastScan        string `json:"last_scan,omitempty"`
	Enabled         bool   `json:"enabled"`
	Source          string `json:"source"`
	Notes           string `json:"notes,omitempty"`
}

type DataPath struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Writable bool   `json:"writable"`
	Role     string `json:"role"`
}

type TaskBackend struct {
	Active           string        `json:"active"`
	ConfigFile       string        `json:"config_file"`
	ConfigFileExists bool          `json:"config_file_exists"`
	Implementations  []string      `json:"implementations"`
	PlaneDeclared    bool          `json:"plane_declared,omitempty"`
	Plane            *PlaneInspect `json:"plane,omitempty"`
}

type PlaneInspect struct {
	Workspace   string            `json:"workspace,omitempty"`
	BaseURL     string            `json:"base_url,omitempty"`
	TokenEnv    string            `json:"token_env,omitempty"`
	TokenEnvSet bool              `json:"token_env_set"`
	CoreProject string            `json:"core_project,omitempty"`
	Repository  string            `json:"repository,omitempty"`
	StatusMap   map[string]string `json:"status_map,omitempty"`
	Active      bool              `json:"active"`
	Notes       string            `json:"notes,omitempty"`
}

type Agents struct {
	Providers []Agent  `json:"providers"`
	Notes     []string `json:"notes"`
}

type Agent struct {
	ID                   string                         `json:"id"`
	Name                 string                         `json:"name"`
	Status               string                         `json:"status"`
	Enabled              BoolField                      `json:"enabled"`
	ConfiguredExecutable Field                          `json:"configured_executable"`
	DetectedExecutables  []providers.DetectedExecutable `json:"detected_executables,omitempty"`
	EffectiveExecutable  Field                          `json:"effective_executable"`
	Canonical            bool                           `json:"canonical"`
	Expected             string                         `json:"expected,omitempty"`
	Version              string                         `json:"version,omitempty"`
	Error                string                         `json:"error,omitempty"`
	VisibilityClass      string                         `json:"visibility_class,omitempty"`
	LiveReady            bool                           `json:"live_ready"`
	RoutingInstructions  Field                          `json:"routing_instructions"`
	PreferredUse         string                         `json:"preferred_use,omitempty"`
	ActiveSessions       int                            `json:"active_sessions"`
}

type Manager struct {
	Role               string          `json:"role"`
	Provider           string          `json:"provider"`
	Session            *ManagerSession `json:"session"`
	STT                string          `json:"stt"`
	Classifier         string          `json:"classifier"`
	FastPath           bool            `json:"fast_path"`
	Endpoints          []string        `json:"endpoints"`
	Instructions       string          `json:"instructions,omitempty"`
	InstructionsSource string          `json:"instructions_source,omitempty"`
	Notes              []string        `json:"notes"`
}

type ManagerSession struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

type Workflow struct {
	Concurrency         []ConcurrencyScope  `json:"concurrency"`
	Statuses            []string            `json:"statuses"`
	OperatorTransitions map[string][]string `json:"operator_transitions"`
	RuntimeOutcomes     []OutcomeRow        `json:"runtime_outcomes"`
	HITL                HITLContract        `json:"hitl"`
	Notes               []string            `json:"notes"`
}

type ConcurrencyScope struct {
	Scope  string `json:"scope"`
	Limit  int    `json:"limit"`
	Source string `json:"source"`
}

type OutcomeRow struct {
	Outcome      string `json:"outcome"`
	Task         string `json:"task"`
	Session      string `json:"session"`
	ReleasesSlot bool   `json:"releases_slot"`
	Source       string `json:"source"`
}

type HITLContract struct {
	TaskStatus           string `json:"task_status"`
	SessionStatus        string `json:"session_status"`
	ReleasesSlot         bool   `json:"releases_slot"`
	EqualsClosed         bool   `json:"equals_closed"`
	Source               string `json:"source"`
	FinalizerIfPublished string `json:"finalizer_if_published"`
}

type Integrations struct {
	SourceFile string         `json:"source_file"`
	SourceRole string         `json:"source_role"`
	MCP        []MCPServer    `json:"mcp"`
	FleetMCP   FleetMCPStatus `json:"fleet_mcp"`
	Notes      []string       `json:"notes"`
}

type MCPServer struct {
	Name         string `json:"name"`
	Transport    string `json:"transport,omitempty"`
	Command      string `json:"command,omitempty"`
	Configured   bool   `json:"configured"`
	Detected     bool   `json:"detected"`
	Reachable    string `json:"reachable"`
	Expected     string `json:"expected,omitempty"`
	Error        string `json:"error,omitempty"`
	InstallKnown bool   `json:"install_known"`
}

type Diagnostics struct {
	Health   []HealthItem  `json:"health"`
	Sessions SessionCounts `json:"sessions"`
	Issues   []Issue       `json:"issues"`
	Notes    []string      `json:"notes"`
}

type HealthItem struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type SessionCounts struct {
	Active            int `json:"active"`
	HITL              int `json:"hitl"`
	Closed            int `json:"closed"`
	Failed            int `json:"failed"`
	Orphaned          int `json:"orphaned"`
	OrphanedResumable int `json:"orphaned_resumable"`
}

type Issue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Section  string `json:"section"`
}

type InventoryEntry struct {
	Setting         string `json:"setting"`
	CurrentSource   string `json:"current_source"`
	RuntimeMutable  bool   `json:"runtime_mutable"`
	RequiresRestart bool   `json:"requires_restart"`
	SafeForGUI      bool   `json:"safe_for_gui"`
	Section         string `json:"section"`
	ConfigurableNow bool   `json:"configurable_now"`
}
