package projectscan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ScanOptions selects the project trees to walk and the Data registry to write.
type ScanOptions struct {
	ScanRoots   []string
	RegistryDir string
	// WorkDir is Data/Work, used only to read "## Technology Overrides" from PROJECT.md.
	WorkDir string
}

// ScanResult counts what the scan wrote.
type ScanResult struct {
	Roots              []string
	RegistryDir        string
	Repositories       int
	DeletionCandidates int
}

// Scan discovers git repositories under each ScanRoot and rewrites the registry files.
func Scan(opts ScanOptions) (ScanResult, error) {
	roots, err := resolveScanRoots(opts.ScanRoots)
	if err != nil {
		return ScanResult{}, err
	}
	if len(roots) == 0 {
		return ScanResult{}, fmt.Errorf("scan root is required")
	}
	registryDir, err := filepath.Abs(opts.RegistryDir)
	if err != nil {
		return ScanResult{}, err
	}
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		return ScanResult{}, err
	}
	workDir := opts.WorkDir
	if workDir == "" && len(roots) == 1 {
		for _, candidate := range []string{filepath.Join(roots[0], "Work"), filepath.Join(roots[0], "core.eggs.gd", "Work")} {
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				workDir = candidate
				break
			}
		}
	}

	var repositories []Repository
	for _, root := range roots {
		found, err := buildRepositories(root, registryDir, workDir)
		if err != nil {
			return ScanResult{}, err
		}
		for i := range found {
			found[i].ScanRoot = root
		}
		repositories = append(repositories, found...)
	}
	groups := suggestGroups(repositories)
	rules := loadProtectionRules(registryDir)
	candidates := suggestDeletionCandidates(repositories, rules)
	generatedAt := utcNow()
	if len(repositories) > 0 {
		generatedAt = repositories[0].LastSeenAt
	}

	if err := writeJSON(filepath.Join(registryDir, "repositories.json"), map[string]any{
		"generated_at":           generatedAt,
		"roots":                  roots,
		"repositories":           repositoryRecords(repositories),
		"effective_repositories": effectiveRecords(repositories),
	}); err != nil {
		return ScanResult{}, err
	}
	if err := writeJSON(filepath.Join(registryDir, "project-groups.json"), map[string]any{
		"generated_at": generatedAt,
		"roots":        roots,
		"groups":       groups,
	}); err != nil {
		return ScanResult{}, err
	}
	if err := writeJSON(filepath.Join(registryDir, "deletion-candidates.json"), map[string]any{
		"generated_at": generatedAt,
		"roots":        roots,
		"candidates":   candidates,
	}); err != nil {
		return ScanResult{}, err
	}
	rootLabel := joinComma(roots)
	if err := os.WriteFile(filepath.Join(registryDir, "bootstrap-report.md"), []byte(renderBootstrapReport(rootLabel, repositories, groups, candidates, generatedAt)), 0o644); err != nil {
		return ScanResult{}, err
	}
	if err := os.WriteFile(filepath.Join(registryDir, "deletion-candidates.md"), []byte(renderDeletionReport(candidates, generatedAt)), 0o644); err != nil {
		return ScanResult{}, err
	}
	return ScanResult{
		Roots:              roots,
		RegistryDir:        registryDir,
		Repositories:       len(repositories),
		DeletionCandidates: len(candidates),
	}, nil
}

