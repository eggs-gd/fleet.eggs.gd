package server

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/providerconfig"
	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
)

func TestCorechainHasNoRuntimeTypeAlias(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{
		"type Runtime =",
		"type Runtime struct",
		"func New(cfg Config)",
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range banned {
			if bytes.Contains(body, []byte(token)) {
				t.Errorf("%s still defines %q; use App/Compose or server.compose", name, token)
			}
		}
	}
}

func TestServerHasNoProxyConstructors(t *testing.T) {
	// Proxy New* wrappers were the CORE-116 rework failure mode. Active
	// (non-ignored) server sources must not redefine these constructors.
	banned := []string{
		"func NewFsWalker(",
		"func NewValidator(",
		"func NewExecutionEntry(",
		"func NewAgentLauncher(",
		"func NewExecutionFinalizer(",
		"func NewCoreFinalizer(",
		"func NewProjector(",
		"func NewProviderSync(",
		"func newListenerService(",
		"type Projector struct",
		"type ProviderSync struct",
		"type namedService struct",
		"func (runtime *server.App) ServiceNames(",
		"func runServices(",
		"type Validator struct",
		"type ExecutionEntry struct",
		"type ContourTopology",
		"func (topology ContourTopology) HasContinuousFsWalkerToAgent",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.HasPrefix(bytes.TrimSpace(body), []byte("//go:build ignore")) {
			continue
		}
		for _, token := range banned {
			if bytes.Contains(body, []byte(token)) {
				t.Errorf("%s still defines %q; call ownership packages directly", name, token)
			}
		}
	}
}

func TestObsoleteProxyFilesAreRemoved(t *testing.T) {
	obsolete := []string{
		"contours.go",
		"listener_bridge.go",
		"listener_service.go",
		"services.go",
		"finalizer.go",
		"projector.go",
		"provider_sync.go",
		"fs_walker.go",
		"validator.go",
		"execution_entry.go",
		"agent_launcher.go",
		"execution_finalizer.go",
		"session_capability.go",
		"provider_runner_claude.go",
		"provider_runner_cursor.go",
		"provider_runner_process.go",
		"provider_runner_cli.go",
		"codex_app_server.go",
		"session_provider_details.go",
		"guardrails_test.go",
		"contours_test.go",
		"validator_test.go",
		"agent_launcher_test.go",
		"fs_walker_test.go",
		"execution_finalizer_test.go",
		"../execution/pipeline.go",
		"../tasklistener/fs_walker.go",
		"../tasklistener/fs_walker_test.go",
	}
	for _, name := range obsolete {
		if _, err := os.Stat(name); err == nil {
			t.Errorf("%s still exists; obsolete proxy/stub files must be removed, not kept as ignored placeholders", name)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
}

func TestPackageImportBoundaries(t *testing.T) {
	assertPackageDoesNotImport(t, "../taskprovider/markdown", "internal/execution", "internal/execution/providers", "internal/executionfinalizer")
	assertPackageDoesNotImport(t, "../execution", "internal/taskprovider/markdown", "internal/taskprovider/plane", "internal/corechain")
	assertPackageDoesNotImport(t, "../executionfinalizer", "internal/taskprovider/markdown", "internal/execution")
	assertPackageDoesNotImport(t, "../manager", "internal/taskprovider/markdown", "internal/execution")
	// Composition root may import execution; provider implementations stay out.
	assertPackageDoesNotImport(t, ".", "internal/taskprovider/markdown", "internal/execution/providers")
}

func TestServerDoesNotConstructCorechainRuntime(t *testing.T) {
	serverDir := "."
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{
		"corechain.New(",
		"corechain.Compose(",
		"corechain.Runtime{",
		"corechain.App{",
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(serverDir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range banned {
			if bytes.Contains(body, []byte(token)) {
				t.Errorf("%s still constructs %s; production composition must stay in server.compose", name, token)
			}
		}
	}
}

func assertPackageDoesNotImport(t *testing.T, dir string, forbidden ...string) {
	t.Helper()
	filter := func(info os.FileInfo) bool {
		name := info.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}
	pkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Clean(dir), filter, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, spec := range file.Imports {
				path := strings.Trim(spec.Path.Value, `"`)
				for _, banned := range forbidden {
					if strings.Contains(path, banned) {
						t.Fatalf("%s imports forbidden dependency %s", file.Name.Name, path)
					}
				}
			}
		}
	}
}

func TestRuntimeWiresIndependentServices(t *testing.T) {
	root := t.TempDir()
	md := Compose(ComposeConfig{CoreRoot: root, Interval: time.Second, DryRun: true})
	assertProcessorsWired(t, md)

	pl := Compose(ComposeConfig{
		CoreRoot: root,
		Interval: time.Second,
		DryRun:   true,
		TaskProvider: providerconfig.Settings{
			Type: "plane",
			Plane: providerconfig.PlaneSettings{
				Workspace:   "eggs_gd",
				ProjectID:   "proj-1",
				CoreProject: "core-eggs-gd",
				TokenEnv:    "CORE_TEST_PLANE_TOKEN_UNSET",
				BaseURL:     "https://example.invalid",
			},
		},
	})
	assertProcessorsWired(t, pl)
}

func assertProcessorsWired(t *testing.T, app *App) {
	t.Helper()
	if app.Listener == nil {
		t.Fatal("listener processor not wired")
	}
	if app.Exec == nil {
		t.Fatal("execution host not wired")
	}
	if app.FinalizerProc == nil {
		t.Fatal("finalizer processor not wired")
	}
}

func TestExecutionPackageHasNoProviderBranches(t *testing.T) {
	dir := filepath.Clean("../execution")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{
		"taskprovider/markdown",
		"taskprovider/plane",
		"Provider.Type",
		"TaskEventSourceMarkdownFS",
		"TaskEventSourcePlanePoll",
		"TaskEventSourcePlaneWebhook",
		"Source ==",
		"Source !=",
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range banned {
			if bytes.Contains(body, []byte(token)) {
				t.Errorf("%s contains %q; execution must stay provider-neutral", entry.Name(), token)
			}
		}
	}
}

func TestExecutionPackageDoesNotCallReportExecution(t *testing.T) {
	dir := filepath.Clean("../execution")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(body, []byte(".ReportExecution(")) {
			t.Errorf("%s must not call ReportExecution; finalization owns lifecycle writes", entry.Name())
		}
	}
}

