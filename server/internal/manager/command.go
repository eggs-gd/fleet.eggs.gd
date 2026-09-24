package manager

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	refPattern = regexp.MustCompile(`(?i)\b(?:CORE|INBOX|LIFE)-(\d+)\b`)
	// Matches "задача 57", "таска 57", "task 57", "core 57". Go's RE2 \b is
	// ASCII-only word-boundary (based on [0-9A-Za-z_]), so it never matches
	// before/after Cyrillic letters — "задача"/"таска" would silently never
	// match with \b. \p{L}/\p{N} boundaries below are Unicode-aware; RE2 has
	// no lookaround, so the boundary characters are consumed into the match,
	// but callers only read the captured digit group, not the full match.
	looseRefPattern = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}_])(?:задача|таска|task|core)\s*#?\s*(\d+)(?:$|[^\p{N}])`)
	priorityPattern = regexp.MustCompile(`(?i)\b(?:priority|prio|p)\s*[:=]?\s*([1-5])\b|\bP([1-5])\b`)
)

var knownStatuses = map[string]string{
	"backlog":      "backlog",
	"todo":         "todo",
	"doing":        "doing",
	"blocked":      "blocked",
	"needs_review": "needs_review",
	"needs-review": "needs_review",
	"needsreview":  "needs_review",
	"review":       "needs_review",
	"needs_rework": "needs_rework",
	"needs-rework": "needs_rework",
	"rework":       "needs_rework",
	"done":         "done",
	"archived":     "archived",
	"archive":      "archived",
}

var knownAssignees = map[string]bool{
	"alex":       true,
	"codex":      true,
	"claude":     true,
	"cursor":     true,
	"unassigned": true,
}

// ParseFastPath attempts deterministic intent extraction without an LLM.
// Returns (nil, nil) when the text is not an unambiguous board command
// (caller should fall through to the Manager LLM).
func ParseFastPath(text string) (*Intent, *Failure) {
	normalized := collapseSpace(strings.TrimSpace(text))
	if normalized == "" {
		return nil, &Failure{Code: FailureValidation, Message: "text is required"}
	}
	lower := strings.ToLower(normalized)

	if intent, fail := parseShow(lower, normalized); intent != nil || fail != nil {
		return intent, fail
	}
	if intent, fail := parseMove(lower, normalized); intent != nil || fail != nil {
		return intent, fail
	}
	if intent, fail := parseComment(lower, normalized); intent != nil || fail != nil {
		return intent, fail
	}
	if intent, fail := parseAssign(lower, normalized); intent != nil || fail != nil {
		return intent, fail
	}
	if intent, fail := parsePriority(lower, normalized); intent != nil || fail != nil {
		return intent, fail
	}
	if intent, fail := parseCancel(lower, normalized); intent != nil || fail != nil {
		return intent, fail
	}

	// Not an unambiguous deterministic command.
	return nil, nil
}

func parseShow(lower, original string) (*Intent, *Failure) {
	if lower == "show board" || lower == "show the board" || lower == "board" {
		return &Intent{Kind: KindBoardCommand, BoardAction: BoardShowBoard, RawTranscript: original}, nil
	}
	if strings.HasPrefix(lower, "show board for ") {
		project := strings.TrimSpace(original[len(original)-len(lower)+len("show board for "):])
		if project == "" {
			return nil, &Failure{Code: FailureAmbiguousCommand, Message: "show board for <project> needs a project"}
		}
		return &Intent{Kind: KindBoardCommand, BoardAction: BoardShowBoard, Project: project, RawTranscript: original}, nil
	}
	if strings.HasPrefix(lower, "show board ") {
		project := strings.TrimSpace(original[len(original)-len(lower)+len("show board "):])
		if project == "" {
			return &Intent{Kind: KindBoardCommand, BoardAction: BoardShowBoard, RawTranscript: original}, nil
		}
		return &Intent{Kind: KindBoardCommand, BoardAction: BoardShowBoard, Project: project, RawTranscript: original}, nil
	}

	if strings.HasPrefix(lower, "show ") {
		ref := extractRef(original)
		if ref == "" {
			// Tight incomplete command → ambiguous. Free-form "show me …" falls through to LLM.
			rest := strings.TrimSpace(lower[len("show "):])
			if rest == "" || rest == "task" || rest == "the task" {
				return nil, &Failure{Code: FailureAmbiguousCommand, Message: "show command needs a CORE-* ref or 'board'"}
			}
			return nil, nil
		}
		return &Intent{Kind: KindBoardCommand, BoardAction: BoardShowTask, Ref: ref, RawTranscript: original}, nil
	}
	return nil, nil
}

func parseMove(lower, original string) (*Intent, *Failure) {
	if !strings.HasPrefix(lower, "move ") && !strings.HasPrefix(lower, "set status of ") {
		return nil, nil
	}

	ref := extractRef(original)
	if ref == "" {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: "move/status command needs a CORE-* ref"}
	}
	status := extractStatusAfterTo(lower)
	if status == "" {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: "move/status command needs a target status"}
	}
	return &Intent{Kind: KindStatusChange, Ref: ref, Status: status, RawTranscript: original}, nil
}

