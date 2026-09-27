package projectscan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
)

// CardReport lists what MaintainCards changed and what it could not fix.
type CardReport struct {
	Created  []string
	Repaired []string
	// Tagged lists the projects that got a new ref tag.
	Tagged []string
	// Problems are cards a person has to fix: a duplicate or malformed tag, or a
	// card without frontmatter that the registry does not know.
	Problems []string
}

// MaintainCards keeps Work/<id>/PROJECT.md in step with the registry, without
// rewriting what people and the Manager wrote. It creates a missing card,
// restores a missing frontmatter block, makes `aliases` a block list, and
// syncs the `repositories` list. Every card also gets a ref `tag` when it has
// none (see assignTags). The body, aliases and default_assignee are never
// touched. Cards for ids the registry does not know are kept
// except for their tag. A registry that has not been scanned yet is not an
// error.
func MaintainCards(registryDir, workDir string) (CardReport, error) {
	mdfile.EditMu.Lock()
	defer mdfile.EditMu.Unlock()
	var report CardReport
	projects, err := BuildWorkspaces(registryDir)
	if errors.Is(err, fs.ErrNotExist) {
		projects = nil
	} else if err != nil {
		return report, fmt.Errorf("read registry: %w", err)
	}
	for _, project := range projects {
		dir := filepath.Join(workDir, project.ID)
		path := filepath.Join(dir, "PROJECT.md")
		rendered := renderWorkspace(project)
		existing, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0o755); err != nil {
				return report, err
			}
			if err := writeFileAtomic(path, []byte(rendered)); err != nil {
				return report, err
			}
			report.Created = append(report.Created, project.ID)
			continue
		}
		if err != nil {
			return report, err
		}
		repaired, changed, err := repairCard(string(existing), rendered, project)
		if err != nil {
			return report, fmt.Errorf("card %s: %w", path, err)
		}
		if !changed {
			continue
		}
		if err := writeFileAtomic(path, []byte(repaired)); err != nil {
			return report, err
		}
		report.Repaired = append(report.Repaired, project.ID)
	}
	return assignTags(workDir, report)
}

func repairCard(existing, rendered string, project Workspace) (string, bool, error) {
	text := existing
	if !strings.HasPrefix(text, "---\n") || !strings.Contains(text[4:], "\n---") {
		head, _, ok := strings.Cut(rendered, "\n---\n")
		if !ok {
			return "", false, errors.New("rendered card has no frontmatter")
		}
		text = head + "\n---\n\n" + strings.TrimLeft(existing, "\n")
	}
	if items, err := mdfile.ListItems(text, "aliases"); err != nil {
		return "", false, err
	} else if len(items) > 0 {
		next, err := mdfile.SetList(text, "aliases", items)
		if err != nil {
			return "", false, err
		}
		text = next
	}
	want := make([]string, 0, len(project.Repositories))
	for _, repo := range project.Repositories {
		want = append(want, jsonString(repo.RelativePath))
	}
	have, err := mdfile.ListItems(text, "repositories")
	if err != nil {
		return "", false, err
	}
	if !sameStrings(have, unquoted(want)) {
		next, err := mdfile.SetList(text, "repositories", want)
		if err != nil {
			return "", false, err
		}
		text = next
	}
	return text, text != existing, nil
}

func unquoted(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strings.Trim(v, `"`)
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return err
	}
	return os.Rename(name, path)
}
