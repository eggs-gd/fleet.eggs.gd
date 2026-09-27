package tasklifecycle

import (
	"regexp"
	"strings"
)

// SystemCommentAuthor is the author Fleet gives its own comments on a card.
// Cards written before the rename carry "core".
const SystemCommentAuthor = "fleet"

// IsSystemCommentAuthor reports whether a comment was written by Fleet itself
// (launch notes, session details, outcomes) rather than by a person or agent.
func IsSystemCommentAuthor(author string) bool {
	switch strings.ToLower(strings.TrimSpace(author)) {
	case SystemCommentAuthor, "core":
		return true
	default:
		return false
	}
}

// Card sections that are for people or for Fleet, not for the worker.
var briefSkippedSections = map[string]bool{
	"raw input":          true,
	"project resolution": true,
	"handoff":            true,
	"activity log":       true,
	"review comments":    true,
}

var (
	briefHeading = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	// Older cards point at Data with "- Workspace: `Work/<id>/PROJECT.md`".
	// The worker works in another directory, so the path is a dead end.
	briefDataPath = regexp.MustCompile("^\\s*[-*]?\\s*Workspace:\\s*`Work/[^`]*`\\s*$")
)

// WorkerBrief is what a worker is told about its task: the request, the
// acceptance criteria, the context, and the review comments people left. It
// is built from the card, but the worker never sees the card itself. It leaves
// out the raw input, Fleet's own comments and log, the resolution notes, the
// hand-off boilerplate, and paths into the data root.
func WorkerBrief(task Task) string {
	var sections []string
	for _, section := range splitCardSections(task.Body) {
		name := strings.ToLower(section.name)
		if briefSkippedSections[name] {
			continue
		}
		text := cleanBriefText(section.text)
		if name == "deliverable" {
			text = meaningfulDeliverable(text)
		}
		if text == "" {
			continue
		}
		if section.name == "" {
			sections = append(sections, text)
			continue
		}
		sections = append(sections, "## "+section.name+"\n\n"+text)
	}
	if comments := peopleComments(task.Comments); comments != "" {
		sections = append(sections, "## Review comments from people\n\n"+comments)
	}
	return strings.Join(sections, "\n\n")
}

type cardSection struct {
	name string
	text string
}

// splitCardSections splits a card body at "## " headings that are not inside
// a code fence. Text before the first heading has an empty name. The leading
// "# Title" line is dropped because the title is given separately.
func splitCardSections(body string) []cardSection {
	var out []cardSection
	current := cardSection{}
	var lines []string
	flush := func() {
		current.text = strings.Join(lines, "\n")
		out = append(out, current)
		lines = nil
	}
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
		}
		if !inFence {
			if match := briefHeading.FindStringSubmatch(line); match != nil {
				flush()
				current = cardSection{name: match[1]}
				continue
			}
			if current.name == "" && len(out) == 0 && strings.HasPrefix(line, "# ") {
				continue
			}
		}
		lines = append(lines, line)
	}
	flush()
	return out
}

func cleanBriefText(text string) string {
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		if briefDataPath.MatchString(line) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// meaningfulDeliverable keeps a "Deliverable" section only when someone wrote
// something in it. The card template used to fill it with "Pending".
func meaningfulDeliverable(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, ":") {
			continue
		}
		if strings.EqualFold(strings.TrimLeft(line, "-* "), "pending.") || strings.EqualFold(strings.TrimLeft(line, "-* "), "pending") {
			continue
		}
		return text
	}
	return ""
}

func peopleComments(comments []Comment) string {
	var lines []string
	for _, comment := range comments {
		if IsSystemCommentAuthor(comment.Author) {
			continue
		}
		text := strings.TrimSpace(comment.Text)
		if text == "" {
			continue
		}
		author := strings.TrimSpace(comment.Author)
		if author == "" {
			author = "unknown"
		}
		lines = append(lines, "- "+author+": "+indentContinuation(text))
	}
	return strings.Join(lines, "\n")
}

func indentContinuation(text string) string {
	return strings.ReplaceAll(text, "\n", "\n  ")
}