func parseComment(lower, original string) (*Intent, *Failure) {
	if !strings.HasPrefix(lower, "add comment") && !strings.HasPrefix(lower, "comment on") {
		return nil, nil
	}
	ref := extractRef(original)
	if ref == "" {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: "comment command needs a CORE-* ref"}
	}
	comment := ""
	if idx := strings.Index(original, ":"); idx >= 0 {
		comment = strings.TrimSpace(original[idx+1:])
	}
	if comment == "" {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: "comment command needs text after ':'"}
	}
	return &Intent{
		Kind:          KindComment,
		Ref:           ref,
		Comment:       comment,
		CommentAuthor: "alex",
		RawTranscript: original,
	}, nil
}

func parseAssign(lower, original string) (*Intent, *Failure) {
	if !strings.HasPrefix(lower, "assign ") && !strings.HasPrefix(lower, "change assignee") {
		return nil, nil
	}
	ref := extractRef(original)
	if ref == "" {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: "assign command needs a CORE-* ref"}
	}
	assignee := ""
	if idx := strings.LastIndex(lower, " to "); idx >= 0 {
		assignee = strings.TrimSpace(lower[idx+4:])
	} else if idx := strings.LastIndex(lower, " "); idx >= 0 {
		assignee = strings.TrimSpace(lower[idx+1:])
	}
	assignee = strings.Trim(assignee, ".")
	if !knownAssignees[assignee] {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: fmt.Sprintf("unknown assignee %q", assignee)}
	}
	return &Intent{Kind: KindAssigneeChange, Ref: ref, Assignee: assignee, RawTranscript: original}, nil
}

func parsePriority(lower, original string) (*Intent, *Failure) {
	if !strings.HasPrefix(lower, "set priority") && !strings.HasPrefix(lower, "change priority") {
		return nil, nil
	}
	ref := extractRef(original)
	if ref == "" {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: "priority command needs a CORE-* ref"}
	}
	priority, ok := extractPriority(lower)
	if !ok {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: "priority command needs a value 1-5"}
	}
	return &Intent{Kind: KindPriorityChange, Ref: ref, Priority: &priority, RawTranscript: original}, nil
}

func parseCancel(lower, original string) (*Intent, *Failure) {
	if !strings.HasPrefix(lower, "cancel ") && !strings.HasPrefix(lower, "archive ") {
		return nil, nil
	}
	ref := extractRef(original)
	if ref == "" {
		return nil, &Failure{Code: FailureAmbiguousCommand, Message: "cancel/archive command needs a CORE-* ref"}
	}
	confirm := strings.Contains(lower, " confirm") || strings.HasSuffix(lower, " confirm")
	if !confirm {
		return nil, &Failure{
			Code:    FailureUnsafeCommand,
			Message: fmt.Sprintf("refusing to archive %s without explicit 'confirm'", ref),
		}
	}
	return &Intent{Kind: KindCancel, Ref: ref, Confirm: true, Status: "archived", RawTranscript: original}, nil
}

func extractRef(text string) string {
	if match := refPattern.FindStringSubmatch(text); len(match) == 2 {
		prefix := strings.ToUpper(match[0][:strings.Index(match[0], "-")])
		return prefix + "-" + match[1]
	}
	if match := looseRefPattern.FindStringSubmatch(text); len(match) == 2 {
		return "CORE-" + match[1]
	}
	return ""
}

func extractStatusAfterTo(lower string) string {
	idx := strings.LastIndex(lower, " to ")
	if idx < 0 {
		return ""
	}
	candidate := strings.TrimSpace(lower[idx+4:])
	candidate = strings.Trim(candidate, ".")
	candidate = strings.ReplaceAll(candidate, " ", "_")
	candidate = strings.ReplaceAll(candidate, "-", "_")
	if status, ok := knownStatuses[candidate]; ok {
		return status
	}
	// allow "needs review"
	candidate = strings.ReplaceAll(strings.TrimSpace(lower[idx+4:]), " ", "_")
	if status, ok := knownStatuses[candidate]; ok {
		return status
	}
	return ""
}

func extractPriority(lower string) (int, bool) {
	if match := priorityPattern.FindStringSubmatch(lower); len(match) >= 2 {
		for _, part := range match[1:] {
			if part == "" {
				continue
			}
			n, err := strconv.Atoi(part)
			if err == nil && n >= 1 && n <= 5 {
				return n, true
			}
		}
	}
	if idx := strings.LastIndex(lower, " to "); idx >= 0 {
		n, err := strconv.Atoi(strings.TrimSpace(lower[idx+4:]))
		if err == nil && n >= 1 && n <= 5 {
			return n, true
		}
	}
	return 0, false
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
