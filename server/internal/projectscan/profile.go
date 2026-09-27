package projectscan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type techOverride struct {
	Add       []string `json:"add"`
	Remove    []string `json:"remove"`
	ProjectID string   `json:"-"`
}

type overrideFile struct {
	Repositories map[string]techOverride `json:"repositories"`
	Projects     map[string]techOverride `json:"projects"`
	IgnorePaths  []ignoreRule            `json:"ignore_paths"`
}

type protectionRules struct {
	ProtectedPrefixes        []prefixRule `json:"protected_prefixes"`
	ProtectedRepositoryPaths []pathRule   `json:"protected_repository_paths"`
	ProtectedRemoteGroups    []remoteRule `json:"protected_remote_groups"`
}

type prefixRule struct {
	Prefix string `json:"prefix"`
	Reason string `json:"reason"`
}

type pathRule struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type remoteRule struct {
	Paths  []string `json:"paths"`
	Reason string   `json:"reason"`
}

func loadProtectionRules(registryDir string) protectionRules {
	var rules protectionRules
	if err := readJSON(filepath.Join(registryDir, "protection-rules.json"), &rules); err != nil {
		return protectionRules{}
	}
	return rules
}

func loadOverrides(registryDir string) overrideFile {
	var file overrideFile
	if err := readJSON(filepath.Join(registryDir, "technology-overrides.json"), &file); err != nil {
		return overrideFile{}
	}
	return file
}

func loadPreviousSeen(registryDir string) map[string]string {
	var payload struct {
		Repositories []struct {
			ID          string `json:"id"`
			FirstSeenAt string `json:"first_seen_at"`
		} `json:"repositories"`
	}
	seen := map[string]string{}
	if err := readJSON(filepath.Join(registryDir, "repositories.json"), &payload); err != nil {
		return seen
	}
	for _, item := range payload.Repositories {
		if item.ID != "" {
			seen[item.ID] = item.FirstSeenAt
		}
	}
	return seen
}

