package projectscan

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Workspace is one Work/<id>/PROJECT.md card derived from the registry.
type Workspace struct {
	ID           string
	Title        string
	Kind         string
	ReviewStatus string
	Status       string
	Source       string
	Summary      string
	Stack        []string
	Repositories []workspaceRepo
	Relations    []workspaceRelation
	Protection   []string
}

type workspaceRepo struct {
	ID           string
	Name         string
	Title        string
	RelativePath string
	Remote       string
	Branch       string
	Summary      string
	Stack        []string
	Effective    *Effective
	Detected     *Detected
}

type workspaceRelation struct {
	Kind            string
	Reason          string
	Confidence      string
	RepositoryPaths []string
}

// BuildWorkspaces reads registry JSON and returns workspace cards without writing them.
func BuildWorkspaces(registryDir string) ([]Workspace, error) {
	repos, err := loadWorkspaceRepos(registryDir)
	if err != nil {
		return nil, err
	}
	var groupsFile struct {
		Groups []Group `json:"groups"`
	}
	if err := readJSON(filepath.Join(registryDir, "project-groups.json"), &groupsFile); err != nil {
		return nil, err
	}
	rules := loadProtectionRules(registryDir)
	byID := map[string]workspaceRepo{}
	for _, repo := range repos {
		byID[repo.ID] = repo
	}

	var workspaceGroups []Group
	covered := map[string]struct{}{}
	for _, group := range groupsFile.Groups {
		if group.Kind != "workspace_group" {
			continue
		}
		workspaceGroups = append(workspaceGroups, group)
		for _, id := range group.RepositoryIDs {
			covered[id] = struct{}{}
		}
	}

	var projects []Workspace
	for _, group := range workspaceGroups {
		var members []workspaceRepo
		for _, id := range group.RepositoryIDs {
			if repo, ok := byID[id]; ok {
				members = append(members, repo)
			}
		}
		projects = append(projects, Workspace{
			ID:           group.SuggestedProjectID,
			Title:        workspaceTitle(members),
			Kind:         "workspace_group",
			ReviewStatus: "draft",
			Status:       "discovered",
			Source:       group.Reason,
			Repositories: members,
			Summary:      summarizeProject(members, "workspace_group"),
			Stack:        collectStack(members),
		})
	}
	for _, repo := range repos {
		if _, ok := covered[repo.ID]; ok {
			continue
		}
		projects = append(projects, Workspace{
			ID:           slugify(repo.Name, "project"),
			Title:        repoTitle(repo),
			Kind:         "standalone_repository",
			ReviewStatus: "draft",
			Status:       "discovered",
			Source:       "single repository",
			Repositories: []workspaceRepo{repo},
			Summary:      summarizeProject([]workspaceRepo{repo}, "standalone_repository"),
			Stack:        collectStack([]workspaceRepo{repo}),
		})
	}
	attachProtection(projects, rules)
	attachRelations(projects, groupsFile.Groups, byID, rules)
	sort.Slice(projects, func(i, j int) bool { return projects[i].ID < projects[j].ID })
	return projects, nil
}

func loadWorkspaceRepos(registryDir string) ([]workspaceRepo, error) {
	var payload struct {
		Repositories          []workspaceRepoJSON `json:"repositories"`
		EffectiveRepositories []workspaceRepoJSON `json:"effective_repositories"`
	}
	if err := readJSON(filepath.Join(registryDir, "repositories.json"), &payload); err != nil {
		return nil, err
	}
	raw := payload.Repositories
	if len(raw) == 0 {
		raw = payload.EffectiveRepositories
	}
	byID := map[string]workspaceRepoJSON{}
	byPath := map[string]workspaceRepoJSON{}
	for _, repo := range payload.EffectiveRepositories {
		if repo.ID != "" {
			byID[repo.ID] = repo
		}
		if repo.RelativePath != "" {
			byPath[repo.RelativePath] = repo
		}
	}
	out := make([]workspaceRepo, 0, len(raw))
	for _, repo := range raw {
		effective, ok := byID[repo.ID]
		if !ok {
			effective, ok = byPath[repo.RelativePath]
		}
		merged := repo
		if ok {
			if effective.Effective != nil {
				merged.Effective = effective.Effective
			}
			if effective.Summary != "" {
				merged.Summary = effective.Summary
			}
			if effective.Branch != "" {
				merged.Branch = effective.Branch
			}
			if effective.Remote != "" {
				merged.Remote = effective.Remote
			}
		}
		out = append(out, merged.model())
	}
	return out, nil
}

