package projectscan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestNestedRepositoryMarkersDoNotLeakToParent(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "Audiophile", "jivemax")
	child := filepath.Join(parent, "jivelite")
	if err := os.MkdirAll(filepath.Join(parent, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(child, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "jivelite.sln"), []byte("Microsoft Visual Studio Solution File\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repositories, err := buildRepositories(root, filepath.Join(root, "_registry"), "")
	if err != nil {
		t.Fatal(err)
	}
	byRelative := map[string]Repository{}
	for _, repo := range repositories {
		byRelative[repo.RelativePath] = repo
	}
	parentRepo, ok := byRelative["Audiophile/jivemax"]
	if !ok {
		t.Fatalf("missing parent: %#v", repositories)
	}
	childRepo, ok := byRelative["Audiophile/jivemax/jivelite"]
	if !ok {
		t.Fatalf("missing child: %#v", repositories)
	}
	if !reflect.DeepEqual(parentRepo.Effective.Languages, []string{}) || !reflect.DeepEqual(parentRepo.Effective.Runtimes, []string{}) {
		t.Fatalf("parent effective = %#v", parentRepo.Effective)
	}
	if !reflect.DeepEqual(childRepo.Effective.Languages, []string{"csharp"}) || !reflect.DeepEqual(childRepo.Effective.Runtimes, []string{"dotnet"}) {
		t.Fatalf("child effective = %#v", childRepo.Effective)
	}
}

func TestEffectiveProfileCompositionAppliesOverrides(t *testing.T) {
	detected := emptyDetected()
	detected.Languages = []string{"javascript"}
	detected.Frameworks = []string{"react"}
	detected.Runtimes = []string{"node"}
	detected.Tooling = []string{"npm"}

	effective := composeEffective(detected, techOverride{
		Remove: []string{"javascript", "react", "node", "npm"},
		Add:    []string{"python"},
	}, techOverride{
		Add: []string{"docker"},
	})

	if !reflect.DeepEqual(effective.Languages, []string{"python"}) {
		t.Fatalf("languages = %#v", effective.Languages)
	}
	if len(effective.Frameworks) != 0 || len(effective.Runtimes) != 0 {
		t.Fatalf("effective = %#v", effective)
	}
	if !reflect.DeepEqual(effective.Tooling, []string{"docker"}) {
		t.Fatalf("tooling = %#v", effective.Tooling)
	}
}

func TestDetectionKeepsEvidenceForActiveMarkers(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte(`{"dependencies":{"svelte":"latest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	detected := detectTechnologies(repo, nil, nil)
	if !reflect.DeepEqual(detected.Languages, []string{"javascript"}) {
		t.Fatalf("languages = %#v", detected.Languages)
	}
	if !reflect.DeepEqual(detected.Frameworks, []string{"svelte"}) {
		t.Fatalf("frameworks = %#v", detected.Frameworks)
	}
	var paths []string
	for _, item := range detected.Evidence {
		paths = append(paths, item.Path)
	}
	if !reflect.DeepEqual(paths, []string{"package.json", "package.json"}) {
		t.Fatalf("evidence paths = %#v", paths)
	}
}

func TestWorkspaceGeneratorUsesEffectiveRepositoryView(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "_registry")
	if err := os.MkdirAll(registry, 0o755); err != nil {
		t.Fatal(err)
	}
	repos := `{
  "repositories": [
    {"id": "api", "name": "api", "relative_path": "Space/api", "stack": ["python"]},
    {"id": "web", "name": "web", "relative_path": "Space/web", "stack": ["javascript"]}
  ],
  "effective_repositories": [
    {"id": "api", "name": "api", "relative_path": "Space/api", "effective": {"languages": ["python"], "frameworks": [], "runtimes": [], "tooling": []}},
    {"id": "web", "name": "web", "relative_path": "Space/web", "effective": {"languages": ["typescript"], "frameworks": ["svelte"], "runtimes": ["node"], "tooling": []}}
  ]
}`
	groups := `{"groups": [{"kind": "workspace_group", "reason": "same top-level folder", "suggested_project_id": "space", "repository_ids": ["api", "web"]}]}`
	if err := os.WriteFile(filepath.Join(registry, "repositories.json"), []byte(repos), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(registry, "project-groups.json"), []byte(groups), 0o644); err != nil {
		t.Fatal(err)
	}

	projects, err := BuildWorkspaces(registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Fatalf("projects = %d", len(projects))
	}
	if !reflect.DeepEqual(projects[0].Stack, []string{"node", "python", "svelte", "typescript"}) {
		t.Fatalf("stack = %#v", projects[0].Stack)
	}
}

func TestScanKeepsRelativePathsAndGroupsPerRoot(t *testing.T) {
	base := t.TempDir()
	rootA := filepath.Join(base, "alpha")
	rootB := filepath.Join(base, "beta")
	for _, dir := range []string{
		filepath.Join(rootA, "Space", "api", ".git"),
		filepath.Join(rootA, "Space", "web", ".git"),
		filepath.Join(rootB, "Space", "api", ".git"),
		filepath.Join(rootB, "Space", "other", ".git"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	registry := filepath.Join(base, "_registry")
	result, err := Scan(ScanOptions{ScanRoots: []string{rootA, rootB}, RegistryDir: registry})
	if err != nil {
		t.Fatal(err)
	}
	if result.Repositories != 4 {
		t.Fatalf("repositories = %d", result.Repositories)
	}

	data, err := os.ReadFile(filepath.Join(registry, "repositories.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Roots        []string `json:"roots"`
		Repositories []struct {
			RelativePath string `json:"relative_path"`
			ScanRoot     string `json:"scan_root"`
		} `json:"repositories"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Roots) != 2 {
		t.Fatalf("roots = %#v", payload.Roots)
	}
	byRoot := map[string][]string{}
	for _, repo := range payload.Repositories {
		if strings.Contains(repo.RelativePath, "alpha") || strings.Contains(repo.RelativePath, "beta") {
			t.Fatalf("relative path prefixed with scan root: %s", repo.RelativePath)
		}
		byRoot[repo.ScanRoot] = append(byRoot[repo.ScanRoot], repo.RelativePath)
	}
	sets := []string{}
	for _, paths := range byRoot {
		sets = append(sets, strings.Join(sortedCopy(paths), ","))
	}
	sort.Strings(sets)
	if !reflect.DeepEqual(sets, []string{"Space/api,Space/other", "Space/api,Space/web"}) {
		t.Fatalf("paths = %#v", byRoot)
	}

	groupsData, err := os.ReadFile(filepath.Join(registry, "project-groups.json"))
	if err != nil {
		t.Fatal(err)
	}
	var groupsFile struct {
		Groups []Group `json:"groups"`
	}
	if err := json.Unmarshal(groupsData, &groupsFile); err != nil {
		t.Fatal(err)
	}
	var workspaceIDs []string
	for _, group := range groupsFile.Groups {
		if group.Kind == "workspace_group" {
			workspaceIDs = append(workspaceIDs, group.SuggestedProjectID)
		}
	}
	sort.Strings(workspaceIDs)
	if !reflect.DeepEqual(workspaceIDs, []string{"alpha-space", "beta-space"}) {
		t.Fatalf("workspace groups = %#v", workspaceIDs)
	}
}

func sortedCopy(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}