func resolveScanRoots(roots []string) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		root = filepath.Clean(root)
		if root == "" || root == "." {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		abs = resolvePath(abs)
		if _, ok := seen[abs]; ok {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("scan root %s is not a directory", abs)
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	return out, nil
}

func writeJSON(path string, payload any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

type repositoryRecord struct {
	ID                  string    `json:"id"`
	RemoteIdentity      *string   `json:"remote_identity"`
	Name                string    `json:"name"`
	Path                string    `json:"path"`
	RelativePath        string    `json:"relative_path"`
	ScanRoot            string    `json:"scan_root"`
	Remote              *string   `json:"remote"`
	Branch              *string   `json:"branch"`
	Head                *string   `json:"head"`
	ReadmePath          *string   `json:"readme_path"`
	Title               string    `json:"title"`
	Summary             string    `json:"summary"`
	SummarySource       string    `json:"summary_source"`
	Detected            Detected  `json:"detected"`
	Effective           Effective `json:"effective"`
	Stack               []string  `json:"stack"`
	Markers             []string  `json:"markers"`
	ParentRepositoryID  *string   `json:"parent_repository_id"`
	NestedRepositoryIDs []string  `json:"nested_repository_ids"`
	FirstSeenAt         string    `json:"first_seen_at"`
	LastSeenAt          string    `json:"last_seen_at"`
}

type effectiveRecord struct {
	ID                  string     `json:"id"`
	RemoteIdentity      *string    `json:"remote_identity"`
	Name                string     `json:"name"`
	Path                string     `json:"path"`
	RelativePath        string     `json:"relative_path"`
	ScanRoot            string     `json:"scan_root"`
	Remote              *string    `json:"remote"`
	Branch              *string    `json:"branch"`
	Summary             string     `json:"summary"`
	Effective           Effective  `json:"effective"`
	Evidence            []Evidence `json:"evidence"`
	ParentRepositoryID  *string    `json:"parent_repository_id"`
	NestedRepositoryIDs []string   `json:"nested_repository_ids"`
}

func repositoryRecords(repositories []Repository) []repositoryRecord {
	out := make([]repositoryRecord, 0, len(repositories))
	for _, repo := range repositories {
		out = append(out, repositoryRecord{
			ID:                  repo.ID,
			RemoteIdentity:      strPtr(repo.RemoteIdentity),
			Name:                repo.Name,
			Path:                repo.Path,
			RelativePath:        repo.RelativePath,
			ScanRoot:            repo.ScanRoot,
			Remote:              strPtr(repo.Remote),
			Branch:              strPtr(repo.Branch),
			Head:                strPtr(repo.Head),
			ReadmePath:          strPtr(repo.ReadmePath),
			Title:               repo.Title,
			Summary:             repo.Summary,
			SummarySource:       repo.SummarySource,
			Detected:            ensureDetected(repo.Detected),
			Effective:           ensureEffective(repo.Effective),
			Stack:               nonNil(repo.Stack),
			Markers:             nonNil(repo.Markers),
			ParentRepositoryID:  strPtr(repo.ParentRepositoryID),
			NestedRepositoryIDs: nonNil(repo.NestedRepositoryIDs),
			FirstSeenAt:         repo.FirstSeenAt,
			LastSeenAt:          repo.LastSeenAt,
		})
	}
	return out
}

func effectiveRecords(repositories []Repository) []effectiveRecord {
	out := make([]effectiveRecord, 0, len(repositories))
	for _, repo := range repositories {
		out = append(out, effectiveRecord{
			ID:                  repo.ID,
			RemoteIdentity:      strPtr(repo.RemoteIdentity),
			Name:                repo.Name,
			Path:                repo.Path,
			RelativePath:        repo.RelativePath,
			ScanRoot:            repo.ScanRoot,
			Remote:              strPtr(repo.Remote),
			Branch:              strPtr(repo.Branch),
			Summary:             repo.Summary,
			Effective:           ensureEffective(repo.Effective),
			Evidence:            ensureDetected(repo.Detected).Evidence,
			ParentRepositoryID:  strPtr(repo.ParentRepositoryID),
			NestedRepositoryIDs: nonNil(repo.NestedRepositoryIDs),
		})
	}
	return out
}

func ensureDetected(detected Detected) Detected {
	if detected.Languages == nil {
		detected.Languages = []string{}
	}
	if detected.Frameworks == nil {
		detected.Frameworks = []string{}
	}
	if detected.Runtimes == nil {
		detected.Runtimes = []string{}
	}
	if detected.Tooling == nil {
		detected.Tooling = []string{}
	}
	if detected.Evidence == nil {
		detected.Evidence = []Evidence{}
	}
	return detected
}

func ensureEffective(effective Effective) Effective {
	if effective.Languages == nil {
		effective.Languages = []string{}
	}
	if effective.Frameworks == nil {
		effective.Frameworks = []string{}
	}
	if effective.Runtimes == nil {
		effective.Runtimes = []string{}
	}
	if effective.Tooling == nil {
		effective.Tooling = []string{}
	}
	return effective
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