func readJSON(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func composeEffective(detected Detected, repoOverride, projectOverride techOverride) Effective {
	techs := map[string]struct{}{}
	for _, name := range append(append(append(append([]string{}, detected.Languages...), detected.Frameworks...), detected.Runtimes...), detected.Tooling...) {
		techs[name] = struct{}{}
	}
	for _, override := range []techOverride{projectOverride, repoOverride} {
		for _, name := range normalizeTechnologies(override.Remove) {
			delete(techs, name)
		}
		for _, name := range normalizeTechnologies(override.Add) {
			techs[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(techs))
	for name := range techs {
		names = append(names, name)
	}
	return Effective{TechLists: categorize(names)}
}

func repositoryOverride(repo Repository, relativePath string, file overrideFile) techOverride {
	candidates := []string{relativePath, repo.ID, repo.RemoteIdentity}
	for _, key := range candidates {
		if key == "" {
			continue
		}
		if override, ok := file.Repositories[key]; ok {
			return override
		}
	}
	return techOverride{}
}

func projectOverrideFor(relativePath string, cards map[string]techOverride, file overrideFile) techOverride {
	card := cards[relativePath]
	var registry techOverride
	if card.ProjectID != "" {
		registry = file.Projects[card.ProjectID]
	}
	merged := techOverride{
		Add:    append(append([]string{}, registry.Add...), card.Add...),
		Remove: append(append([]string{}, registry.Remove...), card.Remove...),
	}
	if len(merged.Add) == 0 && len(merged.Remove) == 0 {
		return techOverride{}
	}
	return merged
}

func protectedRepository(relativePath string, rules protectionRules) (bool, string) {
	for _, rule := range rules.ProtectedPrefixes {
		prefix := strings.TrimRight(rule.Prefix, "/")
		if relativePath == prefix || strings.HasPrefix(relativePath, prefix+"/") {
			return true, rule.Reason
		}
	}
	for _, rule := range rules.ProtectedRepositoryPaths {
		if relativePath == rule.Path {
			return true, rule.Reason
		}
	}
	return false, ""
}

func protectedRemoteGroup(paths []string, rules protectionRules) bool {
	sorted := append([]string{}, paths...)
	sort.Strings(sorted)
	for _, rule := range rules.ProtectedRemoteGroups {
		rulePaths := append([]string{}, rule.Paths...)
		sort.Strings(rulePaths)
		if strings.Join(sorted, "\n") == strings.Join(rulePaths, "\n") {
			return true
		}
	}
	return false
}

var overrideKey = regexp.MustCompile(`^[-*]?\s*([a-z_]+):\s*(.*)$`)

func parseProjectCardOverrides(workDir string) map[string]techOverride {
	overrides := map[string]techOverride{}
	if workDir == "" {
		return overrides
	}
	matches, err := filepath.Glob(filepath.Join(workDir, "*", "PROJECT.md"))
	if err != nil {
		return overrides
	}
	sort.Strings(matches)
	for _, card := range matches {
		text := readText(card, 20000)
		if !strings.HasPrefix(text, "---\n") {
			continue
		}
		end := strings.Index(text[4:], "\n---")
		if end < 0 {
			continue
		}
		frontmatter := text[4 : 4+end]
		projectID := ""
		var repositories []string
		inRepositories := false
		for _, line := range strings.Split(frontmatter, "\n") {
			stripped := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(stripped, "id:"):
				projectID = strings.Trim(strings.TrimSpace(strings.TrimPrefix(stripped, "id:")), `"`)
			case stripped == "repositories:":
				inRepositories = true
			case inRepositories && strings.HasPrefix(stripped, "-"):
				repositories = append(repositories, strings.Trim(strings.TrimSpace(strings.TrimPrefix(stripped, "-")), `"`))
			case stripped != "" && !strings.HasPrefix(line, " "):
				inRepositories = false
			}
		}
		marker := "\n## Technology Overrides\n"
		if !strings.Contains(text, marker) {
			continue
		}
		section := strings.SplitN(strings.SplitN(text, marker, 2)[1], "\n## ", 2)[0]
		override := parseOverrideSection(section)
		if len(override.Add) == 0 && len(override.Remove) == 0 {
			continue
		}
		if projectID == "" {
			projectID = filepath.Base(filepath.Dir(card))
		}
		override.ProjectID = projectID
		for _, relativePath := range repositories {
			overrides[relativePath] = override
		}
	}
	return overrides
}

func parseOverrideSection(section string) techOverride {
	aliases := map[string]string{
		"add_languages": "add", "add_frameworks": "add", "add_runtimes": "add", "add_tooling": "add",
		"remove_languages": "remove", "remove_frameworks": "remove", "remove_runtimes": "remove", "remove_tooling": "remove",
	}
	lists := map[string][]string{}
	current := ""
	for _, line := range strings.Split(section, "\n") {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			continue
		}
		if match := overrideKey.FindStringSubmatch(stripped); match != nil {
			key, value := match[1], match[2]
			current = ""
			if alias, ok := aliases[key]; ok {
				current = key
				if value != "" {
					lists[alias] = append(lists[alias], parseInlineValues(value)...)
				}
			}
			continue
		}
		if current != "" && strings.HasPrefix(stripped, "-") {
			lists[aliases[current]] = append(lists[aliases[current]], parseInlineValues(strings.TrimPrefix(stripped, "-"))...)
		}
	}
	override := techOverride{
		Add:    normalizeTechnologies(lists["add"]),
		Remove: normalizeTechnologies(lists["remove"]),
	}
	return override
}

func parseInlineValues(value string) []string {
	value = strings.Trim(strings.TrimSpace(value), "[]")
	var out []string
	for _, item := range regexp.MustCompile(`[, ]+`).Split(value, -1) {
		item = strings.Trim(strings.TrimSpace(item), `"'`)
		item = strings.Trim(item, ",")
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func stackFrom(profile TechLists) []string {
	return sortedUnique(append(append(append(append([]string{}, profile.Languages...), profile.Frameworks...), profile.Runtimes...), profile.Tooling...))
}

func markersFrom(detected Detected) []string {
	var markers []string
	for _, item := range detected.Evidence {
		if !item.Ignored {
			markers = append(markers, item.Path)
		}
	}
	return sortedUnique(markers)
}
