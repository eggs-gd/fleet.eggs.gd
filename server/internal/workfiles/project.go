package workfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// ValidProjectID rejects ids that would leave Work/.
func ValidProjectID(project string) bool {
	project = strings.TrimSpace(project)
	return project != "" && !strings.Contains(project, "..") && !strings.ContainsAny(project, `/\`)
}

func projectCard(root, project string) string {
	return filepath.Join(root, "Work", strings.TrimSpace(project), "PROJECT.md")
}

// AddProjectAlias records an alias in the frontmatter `aliases` list of an
// existing project card, where project resolution reads it. Repeating an alias
// writes nothing. A project without a card is an error: this does not invent
// projects.
func AddProjectAlias(root, project, alias string) (path string, wrote bool, err error) {
	mdfile.EditMu.Lock()
	defer mdfile.EditMu.Unlock()
	path = projectCard(root, project)
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return path, false, fmt.Errorf("project %s has no card yet (%s); the scanner creates cards on its next pass", project, path)
	}
	if err != nil {
		return path, false, err
	}
	next, added, err := mdfile.AddListItem(string(body), "aliases", alias)
	if err != nil {
		return path, false, fmt.Errorf("project card %s is malformed (%w); the scanner repairs cards on its next pass", path, err)
	}
	if !added {
		return path, false, nil
	}
	return path, true, writeAtomic(path, []byte(next))
}

// SetProjectSummary writes the one paragraph under a project card's title and
// marks it confirmed, so the scanner never overwrites it again. It is how
// manager_describe records what a project is — read from the project's own
// repository, approved by the person before it is written.
func SetProjectSummary(root, project, summary string) (path string, err error) {
	mdfile.EditMu.Lock()
	defer mdfile.EditMu.Unlock()
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "", errors.New("summary is required")
	}
	path = projectCard(root, project)
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return path, fmt.Errorf("project %s has no card yet (%s); the scanner creates cards on its next pass", project, path)
	}
	if err != nil {
		return path, err
	}
	text, err := mdfile.SetFirstParagraph(string(body), summary)
	if err != nil {
		return path, fmt.Errorf("project card %s is malformed (%w); the scanner repairs cards on its next pass", path, err)
	}
	text, err = mdfile.SetScalar(text, "summary_source", `"confirmed"`)
	if err != nil {
		return path, fmt.Errorf("project card %s is malformed (%w); the scanner repairs cards on its next pass", path, err)
	}
	return path, writeAtomic(path, []byte(text))
}

// ProjectAliases returns the aliases on a project card. A project without a
// card has none.
func ProjectAliases(root, project string) ([]string, error) {
	if !ValidProjectID(project) {
		return nil, nil
	}
	path := projectCard(root, project)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	aliases, err := mdfile.ListItems(string(raw), "aliases")
	if errors.Is(err, mdfile.ErrNoFrontmatter) {
		// A malformed card has no aliases yet. The scanner repairs it.
		return nil, nil
	}
	return aliases, err
}

// HumanWorker reports whether name is a person in the Fleet roster.
func HumanWorker(root, name string) bool {
	return tasklifecycle.RosterPerson(root, name)
}

// DefaultAssignee reads default_assignee from the project card, or "".
func DefaultAssignee(root, project string) string {
	if strings.TrimSpace(root) == "" || !ValidProjectID(project) {
		return ""
	}
	body, err := os.ReadFile(projectCard(root, project))
	if err != nil {
		return ""
	}
	text := string(body)
	if !strings.HasPrefix(text, "---\n") {
		return ""
	}
	block, _, _ := strings.Cut(text[4:], "\n---")
	for _, line := range strings.Split(block, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "default_assignee:")
		if !ok {
			continue
		}
		return strings.ToLower(strings.Trim(strings.TrimSpace(rest), `"'`))
	}
	return ""
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
