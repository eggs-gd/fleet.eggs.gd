package tasklifecycle

import (
	"strings"
	"testing"
)

const oldCard = "# Fix billing\n\n## Request\n\nMake the invoice total correct.\n\n## Acceptance\nTotals match the ledger.\n\n## Raw Input\n\nmake the invoice thing right pls\n\n## Project Resolution\n\n- Matched workspace: `acme`\n- Resolution method: backoffice dashboard\n\n## Acceptance Criteria\n\n- [ ] Describe the expected outcome.\n\n## Context\n\n- Workspace: `Work/acme/PROJECT.md`\n- Repository: `acme/web`\n\n## Deliverable\n\nExpected physical artifact:\n\n- Pending.\n\nProduced artifacts:\n\n- Pending.\n\n## Handoff\n\nCreated from the backoffice dashboard.\nLaunch policy: todo tasks are daemon-pickup candidates.\n\n## Review Comments\n\n## Activity Log\n\n- 2026-08-01T10:00:00+03:00 — Created.\n"

func TestWorkerBriefKeepsWhatTheWorkerNeeds(t *testing.T) {
	brief := WorkerBrief(Task{Body: oldCard})
	for _, want := range []string{"Make the invoice total correct.", "## Acceptance", "Totals match the ledger.", "Repository: `acme/web`"} {
		if !strings.Contains(brief, want) {
			t.Errorf("brief lacks %q:\n%s", want, brief)
		}
	}
}

func TestWorkerBriefDropsNoiseAndPathsIntoData(t *testing.T) {
	brief := WorkerBrief(Task{Body: oldCard})
	for _, banned := range []string{"make the invoice thing right pls", "Resolution method", "Work/acme/PROJECT.md", "Pending", "Launch policy", "Activity Log", "Created.", "# Fix billing"} {
		if strings.Contains(brief, banned) {
			t.Errorf("brief keeps %q:\n%s", banned, brief)
		}
	}
}

func TestWorkerBriefShowsPeopleCommentsNotFleetOnes(t *testing.T) {
	brief := WorkerBrief(Task{
		Body: "## Request\n\nDo it.\n",
		Comments: []Comment{
			{Author: "core", Text: "Cursor CLI-visible session is live. Chat id: abc. /Users/x/.local/bin/cursor-agent --resume abc"},
			{Author: "fleet", Text: "Execution outcome: completed. Execution ID: xyz"},
			{Author: "owner", Text: "The date format is still wrong.\nUse ISO 8601."},
		},
	})
	if !strings.Contains(brief, "- owner: The date format is still wrong.\n  Use ISO 8601.") {
		t.Fatalf("the person's comment is missing:\n%s", brief)
	}
	for _, banned := range []string{"Chat id", "/Users/", "Execution ID"} {
		if strings.Contains(brief, banned) {
			t.Errorf("brief leaks a system comment (%q):\n%s", banned, brief)
		}
	}
}

func TestWorkerBriefKeepsADeliverableSomeoneWrote(t *testing.T) {
	brief := WorkerBrief(Task{Body: "## Request\n\nDo it.\n\n## Deliverable\n\nA migration script in db/migrations.\n"})
	if !strings.Contains(brief, "A migration script in db/migrations.") {
		t.Fatalf("a written deliverable was dropped:\n%s", brief)
	}
}

func TestWorkerBriefIgnoresHeadingsInsideCodeFences(t *testing.T) {
	brief := WorkerBrief(Task{Body: "## Request\n\nExample:\n\n```md\n## Raw Input\nnot a section\n```\n"})
	if !strings.Contains(brief, "not a section") {
		t.Fatalf("a fenced heading split the request:\n%s", brief)
	}
}

func TestWorkerBriefOfACardWithoutSections(t *testing.T) {
	if got := WorkerBrief(Task{Body: "Just fix the typo in the README."}); got != "Just fix the typo in the README." {
		t.Fatalf("brief = %q", got)
	}
}

func TestSystemCommentAuthors(t *testing.T) {
	for _, name := range []string{"fleet", "Fleet", "core", " CORE "} {
		if !IsSystemCommentAuthor(name) {
			t.Errorf("%q should be a system author", name)
		}
	}
	for _, name := range []string{"owner", "codex", "manager", ""} {
		if IsSystemCommentAuthor(name) {
			t.Errorf("%q must not be a system author", name)
		}
	}
}
