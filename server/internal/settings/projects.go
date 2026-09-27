package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/providerconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
)

func inspectProjects(root, runtimeRoot string, workspaces []board.Workspace, projects []board.Project, registry board.RegistryInfo, overlay Overlay, scan ScanStatus) Projects {
	coreRoot := absPath(root)
	meta := readRegistryMeta(filepath.Join(coreRoot, "_registry", "repositories.json"))
	scanRoots, scanSource := EffectiveScanRoots(overlay)
	if scan.State == "" {
		scan.State = ScanIdle
	}
	if scan.RepositoryCount == 0 {
		scan.RepositoryCount = registry.RepositoriesCount
	}
	out := Projects{
		MultiRootSupported: true,
		ScanRoots:          scanRoots,
		ScanRootsWritable:  true,
		ScanRootsSource:    scanSource,
		Scan:               scan,
		Roots: []ProjectRoot{
			{
				Kind:           "core_root",
				Path:           coreRoot,
				Reachable:      dirExists(coreRoot),
				WorkspaceCount: len(workspaces),
				ProjectCount:   len(projects),
				Enabled:        true,
				Source:         "fleet serve --root / server.Config.CoreRoot",
				Notes:          "Task cards and workspace PROJECT.md files live under Work/ in this tree. Project checkouts are scanned from the scan roots above.",
			},
		},
		Notes: []string{
			"Several project directories can be watched. The daemon still has one Data root.",
			"The running server watches every scan root and rewrites _registry when repositories appear or disappear, then reloads the board.",
		},
	}
	for _, scanRoot := range meta.scanRoots() {
		out.Roots = append(out.Roots, ProjectRoot{
			Kind:            "registry_scan",
			Path:            scanRoot,
			Reachable:       dirExists(scanRoot),
			RepositoryCount: registry.RepositoriesCount,
			LastScan:        meta.GeneratedAt,
			Enabled:         true,
			Source:          "_registry/repositories.json roots (last project scan)",
			Notes:           "Last scan output consumed by the dashboard. The list above is the next scan.",
		})
	}

	out.DataPaths = []DataPath{
		inspectPath("Data root", coreRoot, "process root"),
		inspectPath("Tasks", filepath.Join(coreRoot, "Work"), "Markdown task cards Work/<project>/tasks"),
		inspectPath("Workspaces", filepath.Join(coreRoot, "Work"), "PROJECT.md workspace cards"),
		inspectPath("Registry", filepath.Join(coreRoot, "_registry"), "generated discovery/runtime state"),
		inspectPath("Sessions", runtimedb.Path(absPath(runtimeRoot)), "runtime sqlite database (App's own ~/.fleet home, not Data)"),
		inspectPath("Config", filepath.Join(coreRoot, providerconfig.ConfigFileName), "task provider selection"),
		inspectPath("Local overlay", OverlayPath(coreRoot), "gitignored machine overlay; atomic Save target"),
		inspectPath("MCP", filepath.Join(coreRoot, ".mcp.json"), "Cursor project MCP servers"),
	}
	out.TaskBackend = inspectTaskBackend(coreRoot)
	return out
}

type registryFileMeta struct {
	GeneratedAt string   `json:"generated_at"`
	Root        string   `json:"root"`
	Roots       []string `json:"roots"`
}

func (meta registryFileMeta) scanRoots() []string {
	if len(meta.Roots) > 0 {
		return meta.Roots
	}
	if strings.TrimSpace(meta.Root) != "" {
		return []string{meta.Root}
	}
	return nil
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
		Implementations:  []string{"markdown"},
	}
	if err != nil {
		backend.Active = "error"
		return backend
	}
	if loaded.Type == "markdown" || loaded.Type == "" {
		backend.Active = "markdown"
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
