package projectscan

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ignoreRule struct {
	PathPrefix string `json:"path_prefix"`
	Reason     string `json:"reason"`
}

func detectTechnologies(repo string, children map[string]struct{}, ignoreRules []ignoreRule) Detected {
	if children == nil {
		children = map[string]struct{}{}
	}
	var evidence []Evidence
	collectMarkerEvidence(&evidence, repo, children, ignoreRules)
	collectPackageManifestEvidence(&evidence, repo, ignoreRules)
	if isUnityProject(repo) {
		addEvidence(&evidence, "layout", ".", []string{"unity"}, "Assets and ProjectSettings directories", ignoreRules)
	}

	active := map[string]struct{}{}
	for _, item := range evidence {
		if item.Ignored {
			continue
		}
		for _, tech := range item.Technologies {
			active[tech] = struct{}{}
		}
	}
	names := make([]string, 0, len(active))
	for tech := range active {
		names = append(names, tech)
	}
	detected := Detected{TechLists: categorize(names), Evidence: evidence}
	if detected.Evidence == nil {
		detected.Evidence = []Evidence{}
	}
	sort.SliceStable(detected.Evidence, func(i, j int) bool {
		return detected.Evidence[i].Path < detected.Evidence[j].Path
	})
	return detected
}

// collectMarkerEvidence walks repo (not into nested repositories, pruned
// directories or anything deeper than three levels) for files that name a stack.
func collectMarkerEvidence(evidence *[]Evidence, repo string, children map[string]struct{}, ignoreRules []ignoreRule) {
	_ = filepath.WalkDir(repo, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // an unreadable entry is skipped; the scan goes on
		}
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return nil //nolint:nilerr // an unreadable entry is skipped; the scan goes on
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if path != repo && skipMarkerDir(path, entry.Name(), rel, children) {
				return fs.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if techs, ok := stackMarkers[name]; ok {
			addEvidence(evidence, "marker", rel, techs, name+" marker", ignoreRules)
			return nil
		}
		if strings.HasSuffix(name, ".csproj") || strings.HasSuffix(name, ".sln") {
			second := "dotnet"
			if isUnityProject(repo) {
				second = "unity"
			}
			addEvidence(evidence, "marker", rel, []string{"csharp", second}, name+" marker", ignoreRules)
		}
		return nil
	})
}

func skipMarkerDir(path string, name string, rel string, children map[string]struct{}) bool {
	if _, pruned := pruneDirs[name]; pruned || name == ".git" {
		return true
	}
	if _, child := children[path]; child {
		return true
	}
	return strings.Count(rel, "/")+1 > 3
}

func collectPackageManifestEvidence(evidence *[]Evidence, repo string, ignoreRules []ignoreRule) {
	packageJSON := filepath.Join(repo, "package.json")
	info, err := os.Stat(packageJSON)
	if err != nil || !info.Mode().IsRegular() {
		return
	}
	lower := strings.ToLower(readText(packageJSON, 12000))
	var techs []string
	for _, name := range []string{"svelte", "react", "vite", "tauri", "typescript"} {
		if strings.Contains(lower, name) {
			techs = append(techs, name)
		}
	}
	addEvidence(evidence, "package-manifest", "package.json", techs, "package.json dependency/name scan", ignoreRules)
}

func addEvidence(evidence *[]Evidence, kind, markerPath string, technologies []string, detail string, ignoreRules []ignoreRule) {
	techs := normalizeTechnologies(technologies)
	if len(techs) == 0 {
		return
	}
	reason := markerIgnoreReason(markerPath, ignoreRules)
	*evidence = append(*evidence, Evidence{
		Kind:         kind,
		Path:         markerPath,
		Technologies: techs,
		Detail:       detail,
		Ignored:      reason != "",
		IgnoreReason: strPtr(reason),
	})
}

func markerIgnoreReason(markerPath string, ignoreRules []ignoreRule) string {
	marker := filepath.ToSlash(markerPath)
	parts := strings.Split(marker, "/")
	if len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}
	var noisy []string
	for _, part := range parts {
		if _, ok := pruneDirs[part]; ok {
			noisy = append(noisy, part)
		}
	}
	if len(noisy) > 0 {
		sort.Strings(noisy)
		return "ignored source directory: " + noisy[0]
	}
	for _, rule := range ignoreRules {
		prefix := strings.Trim(strings.TrimSpace(rule.PathPrefix), "/")
		if prefix != "" && (marker == prefix || strings.HasPrefix(marker, prefix+"/")) {
			if rule.Reason != "" {
				return rule.Reason
			}
			return "ignored by rule " + prefix
		}
	}
	return ""
}

func isUnityProject(repo string) bool {
	assets, errA := os.Stat(filepath.Join(repo, "Assets"))
	settings, errS := os.Stat(filepath.Join(repo, "ProjectSettings"))
	return errA == nil && assets.IsDir() && errS == nil && settings.IsDir()
}

func normalizeTechnologies(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		tech := strings.ToLower(strings.TrimSpace(value))
		if tech == "" {
			continue
		}
		if _, ok := technologyCatalog[tech]; !ok {
			continue
		}
		if _, ok := seen[tech]; ok {
			continue
		}
		seen[tech] = struct{}{}
		out = append(out, tech)
	}
	sort.Strings(out)
	return out
}

func categorize(technologies []string) TechLists {
	profile := emptyTech()
	grouped := map[string]map[string]struct{}{
		"languages": {}, "frameworks": {}, "runtimes": {}, "tooling": {},
	}
	buckets := map[string]string{
		"language": "languages", "framework": "frameworks", "runtime": "runtimes", "tooling": "tooling",
	}
	for _, tech := range normalizeTechnologies(technologies) {
		field := buckets[technologyCatalog[tech].category]
		if field == "" {
			continue
		}
		grouped[field][tech] = struct{}{}
	}
	profile.Languages = keysOf(grouped["languages"])
	profile.Frameworks = keysOf(grouped["frameworks"])
	profile.Runtimes = keysOf(grouped["runtimes"])
	profile.Tooling = keysOf(grouped["tooling"])
	return profile
}

func keysOf(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
