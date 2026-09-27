package projectscan

import (
	"path/filepath"
	"sort"
	"strings"
)

func buildRepositories(root, registryDir, workDir string) ([]Repository, error) {
	seenAt := utcNow()
	previous := loadPreviousSeen(registryDir)
	overrides := loadOverrides(registryDir)
	cards := parseProjectCardOverrides(workDir)
	repoPaths, err := discoverGitRepos(root)
	if err != nil {
		return nil, err
	}

	ids := map[string]string{}
	remotes := map[string]string{}
	for _, path := range repoPaths {
		remote := runGit(path, "remote", "get-url", "origin")
		remotes[path] = remote
		ids[path] = repoID(path, remote)
	}
	parents := parentMap(repoPaths)
	nestedByParent := map[string][]string{}
	for _, id := range ids {
		nestedByParent[id] = []string{}
	}
	for path, parent := range parents {
		if parent != "" {
			nestedByParent[ids[parent]] = append(nestedByParent[ids[parent]], ids[path])
		}
	}

	repositories := make([]Repository, 0, len(repoPaths))
	for _, path := range repoPaths {
		remote := remotes[path]
		id := ids[path]
		title, summary, source, readme := summarizeReadme(path)
		relative := filepath.ToSlash(mustRel(root, path))
		detected := detectTechnologies(path, nestedRepoRoots(path, repoPaths), overrides.IgnorePaths)
		parent := parents[path]
		parentID := ""
		if parent != "" {
			parentID = ids[parent]
		}
		firstSeen := seenAt
		if previousAt := previous[id]; previousAt != "" {
			firstSeen = previousAt
		} else if previousAt := previous[previousRepoID(path, remote)]; previousAt != "" {
			firstSeen = previousAt
		}
		nested := append([]string{}, nestedByParent[id]...)
		sort.Strings(nested)
		repo := Repository{
			ID:                  id,
			RemoteIdentity:      remoteIdentity(remote),
			Name:                filepath.Base(path),
			Path:                path,
			RelativePath:        relative,
			Remote:              remote,
			Branch:              runGit(path, "branch", "--show-current"),
			Head:                runGit(path, "rev-parse", "--short", "HEAD"),
			ReadmePath:          readme,
			Title:               title,
			Summary:             summary,
			SummarySource:       source,
			Detected:            detected,
			ParentRepositoryID:  parentID,
			NestedRepositoryIDs: nested,
			FirstSeenAt:         firstSeen,
			LastSeenAt:          seenAt,
		}
		repo.Effective = composeEffective(detected, repositoryOverride(repo, relative, overrides), projectOverrideFor(relative, cards, overrides))
		repo.Stack = stackFrom(repo.Effective.TechLists)
		repo.Markers = markersFrom(detected)
		repositories = append(repositories, repo)
	}
	return repositories, nil
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func scanRootsOf(repositories []Repository) []string {
	seen := map[string]struct{}{}
	var roots []string
	for _, repo := range repositories {
		if repo.ScanRoot == "" {
			continue
		}
		if _, ok := seen[repo.ScanRoot]; ok {
			continue
		}
		seen[repo.ScanRoot] = struct{}{}
		roots = append(roots, repo.ScanRoot)
	}
	return roots
}

func suggestGroups(repositories []Repository) []Group {
	var groups []Group
	byTop := map[string][]Repository{}
	for _, repo := range repositories {
		first := strings.Split(repo.RelativePath, "/")[0]
		key := repo.ScanRoot + "\n" + first
		byTop[key] = append(byTop[key], repo)
	}
	multi := len(scanRootsOf(repositories)) > 1
	folders := keysOfStringMap(byTop)
	for _, key := range folders {
		items := byTop[key]
		if len(items) < 2 {
			continue
		}
		folder := strings.Split(items[0].RelativePath, "/")[0]
		projectID := slugify(folder, "repo")
		if multi {
			projectID = slugify(filepath.Base(items[0].ScanRoot)+"-"+folder, "repo")
		}
		groups = append(groups, Group{
			ID:                 "parent-" + projectID,
			Kind:               "workspace_group",
			Confidence:         "high",
			Decision:           "treat_as_group",
			Reason:             "same top-level folder",
			SuggestedProjectID: projectID,
			RepositoryIDs:      sortedIDs(items),
		})
	}

	byRemote := map[string][]Repository{}
	for _, repo := range repositories {
		if repo.RemoteIdentity != "" {
			byRemote[repo.RemoteIdentity] = append(byRemote[repo.RemoteIdentity], repo)
		}
	}
	for _, remoteID := range keysOfStringMap(byRemote) {
		items := byRemote[remoteID]
		if len(items) < 2 {
			continue
		}
		groups = append(groups, Group{
			ID:                 "remote-" + remoteID,
			Kind:               "duplicate_candidate",
			Confidence:         "high",
			Decision:           "review",
			Reason:             "same git remote",
			SuggestedProjectID: slugify(items[0].Name, "repo"),
			RepositoryIDs:      sortedIDs(items),
		})
	}

	byName := map[string][]Repository{}
	for _, repo := range repositories {
		key := comparableName(repo.Name)
		if len(key) >= 4 {
			byName[key] = append(byName[key], repo)
		}
	}
	for _, key := range keysOfStringMap(byName) {
		items := byName[key]
		if len(items) < 2 {
			continue
		}
		groups = append(groups, Group{
			ID:                 "name-" + slugify(key, "repo"),
			Kind:               "related_candidate",
			Confidence:         "medium",
			Decision:           "review",
			Reason:             "same base repository name",
			SuggestedProjectID: slugify(key, "repo"),
			RepositoryIDs:      sortedIDs(items),
		})
	}

	for _, repo := range repositories {
		if len(repo.NestedRepositoryIDs) == 0 {
			continue
		}
		ids := append([]string{repo.ID}, repo.NestedRepositoryIDs...)
		sort.Strings(ids)
		groups = append(groups, Group{
			ID:                 "nested-" + repo.ID,
			Kind:               "nested_group",
			Confidence:         "medium",
			Decision:           "review",
			Reason:             "nested repositories",
			SuggestedProjectID: slugify(repo.Name, "repo"),
			RepositoryIDs:      ids,
		})
	}

	unique := map[string]Group{}
	for _, group := range groups {
		if len(group.RepositoryIDs) < 2 {
			continue
		}
		unique[group.Reason+"\n"+strings.Join(group.RepositoryIDs, "\n")] = group
	}
	out := make([]Group, 0, len(unique))
	for _, group := range unique {
		out = append(out, group)
	}
	sort.Slice(out, func(i, j int) bool {
		if pi, pj := kindPriority(out[i].Kind), kindPriority(out[j].Kind); pi != pj {
			return pi < pj
		}
		return out[i].SuggestedProjectID < out[j].SuggestedProjectID
	})
	return out
}

func comparableName(name string) string {
	value := strings.TrimSuffix(strings.ToLower(name), ".git")
	parts := nonAlnumLower.Split(value, -1)
	suffixes := map[string]struct{}{
		"backup": {}, "copy": {}, "demo": {}, "dev": {}, "frontend": {}, "old": {},
		"playground": {}, "site": {}, "sonar": {}, "test": {}, "tests": {},
	}
	var kept []string
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	for len(kept) > 1 {
		if _, ok := suffixes[kept[len(kept)-1]]; !ok {
			break
		}
		kept = kept[:len(kept)-1]
	}
	return strings.Join(kept, "-")
}

func sortedIDs(items []Repository) []string {
	ids := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		ids = append(ids, item.ID)
	}
	sort.Strings(ids)
	return ids
}

func kindPriority(kind string) int {
	switch kind {
	case "workspace_group":
		return 0
	case "duplicate_candidate":
		return 1
	case "nested_group":
		return 2
	case "related_candidate":
		return 3
	default:
		return 9
	}
}

func keysOfStringMap[T any](values map[string]T) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
