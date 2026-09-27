package projectscan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryFactsForReadsSummaryAndReadme(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "_registry")
	if err := os.MkdirAll(registry, 0o755); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("# Acme\n\nA tool that does the thing.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repos := `{"repositories": [
		{"relative_path": "acme", "summary": "A tool that does the thing.", "summary_source": "readme", "readme_path": "` + readme + `"},
		{"relative_path": "no-readme", "summary": "", "summary_source": "readme-title", "readme_path": ""}
	]}`
	if err := os.WriteFile(filepath.Join(registry, "repositories.json"), []byte(repos), 0o644); err != nil {
		t.Fatal(err)
	}

	facts, err := RepositoryFactsFor(registry, []string{"acme", "no-readme", "unknown"}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 2 {
		t.Fatalf("an unknown path must be skipped, not errored: %#v", facts)
	}
	if facts[0].Summary != "A tool that does the thing." || facts[0].SummarySource != "readme" {
		t.Fatalf("acme facts = %#v", facts[0])
	}
	if facts[0].ReadmeExcerpt != "# Acme\n\nA tool that does the thing." {
		t.Fatalf("readme excerpt = %q", facts[0].ReadmeExcerpt)
	}
	if facts[1].ReadmeExcerpt != "" {
		t.Fatalf("a repository with no README path must have no excerpt: %#v", facts[1])
	}
}

func TestRepositoryFactsForMissingRegistry(t *testing.T) {
	root := t.TempDir()
	facts, err := RepositoryFactsFor(filepath.Join(root, "_registry"), []string{"acme"}, 1000)
	if err != nil || facts != nil {
		t.Fatalf("facts = %#v, %v", facts, err)
	}
}
