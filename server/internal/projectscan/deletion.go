package projectscan

import (
	"regexp"
	"sort"
	"strings"
)

func suggestDeletionCandidates(repositories []Repository, rules protectionRules) []DeletionCandidate {
	candidates := map[string]*DeletionCandidate{}
	add := func(repo Repository, reason, detail string) {
		if protected, _ := protectedRepository(repo.RelativePath, rules); protected {
			return
		}
		entry, ok := candidates[repo.ID]
		if !ok {
			status := runGit(repo.Path, "status", "--short")
			gitStatus := "clean"
			if status != "" {
				gitStatus = "dirty"
			}
			entry = &DeletionCandidate{
				RepositoryID: repo.ID,
				Name:         repo.Name,
				Path:         repo.Path,
				RelativePath: repo.RelativePath,
				Remote:       strPtr(repo.Remote),
				Branch:       strPtr(repo.Branch),
				Head:         strPtr(repo.Head),
				Decision:     "review",
				Reasons:      []string{},
				Details:      []string{},
				GitStatus:    gitStatus,
			}
			candidates[repo.ID] = entry
		}
		if !contains(entry.Reasons, reason) {
			entry.Reasons = append(entry.Reasons, reason)
		}
		if !contains(entry.Details, detail) {
			entry.Details = append(entry.Details, detail)
		}
	}

	byRemote := map[string][]Repository{}
	for _, repo := range repositories {
		if repo.RemoteIdentity != "" {
			byRemote[repo.RemoteIdentity] = append(byRemote[repo.RemoteIdentity], repo)
		}
	}
	for _, items := range byRemote {
		if len(items) < 2 {
			continue
		}
		paths := make([]string, len(items))
		for i, item := range items {
			paths[i] = item.RelativePath
		}
		if protectedRemoteGroup(paths, rules) {
			continue
		}
		sortedItems := append([]Repository{}, items...)
		sort.Slice(sortedItems, func(i, j int) bool {
			di := strings.Count(sortedItems[i].RelativePath, "/") + 1
			dj := strings.Count(sortedItems[j].RelativePath, "/") + 1
			if di != dj {
				return di < dj
			}
			return sortedItems[i].RelativePath < sortedItems[j].RelativePath
		})
		keeper := sortedItems[0]
		for _, repo := range sortedItems[1:] {
			add(repo, "duplicate local checkout", "Shares remote with "+keeper.RelativePath+"; review whether this checkout is still needed.")
		}
	}

	byID := map[string]Repository{}
	for _, repo := range repositories {
		byID[repo.ID] = repo
	}
	for _, repo := range repositories {
		if repo.ParentRepositoryID != "" {
			parentPath := repo.ParentRepositoryID
			if parent, ok := byID[repo.ParentRepositoryID]; ok {
				parentPath = parent.RelativePath
			}
			add(repo, "nested repository", "Repository is nested inside "+parentPath+"; review whether it is vendored/submodule/intentional.")
		}
		var hints []string
		for _, part := range regexp.MustCompile(`[^a-z0-9]+`).Split(strings.ToLower(repo.Name), -1) {
			if _, ok := deletionNameHints[part]; ok {
				hints = append(hints, part)
			}
		}
		sort.Strings(hints)
		if len(hints) > 0 {
			add(repo, "temporary name hint", "Name contains cleanup hint(s): "+strings.Join(hints, ", ")+".")
		}
		if repo.Remote == "" {
			add(repo, "no git remote", "Repository has no origin remote.")
		}
		if repo.ReadmePath == "" {
			add(repo, "no README", "Repository has no README detected at its root.")
		}
	}

	out := make([]DeletionCandidate, 0, len(candidates))
	for _, entry := range candidates {
		entry.Confidence = deletionConfidence(entry.Reasons)
		out = append(out, *entry)
	}
	rank := map[string]int{"high": 0, "medium": 1, "low": 2}
	sort.Slice(out, func(i, j int) bool {
		if rank[out[i].Confidence] != rank[out[j].Confidence] {
			return rank[out[i].Confidence] < rank[out[j].Confidence]
		}
		return out[i].RelativePath < out[j].RelativePath
	})
	return out
}

func deletionConfidence(reasons []string) string {
	strong := map[string]struct{}{"duplicate local checkout": {}, "nested repository": {}}
	medium := map[string]struct{}{"temporary name hint": {}, "no git remote": {}}
	for _, reason := range reasons {
		if _, ok := strong[reason]; ok {
			return "high"
		}
	}
	mediumCount := 0
	for _, reason := range reasons {
		if _, ok := medium[reason]; ok {
			mediumCount++
		}
	}
	if mediumCount > 0 && len(reasons) >= 2 {
		return "medium"
	}
	return "low"
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
