// Package providerconfig loads Core's task-provider selection: which
// taskprovider.Provider implementation the daemon should construct, and the
// settings that implementation needs.
//
// Config lives in an optional root-level core.config.yaml:
//
//	taskProvider:
//	  type: markdown
//
// or:
//
//	taskProvider:
//	  type: plane
//	  workspace: eggs_gd
//	  baseUrl: https://api.plane.so
//	  tokenEnv: PLANE_API_TOKEN
//	  coreProject: core-eggs-gd
//	statusMap:
//	  backlog: Backlog
//	  todo: Todo
//	  doing: In Progress
//	  blocked: Blocked
//	  needs_review: In Review
//	  needs_rework: Todo
//	  done: Done
//	  archived: Cancelled
//
// taskProvider.project (a raw Plane project UUID) may also be set, but is
// optional: when absent, the Plane adapter resolves it at first use by
// matching coreProject against the workspace's project list (identifier or
// name, case-insensitive). Set project explicitly only when the Plane
// project's name doesn't match coreProject.
//
// When the file is absent, Load returns the markdown default unchanged —
// existing installs need no config file to keep working exactly as before
// this package existed.
package providerconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
)

// ConfigFileName is the root-relative config file Load reads.
const ConfigFileName = "core.config.yaml"

// Settings is Core's resolved task-provider configuration.
type Settings struct {
	// Type selects the taskprovider.Provider implementation: "markdown"
	// (default) or "plane".
	Type string
	// Plane carries settings for the Plane adapter. Populated (and
	// validated) only when Type == "plane".
	Plane PlaneSettings
}

// PlaneSettings configures the Plane adapter: which workspace/project to
// talk to, where to find the API token, and how Core statuses map onto that
// project's configured workflow states.
type PlaneSettings struct {
	// Workspace is the Plane workspace slug, e.g. "eggs_gd".
	Workspace string
	// BaseURL is the Plane API base, e.g. "https://api.plane.so". Defaults
	// to Plane Cloud when unset.
	BaseURL string
	// TokenEnv names the environment variable holding the Plane API token
	// (never the token value itself — Core does not persist secrets in
	// files it can commit).
	TokenEnv string
	// ProjectID is the Plane project UUID that holds Core-managed work
	// items. Optional: when unset, the Plane adapter resolves it at first
	// use by listing the workspace's projects and matching one whose
	// identifier or name equals CoreProject (case-insensitive) — see
	// plane.Provider.ensureProjectID. Set this only to pin an explicit UUID
	// when the Plane project's name doesn't match CoreProject.
	ProjectID string
	// CoreProject is the Core project id (Work/<project-id>) whose tasks
	// live in this Plane project — also the default lookup key used to find
	// the matching Plane project when ProjectID is unset. The first Plane
	// adapter supports mapping exactly one Core project to one Plane
	// project; see _docs/PLANE_MULTI_PROJECT_MAPPING.md for the N:N design.
	CoreProject string
	// Repository is the single repository (relative path, matching
	// _registry/repositories.json shape) that Plane-backed tasks under
	// CoreProject launch against. Optional: launch validation simply fails
	// its repository gate (as it already does for a Markdown task with no
	// repositories) when unset.
	Repository string
	// StatusMap overrides the default identity mapping from Core status
	// (e.g. "needs_review") to a Plane workflow state *name* configured in
	// the target project. Keys are Core status strings; values are Plane
	// state names, matched case-insensitively. A Core status absent from
	// this map falls back to a state literally named the same as the
	// status.
	StatusMap map[string]string
}

// Default returns the zero-config, backward-compatible setting: the
// Markdown/file provider.
func Default() Settings {
	return Settings{Type: "markdown"}
}

// Load reads <root>/core.config.yaml, if present, and returns the resolved
// Settings. A missing file is not an error — it returns Default().
func Load(root string) (Settings, error) {
	path := filepath.Join(root, ConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Settings{}, err
	}

	fm, _ := mdfile.ParseFrontmatter("---\n" + string(data) + "\n---\n")

	settings := Default()
	settings.Type = strings.ToLower(strings.TrimSpace(mdfile.Scalar(fm, "taskProvider.type", settings.Type)))
	if settings.Type != "markdown" && settings.Type != "plane" {
		return Settings{}, fmt.Errorf("%s: unknown taskProvider.type %q", ConfigFileName, settings.Type)
	}

	settings.Plane = PlaneSettings{
		Workspace:   mdfile.Scalar(fm, "taskProvider.workspace", ""),
		BaseURL:     mdfile.Scalar(fm, "taskProvider.baseUrl", "https://api.plane.so"),
		TokenEnv:    mdfile.Scalar(fm, "taskProvider.tokenEnv", "PLANE_API_TOKEN"),
		ProjectID:   mdfile.Scalar(fm, "taskProvider.project", ""),
		CoreProject: mdfile.Scalar(fm, "taskProvider.coreProject", ""),
		Repository:  mdfile.Scalar(fm, "taskProvider.repository", ""),
		StatusMap:   statusMapFrom(fm),
	}

	if settings.Type != "plane" {
		return settings, nil
	}
	if settings.Plane.Workspace == "" {
		return Settings{}, fmt.Errorf("%s: taskProvider.workspace is required when taskProvider.type is plane", ConfigFileName)
	}
	if settings.Plane.CoreProject == "" {
		return Settings{}, fmt.Errorf("%s: taskProvider.coreProject is required when taskProvider.type is plane", ConfigFileName)
	}
	return settings, nil
}

func statusMapFrom(fm mdfile.Frontmatter) map[string]string {
	out := map[string]string{}
	for key, value := range fm {
		rest, ok := strings.CutPrefix(key, "statusMap.")
		if !ok {
			continue
		}
		str, ok := value.(string)
		if !ok || str == "" {
			continue
		}
		out[rest] = str
	}
	return out
}
