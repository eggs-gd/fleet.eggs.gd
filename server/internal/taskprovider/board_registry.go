package taskprovider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
)

func loadRegistry(root string) (board.RegistryInfo, error) {
	registryDir := filepath.Join(root, "_registry")
	repositories, err := loadRepositoryTechnologies(filepath.Join(registryDir, "repositories.json"))
	if err != nil {
		return board.RegistryInfo{}, err
	}
	return board.RegistryInfo{
		RepositoriesCount: countJSONItems(filepath.Join(registryDir, "repositories.json"), "repositories"),
		GroupsCount:       countJSONItems(filepath.Join(registryDir, "project-groups.json"), "groups"),
		Repositories:      repositories,
	}, nil
}

type registryRepositoriesPayload struct {
	Repositories          []registryRepositoryJSON `json:"repositories"`
	EffectiveRepositories []registryRepositoryJSON `json:"effective_repositories"`
}

type registryRepositoryJSON struct {
	ID                  string                     `json:"id"`
	Name                string                     `json:"name"`
	RelativePath        string                     `json:"relative_path"`
	Remote              string                     `json:"remote"`
	Branch              string                     `json:"branch"`
	ParentRepositoryID  string                     `json:"parent_repository_id"`
	NestedRepositoryIDs []string                   `json:"nested_repository_ids"`
	Stack               []string                   `json:"stack"`
	Markers             []string                   `json:"markers"`
	Effective           board.TechnologyProfile    `json:"effective"`
	Detected            registryDetectedProfile    `json:"detected"`
	Evidence            []board.TechnologyEvidence `json:"evidence"`
}

type registryDetectedProfile struct {
	board.TechnologyProfile
	Evidence []board.TechnologyEvidence `json:"evidence"`
}

func loadRepositoryTechnologies(path string) ([]board.RepositoryTechnology, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []board.RepositoryTechnology{}, nil
		}
		return nil, err
	}

	var payload registryRepositoriesPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	source := mergeEffectiveRepositoryViews(payload.Repositories, payload.EffectiveRepositories)

	repositories := make([]board.RepositoryTechnology, 0, len(source))
	for _, repo := range source {
		detected := repo.Detected.TechnologyProfile
		evidence := repo.Detected.Evidence
		if len(evidence) == 0 {
			evidence = repo.Evidence
		}
		if profileIsEmpty(detected) && len(repo.Stack) > 0 {
			detected.Tooling = append([]string{}, repo.Stack...)
		}
		effective := repo.Effective
		if profileIsEmpty(effective) && len(repo.Stack) > 0 {
			effective.Tooling = append([]string{}, repo.Stack...)
		}
		repositories = append(repositories, board.RepositoryTechnology{
			ID:                  repo.ID,
			Name:                repo.Name,
			RelativePath:        repo.RelativePath,
			Remote:              repo.Remote,
			Branch:              repo.Branch,
			ParentRepositoryID:  repo.ParentRepositoryID,
			NestedRepositoryIDs: append([]string{}, repo.NestedRepositoryIDs...),
			Effective:           effective,
			Detected:            detected,
			EffectiveTags:       technologyProfileTags(effective),
			DetectedTags:        technologyProfileTags(detected),
			Evidence:            evidence,
		})
	}
	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].RelativePath < repositories[j].RelativePath
	})
	return repositories, nil
}

func mergeEffectiveRepositoryViews(raw []registryRepositoryJSON, effective []registryRepositoryJSON) []registryRepositoryJSON {
	if len(raw) == 0 {
		return effective
	}
	if len(effective) == 0 {
		return raw
	}

	effectiveByID := map[string]registryRepositoryJSON{}
	effectiveByPath := map[string]registryRepositoryJSON{}
	for _, repo := range effective {
		if repo.ID != "" {
			effectiveByID[repo.ID] = repo
		}
		if repo.RelativePath != "" {
			effectiveByPath[repo.RelativePath] = repo
		}
	}

	merged := make([]registryRepositoryJSON, 0, len(raw))
	for _, repo := range raw {
		effectiveRepo, ok := effectiveByID[repo.ID]
		if !ok {
			effectiveRepo, ok = effectiveByPath[repo.RelativePath]
		}
		if ok {
			repo.Effective = effectiveRepo.Effective
			if repo.ID == "" {
				repo.ID = effectiveRepo.ID
			}
			if repo.Name == "" {
				repo.Name = effectiveRepo.Name
			}
			if repo.RelativePath == "" {
				repo.RelativePath = effectiveRepo.RelativePath
			}
			if repo.Remote == "" {
				repo.Remote = effectiveRepo.Remote
			}
			if repo.Branch == "" {
				repo.Branch = effectiveRepo.Branch
			}
			if repo.ParentRepositoryID == "" {
				repo.ParentRepositoryID = effectiveRepo.ParentRepositoryID
			}
			if len(repo.NestedRepositoryIDs) == 0 {
				repo.NestedRepositoryIDs = effectiveRepo.NestedRepositoryIDs
			}
			if len(repo.Evidence) == 0 {
				repo.Evidence = effectiveRepo.Evidence
			}
		}
		merged = append(merged, repo)
	}
	return merged
}

func profileIsEmpty(profile board.TechnologyProfile) bool {
	return len(profile.Languages) == 0 &&
		len(profile.Frameworks) == 0 &&
		len(profile.Runtimes) == 0 &&
		len(profile.Tooling) == 0
}

func technologyProfileTags(profile board.TechnologyProfile) []string {
	tags := []string{}
	tags = append(tags, profile.Languages...)
	tags = append(tags, profile.Frameworks...)
	tags = append(tags, profile.Runtimes...)
	tags = append(tags, profile.Tooling...)
	return sortedUniqueStrings(tags)
}

func countJSONItems(path string, key string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0
	}
	var items []json.RawMessage
	if err := json.Unmarshal(payload[key], &items); err != nil {
		return 0
	}
	return len(items)
}