type workspaceRepoJSON struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Title        string     `json:"title"`
	RelativePath string     `json:"relative_path"`
	Remote       string     `json:"remote"`
	Branch       string     `json:"branch"`
	Summary      string     `json:"summary"`
	Stack        []string   `json:"stack"`
	Effective    *Effective `json:"effective"`
	Detected     *Detected  `json:"detected"`
}

func (repo workspaceRepoJSON) model() workspaceRepo {
	return workspaceRepo{
		ID:           repo.ID,
		Name:         repo.Name,
		Title:        repo.Title,
		RelativePath: repo.RelativePath,
		Remote:       repo.Remote,
		Branch:       repo.Branch,
		Summary:      repo.Summary,
		Stack:        repo.Stack,
		Effective:    repo.Effective,
		Detected:     repo.Detected,
	}
}

func workspaceTitle(repos []workspaceRepo) string {
	if len(repos) == 0 {
		return ""
	}
	return strings.Split(repos[0].RelativePath, "/")[0]
}

func repoTitle(repo workspaceRepo) string {
	title := repo.Title
	if title == "" {
		title = repo.Name
	}
	lower := strings.ToLower(repo.Name)
	if strings.HasSuffix(lower, "_sonar") || strings.HasSuffix(lower, "-sonar") {
		return repo.Name
	}
	switch strings.ToLower(title) {
	case "sv", "how to use", "screenshots", "installation uri", "my project", "set up instructions":
		return repo.Name
	default:
		return title
	}
}

func summarizeProject(repos []workspaceRepo, kind string) string {
	if kind == "workspace_group" && len(repos) > 0 {
		root := strings.Split(repos[0].RelativePath, "/")[0]
		return fmt.Sprintf("Workspace group containing %d repositories under `%s/`.", len(repos), root)
	}
	if len(repos) == 1 {
		if repos[0].Summary != "" {
			return repos[0].Summary
		}
		return "Project backed by " + repos[0].RelativePath + "."
	}
	return fmt.Sprintf("Project containing %d repositories.", len(repos))
}

func collectStack(repos []workspaceRepo) []string {
	var stack []string
	for _, repo := range repos {
		if repo.Effective != nil {
			stack = append(stack, repo.Effective.Languages...)
			stack = append(stack, repo.Effective.Frameworks...)
			stack = append(stack, repo.Effective.Runtimes...)
			stack = append(stack, repo.Effective.Tooling...)
			continue
		}
		stack = append(stack, repo.Stack...)
	}
	if stack == nil {
		return []string{}
	}
	return sortedUnique(stack)
}

func attachProtection(projects []Workspace, rules protectionRules) {
	for i := range projects {
		seen := map[string]struct{}{}
		for _, repo := range projects[i].Repositories {
			_, reason := protectedRepository(repo.RelativePath, rules)
			if reason == "" {
				continue
			}
			if _, ok := seen[reason]; ok {
				continue
			}
			seen[reason] = struct{}{}
			projects[i].Protection = append(projects[i].Protection, reason)
		}
	}
}

func attachRelations(projects []Workspace, groups []Group, byID map[string]workspaceRepo, rules protectionRules) {
	projectByRepo := map[string]int{}
	for i, project := range projects {
		for _, repo := range project.Repositories {
			projectByRepo[repo.ID] = i
		}
	}
	for _, group := range groups {
		if group.Kind == "workspace_group" {
			continue
		}
		var paths []string
		projectIDs := map[int]struct{}{}
		for _, id := range group.RepositoryIDs {
			repo, ok := byID[id]
			if !ok {
				continue
			}
			paths = append(paths, repo.RelativePath)
			if index, ok := projectByRepo[id]; ok {
				projectIDs[index] = struct{}{}
			}
		}
		if len(paths) > 0 && allProtected(paths, rules) {
			continue
		}
		relation := workspaceRelation{
			Kind:            group.Kind,
			Reason:          group.Reason,
			Confidence:      group.Confidence,
			RepositoryPaths: paths,
		}
		for index := range projectIDs {
			projects[index].Relations = append(projects[index].Relations, relation)
		}
	}
}

func allProtected(paths []string, rules protectionRules) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		if _, reason := protectedRepository(path, rules); reason == "" {
			return false
		}
	}
	return true
}
