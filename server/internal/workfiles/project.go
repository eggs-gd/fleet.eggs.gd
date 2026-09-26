package workfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// AddProjectNote records an alias, note, or decision on an existing project
// card. An alias goes into the frontmatter aliases list, where project
// resolution reads it; every write also appends a timestamped Activity Log
// line. Repeating an alias, note, or decision writes nothing. A project
// without a card is an error: this does not invent projects.
func AddProjectNote(root, project, kind, text string, now time.Time) (path string, wrote bool, err error) {
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
	current := string(body)
	if kind == "alias" {
		next, added, err := mdfile.AddListItem(current, "aliases", text)
		if err != nil {
			return path, false, fmt.Errorf("project card %s is malformed (%w); the scanner repairs cards on its next pass", path, err)
		}
		if !added {
			return path, false, nil
		}
		current = next
	} else if strings.Contains(current, " "+kind+": "+text+"\n") {
		return path, false, nil
	}
	if !strings.Contains(current, "## Activity Log") {
		current = strings.TrimRight(current, "\n") + "\n\n## Activity Log\n"
	}
	current = strings.TrimRight(current, "\n") + fmt.Sprintf("\n- %s %s: %s\n", now.Format(time.RFC3339), kind, text)
	return path, true, writeAtomic(path, []byte(current))
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
