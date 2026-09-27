package projectscan

import (
	"os"
	"path/filepath"
	"strings"
)

// RepositoryFacts is what a Manager describing a project needs about one of
// its repositories: the one-line summary the scanner already computed, how
// confident that summary is, and an excerpt of the README behind it — read
// fresh, not the single line stored in the registry.
type RepositoryFacts struct {
	RelativePath  string
	Summary       string
	SummarySource string
	ReadmeExcerpt string
}

// RepositoryFactsFor reads _registry/repositories.json and returns the facts
// for the repositories at the given relative paths, in that order. A path not
// found in the registry is skipped, not an error.
func RepositoryFactsFor(registryDir string, relativePaths []string, maxReadmeBytes int) ([]RepositoryFacts, error) {
	var payload struct {
		Repositories []struct {
			RelativePath  string `json:"relative_path"`
			Summary       string `json:"summary"`
			SummarySource string `json:"summary_source"`
			ReadmePath    string `json:"readme_path"`
		} `json:"repositories"`
	}
	if err := readJSON(filepath.Join(registryDir, "repositories.json"), &payload); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	byPath := make(map[string]RepositoryFacts, len(payload.Repositories))
	for _, repo := range payload.Repositories {
		byPath[repo.RelativePath] = RepositoryFacts{
			RelativePath:  repo.RelativePath,
			Summary:       repo.Summary,
			SummarySource: repo.SummarySource,
			ReadmeExcerpt: readExcerpt(repo.ReadmePath, maxReadmeBytes),
		}
	}
	out := make([]RepositoryFacts, 0, len(relativePaths))
	for _, path := range relativePaths {
		if facts, ok := byPath[path]; ok {
			out = append(out, facts)
		}
	}
	return out, nil
}

func readExcerpt(path string, maxBytes int) string {
	if path == "" || maxBytes <= 0 {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(data) > maxBytes {
		data = data[:maxBytes]
	}
	return strings.TrimSpace(string(data))
}
