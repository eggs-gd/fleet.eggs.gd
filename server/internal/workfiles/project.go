package workfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
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

// Milestones returns the milestones on a project card. A project without a
// card has none.
func Milestones(root, project string) ([]board.Milestone, error) {
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
	records, err := mdfile.ListRecords(string(raw), "milestones")
	if errors.Is(err, mdfile.ErrNoFrontmatter) {
		// A malformed card has no milestones yet. The scanner repairs it.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return recordsToMilestones(records), nil
}

// CreateMilestone appends a new, parked milestone to a project card and
// returns it. The id is slugified from title and de-duplicated against
// existing milestone ids on the same card.
func CreateMilestone(root, project, title, version string) (board.Milestone, error) {
	if !ValidProjectID(project) {
		return board.Milestone{}, errors.New("project id is required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return board.Milestone{}, errors.New("title is required")
	}
	if err := singleLine("title", title); err != nil {
		return board.Milestone{}, err
	}
	if err := singleLine("version", version); err != nil {
		return board.Milestone{}, err
	}
	mdfile.EditMu.Lock()
	defer mdfile.EditMu.Unlock()
	path := projectCard(root, project)
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return board.Milestone{}, fmt.Errorf("project %s has no card yet (%s); the scanner creates cards on its next pass", project, path)
	}
	if err != nil {
		return board.Milestone{}, err
	}
	doc, err := mdfile.ParseDoc(string(body))
	if err != nil {
		return board.Milestone{}, fmt.Errorf("project card %s is malformed (%w); the scanner repairs cards on its next pass", path, err)
	}
	records := doc.Records("milestones")
	id := uniqueMilestoneID(records, title)
	milestone := board.Milestone{ID: id, Title: title, Version: strings.TrimSpace(version), State: "parked"}
	doc.SetRecords("milestones", board.MilestoneFields, append(records, milestoneToRecord(milestone)))
	if err := writeAtomic(path, []byte(doc.String())); err != nil {
		return board.Milestone{}, err
	}
	return milestone, nil
}

// SetMilestoneState updates one milestone's state (parked/active/done) on a
// project card.
func SetMilestoneState(root, project, milestoneID, state string) (board.Milestone, error) {
	return PatchMilestone(root, project, milestoneID, &state, nil, nil)
}

// PatchMilestone applies whichever of state/title/version is non-nil to one
// milestone in a single read-modify-write of the card. A non-nil empty version
// clears it; a title cannot be emptied.
func PatchMilestone(root, project, milestoneID string, state, title, version *string) (board.Milestone, error) {
	if state != nil && !board.IsKnownMilestoneState(*state) {
		return board.Milestone{}, fmt.Errorf("unknown milestone state %q", *state)
	}
	if title != nil {
		trimmed := strings.TrimSpace(*title)
		if trimmed == "" {
			return board.Milestone{}, errors.New("title cannot be empty")
		}
		if err := singleLine("title", trimmed); err != nil {
			return board.Milestone{}, err
		}
		title = &trimmed
	}
	if version != nil {
		trimmed := strings.TrimSpace(*version)
		if err := singleLine("version", trimmed); err != nil {
			return board.Milestone{}, err
		}
		version = &trimmed
	}
	return updateMilestone(root, project, milestoneID, func(m *board.Milestone) {
		if state != nil {
			m.State = *state
		}
		if title != nil {
			m.Title = *title
		}
		if version != nil {
			m.Version = *version
		}
	})
}

// DeleteMilestone removes one milestone from a project card. Tasks that still
// point at it are the caller's to clear first (see Dashboard.DeleteMilestone).
func DeleteMilestone(root, project, milestoneID string) error {
	if !ValidProjectID(project) {
		return errors.New("project id is required")
	}
	mdfile.EditMu.Lock()
	defer mdfile.EditMu.Unlock()
	path := projectCard(root, project)
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("project %s has no card yet (%s); the scanner creates cards on its next pass", project, path)
	}
	if err != nil {
		return err
	}
	doc, err := mdfile.ParseDoc(string(body))
	if err != nil {
		return fmt.Errorf("project card %s is malformed (%w); the scanner repairs cards on its next pass", path, err)
	}
	records := doc.Records("milestones")
	kept := make([]map[string]string, 0, len(records))
	for _, record := range records {
		if record["id"] != milestoneID {
			kept = append(kept, record)
		}
	}
	if len(kept) == len(records) {
		return fmt.Errorf("project %s has no milestone %q", project, milestoneID)
	}
	if len(kept) == 0 {
		doc.Remove("milestones")
	} else {
		doc.SetRecords("milestones", board.MilestoneFields, kept)
	}
	return writeAtomic(path, []byte(doc.String()))
}

func updateMilestone(root, project, milestoneID string, apply func(*board.Milestone)) (board.Milestone, error) {
	if !ValidProjectID(project) {
		return board.Milestone{}, errors.New("project id is required")
	}
	mdfile.EditMu.Lock()
	defer mdfile.EditMu.Unlock()
	path := projectCard(root, project)
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return board.Milestone{}, fmt.Errorf("project %s has no card yet (%s); the scanner creates cards on its next pass", project, path)
	}
	if err != nil {
		return board.Milestone{}, err
	}
	doc, err := mdfile.ParseDoc(string(body))
	if err != nil {
		return board.Milestone{}, fmt.Errorf("project card %s is malformed (%w); the scanner repairs cards on its next pass", path, err)
	}
	milestones := recordsToMilestones(doc.Records("milestones"))
	index := -1
	for i, m := range milestones {
		if m.ID == milestoneID {
			index = i
			break
		}
	}
	if index < 0 {
		return board.Milestone{}, fmt.Errorf("project %s has no milestone %q", project, milestoneID)
	}
	apply(&milestones[index])
	doc.SetRecords("milestones", board.MilestoneFields, milestonesToRecords(milestones))
	if err := writeAtomic(path, []byte(doc.String())); err != nil {
		return board.Milestone{}, err
	}
	return milestones[index], nil
}

func singleLine(field, value string) error {
	if strings.ContainsFunc(value, unicode.IsControl) {
		return fmt.Errorf("%s must be a single line", field)
	}
	return nil
}

func uniqueMilestoneID(records []map[string]string, title string) string {
	base := tasklifecycle.SlugifyTitle(title)
	if base == "" {
		base = "milestone"
	}
	taken := map[string]bool{}
	for _, record := range records {
		taken[record["id"]] = true
	}
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if !taken[candidate] {
			return candidate
		}
	}
}

func milestoneToRecord(m board.Milestone) map[string]string {
	record := map[string]string{"id": m.ID, "title": m.Title, "state": m.State}
	if m.Version != "" {
		record["version"] = m.Version
	}
	return record
}

func recordsToMilestones(records []map[string]string) []board.Milestone {
	milestones := make([]board.Milestone, 0, len(records))
	for _, record := range records {
		milestones = append(milestones, board.Milestone{
			ID:      record["id"],
			Title:   record["title"],
			Version: record["version"],
			State:   record["state"],
		})
	}
	return milestones
}

func milestonesToRecords(milestones []board.Milestone) []map[string]string {
	records := make([]map[string]string, 0, len(milestones))
	for _, m := range milestones {
		records = append(records, milestoneToRecord(m))
	}
	return records
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