func TestCorechainDoesNotCallReportExecution(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(body, []byte(".ReportExecution(")) {
			t.Errorf("%s must not call ReportExecution directly; publish ExecutionResult instead", entry.Name())
		}
	}
}

func TestDashboardStaysOutsideExecutionWiring(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), Interval: time.Second, DryRun: true})
	var surface DashboardSurface = app.Surface()
	_ = surface.State()
	if app.TaskService() == nil {
		t.Fatal("Dashboard.TaskService must remain wired for Manager")
	}
	var _ taskflow.TaskService = app.TaskService()
}

func TestDashboardPackageHasNoChannelOrExecutionCoupling(t *testing.T) {
	// HTTP dashboard handlers must stay poll/command-only. Composition root
	// (compose.go / server.go) is allowed to wire execution siblings.
	serverDir := "."
	dashboardFiles := []string{"dashboard.go"}

	bannedImportFragments := []string{
		"internal/taskprovider",
		"lib/chain",
		"internal/execution/providers",
		"internal/executionfinalizer",
	}
	bannedTokens := []string{
		"ExecutionResults(",
		"taskEvents",
		"executionResults",
		"NewFsWalker",
		"NewExecutionEntry",
		"NewAgentLauncher",
		"NewProviderSync",
		"NewExecutionFinalizer",
		"namedService",
		"ServiceNames",
		"runServices",
		"newListenerService",
	}

	for _, name := range dashboardFiles {
		path := filepath.Join(serverDir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, body, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			pathLit := strings.Trim(imp.Path.Value, `"`)
			for _, frag := range bannedImportFragments {
				if strings.Contains(pathLit, frag) {
					t.Errorf("%s imports %q; dashboard must stay on TaskService/RuntimeService", path, pathLit)
				}
			}
		}

		for _, tok := range bannedTokens {
			if bytes.Contains(body, []byte(tok)) {
				t.Errorf("%s references %q; dashboard must not couple to execution/channels", path, tok)
			}
		}
		if serverSourceUsesChannels(t, path, body) {
			t.Errorf("%s uses Go channels; dashboard must poll REST only", path)
		}
	}
}

func serverSourceUsesChannels(t *testing.T, path string, body []byte) bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, body, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ChanType, *ast.SendStmt:
			found = true
			return false
		case *ast.UnaryExpr:
			if node.Op == token.ARROW {
				found = true
				return false
			}
		}
		return true
	})
	return found
}
