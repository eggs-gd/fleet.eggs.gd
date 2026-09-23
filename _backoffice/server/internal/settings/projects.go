package settings

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/eggs-gd/core.eggs.gd/internal/board"
	"github.com/eggs-gd/core.eggs.gd/internal/providerconfig"
)

func inspectProjects(root string, workspaces []board.Workspace, projects []board.Project, registry board.RegistryInfo, overlay Overlay, scan ScanStatus) Projects {
	coreRoot := absPath(root)
	meta := readRegistryMeta(filepath.Join(coreRoot, "_registry", "repositories.json"))
	scanValue, scanSource := EffectiveScanRoot(overlay)
	if scan.State == "" {
		scan.State = ScanIdle
	}
	if scan.RepositoryCount == 0 {
		scan.RepositoryCount = registry.RepositoriesCount
	}
	out := Projects{
		MultiRootSupported: false,
		ScanRoot:           Field{Value: scanValue, Source: scanSource, Writable: true},
		Scan:               scan,
		Roots: []ProjectRoot{
			{
				Kind:           "core_root",
				Path:           coreRoot,
				Reachable:      dirExists(coreRoot),
				WorkspaceCount: len(workspaces),
				ProjectCount:   len(projects),
				Enabled:        true,
				Source:         "core serve --root / server.Config.CoreRoot",
				Notes:          "Task cards and workspace PROJECT.md files live under Work/ in this tree. Serve does not scan arbitrary extra roots.",
			},
		},
		Notes: []string{
			"Multiple project roots are not implemented. The daemon has one Core root.",
			"Rescan runs sniff_projects.py against the saved scan root, then reloads _registry into the live board.",
			"Personal tasks/context still live inside the Core repository (Work/, _registry/). Moving them to ~/.eggs-core/ is not implemented.",
		},
	}
	lastScanRoot := stringsOr(meta.Root, "")
	if lastScanRoot != "" {
		out.Roots = append(out.Roots, ProjectRoot{
			Kind:            "registry_scan",
			Path:            lastScanRoot,
			Reachable:       dirExists(lastScanRoot),
			RepositoryCount: registry.RepositoriesCount,
			LastScan:        meta.GeneratedAt,
			Enabled:         true,
			Source:          "_registry/repositories.json root/generated_at (last sniff_projects.py run)",
			Notes:           "Last scan output consumed by the dashboard. Editor above is the next Rescan root.",
		})
	} else if registry.RepositoriesCount > 0 {
		out.Roots[0].RepositoryCount = registry.RepositoriesCount
		out.Roots[0].LastScan = meta.GeneratedAt
	}

	out.DataPaths = []DataPath{
		inspectPath("Core root", coreRoot, "process root"),
		inspectPath("Tasks", filepath.Join(coreRoot, "Work"), "Markdown task cards Work/<project>/tasks"),
		inspectPath("Workspaces", filepath.Join(coreRoot, "Work"), "PROJECT.md workspace cards"),
		inspectPath("Registry", filepath.Join(coreRoot, "_registry"), "generated discovery/runtime state"),
		inspectPath("Sessions", filepath.Join(coreRoot, "_registry", "sessions"), "runtime session logs"),
		inspectPath("Config", filepath.Join(coreRoot, providerconfig.ConfigFileName), "task provider selection"),
		inspectPath("Local overlay", OverlayPath(coreRoot), "gitignored machine overlay; atomic Save target"),
		inspectPath("MCP", filepath.Join(coreRoot, ".mcp.json"), "Cursor project MCP servers"),
	}
	out.TaskBackend = inspectTaskBackend(coreRoot)
	return out
}

type registryFileMeta struct {
	GeneratedAt string `json:"generated_at"`
	Root        string `json:"root"`
}

func readRegistryMeta(path string) registryFileMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return registryFileMeta{}
	}
	var meta registryFileMeta
	_ = json.Unmarshal(data, &meta)
	return meta
}

func inspectTaskBackend(root string) TaskBackend {
	cfgPath := filepath.Join(root, providerconfig.ConfigFileName)
	loaded, err := providerconfig.Load(root)
	backend := TaskBackend{
		Active:           "markdown",
		ConfigFile:       cfgPath,
		ConfigFileExists: fileExists(cfgPath),
		Implementations:  []string{"markdown", "plane"},
	}
	if err != nil {
		backend.Active = "error"
		return backend
	}
	backend.Active = loaded.Type
	plane := loaded.Plane
	declared := loaded.Type == "plane" || plane.Workspace != "" || plane.TokenEnv != "" || plane.CoreProject != ""
	backend.PlaneDeclared = declared
	if !declared && loaded.Type != "plane" {
		return backend
	}
	tokenEnv := stringsOr(plane.TokenEnv, "PLANE_API_TOKEN")
	backend.Plane = &PlaneInspect{
		Workspace:   plane.Workspace,
		BaseURL:     plane.BaseURL,
		TokenEnv:    tokenEnv,
		TokenEnvSet: os.Getenv(tokenEnv) != "",
		CoreProject: plane.CoreProject,
		Repository:  plane.Repository,
		StatusMap:   plane.StatusMap,
		Active:      loaded.Type == "plane",
		Notes:       "Token value is never included. token_env_set is whether the named env var is non-empty.",
	}
	if loaded.Type != "plane" {
		backend.Plane.Notes += " Plane fields are present in " + providerconfig.ConfigFileName + " but taskProvider.type is markdown, so Plane is not the active backend."
	}
	if fileExists(filepath.Join(root, ".env")) {
		backend.Plane.Notes += " .env exists under the Core root, but core serve does not load it. token_env_set is the process environment, not the file."
	}
	return backend
}

func inspectPath(name, path, role string) DataPath {
	exists := pathExists(path)
	return DataPath{
		Name:     name,
		Path:     path,
		Exists:   exists,
		Writable: exists && pathWritable(path),
		Role:     role,
	}
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func pathWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0o200 != 0
}

func stringsOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
