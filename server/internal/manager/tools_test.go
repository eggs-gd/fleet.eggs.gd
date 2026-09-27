package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/health"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
)

type memTasks struct {
	tasks   []taskflow.Task
	created []taskflow.CreateTask
}

func (m *memTasks) Create(_ context.Context, input taskflow.CreateTask) (taskflow.Task, error) {
	m.created = append(m.created, input)
	task := taskflow.Task{
		Ref:          "CORE-" + itoa(len(m.tasks)+1),
		ID:           "CORE-" + itoa(len(m.tasks)+1),
		Locator:      "CORE-" + itoa(len(m.tasks)+1),
		Title:        input.Title,
		Summary:      input.Description,
		Body:         input.Description,
		Project:      input.Project,
		Status:       input.Status,
		Type:         input.Type,
		Assignee:     input.Assignee,
		DependsOn:    append([]string{}, input.DependsOn...),
		Priority:     input.Priority,
		Repositories: nil,
		SourceInbox:  input.SourceInbox,
	}
	if task.Status == "" {
		task.Status = taskflow.Status("backlog")
	}
	if input.Repository != "" {
		task.Repositories = []string{input.Repository}
	}
	m.tasks = append(m.tasks, task)
	return task, nil
}

func (m *memTasks) Get(_ context.Context, id string) (taskflow.Task, error) {
	for _, task := range m.tasks {
		if task.ID == id || task.Locator == id || task.Ref == id {
			return task, nil
		}
	}
	return taskflow.Task{}, Failure{Code: FailureNotFound, Message: "missing"}
}

func (m *memTasks) List(_ context.Context, filter taskflow.TaskFilter) ([]taskflow.Task, error) {
	var out []taskflow.Task
	for _, task := range m.tasks {
		if filter.Ref != "" && !strings.EqualFold(task.Ref, filter.Ref) {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		out = append(out, task)
	}
	return out, nil
}

func (m *memTasks) Patch(_ context.Context, id string, patch taskflow.PatchInput) (taskflow.Task, error) {
	for i, task := range m.tasks {
		if task.Locator != id && task.Ref != id && task.ID != id {
			continue
		}
		if patch.Status != "" {
			task.Status = patch.Status
		}
		if patch.Comment != "" {
			task.Comments = append(task.Comments, taskflow.Comment{Author: patch.CommentAuthor, Text: patch.Comment})
		}
		if patch.Project != "" {
			task.Project = patch.Project
		}
		if patch.Repository != "" {
			task.Repositories = []string{patch.Repository}
		}
		if patch.DependsOn != nil {
			task.DependsOn = append([]string{}, *patch.DependsOn...)
		}
		if patch.Body != nil {
			task.Body = *patch.Body
		}
		m.tasks[i] = task
		return task, nil
	}
	return taskflow.Task{}, Failure{Code: FailureNotFound, Message: "missing"}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func TestBoardTaskAndSuggestions(t *testing.T) {
	mem := &memTasks{tasks: []taskflow.Task{
		{Ref: "CORE-1", ID: "CORE-1", Locator: "CORE-1", Title: "Invoice status", Status: taskflow.StatusBlocked, Assignee: "claude", BlockedReason: "which states", Repositories: []string{"acme/web"}},
		{Ref: "CORE-2", ID: "CORE-2", Locator: "CORE-2", Title: "Other", Status: "todo", Assignee: "codex"},
	}}
	svc := NewService(mem, BoardView{})
	board := svc.ShowBoard(context.Background(), BoardQuery{View: "needs_attention"})
	if !board.OK {
		t.Fatalf("board: %+v", board.Failure)
	}
	detail := board.Result.Detail.(map[string]any)
	tasks := detail["tasks"].([]map[string]any)
	if len(tasks) != 1 || tasks[0]["ref"] != "CORE-1" {
		t.Fatalf("needs_attention = %#v", tasks)
	}
	missing := svc.Task(context.Background(), "CORE-9")
	if missing.OK || missing.Failure == nil || len(missing.Failure.Suggestions) == 0 {
		t.Fatalf("missing task = %+v", missing.Failure)
	}
}

func TestValidateResolveRouteSimilar(t *testing.T) {
	mem := &memTasks{tasks: []taskflow.Task{
		{Ref: "CORE-1", ID: "CORE-1", Locator: "CORE-1", Title: "Show invoice status", Status: taskflow.StatusDoing, Assignee: "claude", DependsOn: []string{"CORE-2"}},
		{Ref: "CORE-2", ID: "CORE-2", Locator: "CORE-2", Title: "List invoice states", Status: "todo", DependsOn: []string{"CORE-1"}},
	}}
	svc := NewService(mem, BoardView{Workspaces: []board.Workspace{
		{ID: "acme-web", Title: "Acme Web", Repositories: []string{"acme/web"}},
		{ID: "acme-api", Title: "Acme API"},
	}})

	before := len(mem.tasks)
	invalid := svc.Validate(context.Background(), Intent{Kind: KindTask, Project: "acme-web", Title: "Show invoice status", Description: "surface it", Repository: "missing/repo", Ref: "CORE-1", DependsOn: []string{"CORE-2"}})
	if invalid.OK || len(invalid.Failure.Suggestions) < 2 {
		t.Fatalf("validate = %+v", invalid.Failure)
	}
	if len(mem.tasks) != before {
		t.Fatal("validate wrote a task")
	}

	confident := svc.ResolveProject("Acme Web")
	verdict := confident.Result.Detail.(map[string]any)["verdict"]
	if verdict != "confident" {
		t.Fatalf("resolve exact = %#v", confident.Result.Detail)
	}
	ambiguous := svc.ResolveProject("acme")
	if ambiguous.Result.Detail.(map[string]any)["verdict"] != "ambiguous" {
		t.Fatalf("resolve fuzzy = %#v", ambiguous.Result.Detail)
	}
	none := svc.ResolveProject("nope")
	if none.Result.Detail.(map[string]any)["verdict"] != "none" {
		t.Fatalf("resolve none = %#v", none.Result.Detail)
	}

	similar := svc.Similar(context.Background(), "invoice status on the web")
	hits := similar.Result.Detail.([]map[string]any)
	if len(hits) == 0 || hits[0]["ref"] != "CORE-1" {
		t.Fatalf("similar = %#v", similar.Result.Detail)
	}

	explicit := svc.Route(context.Background(), Intent{Assignee: "claude", Type: "bug"})
	if explicit.Result.Detail.(map[string]any)["confident"] != true {
		t.Fatalf("explicit route = %#v", explicit.Result.Detail)
	}
	fallback := svc.Route(context.Background(), Intent{Type: "bug"})
	fallbackDetail := fallback.Result.Detail.(map[string]any)
	if fallbackDetail["confident"] != false || fallbackDetail["worker"] != "cursor" {
		t.Fatalf("busy preferred route = %#v", fallbackDetail)
	}
	root := t.TempDir()
	svc.DataRoot = root
	if err := os.MkdirAll(filepath.Join(root, "Work", "acme-web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Work", "acme-web", "PROJECT.md"), []byte("---\nid: acme-web\nassignment:\n  default_assignee: codex\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	override := svc.Route(context.Background(), Intent{Project: "acme-web", Type: "bug"})
	if override.Result.Detail.(map[string]any)["worker"] != "codex" {
		t.Fatalf("project override = %#v", override.Result.Detail)
	}
}

func TestInboxProjectAnswerReview(t *testing.T) {
	mem := &memTasks{}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "_registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_registry", "counters.json"), []byte(`{"next_work_ref":1,"next_inbox_ref":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(mem, BoardView{})
	svc.DataRoot = root

	captured := svc.Inbox(context.Background(), "capture", "look at billing later", "", "", "")
	if !captured.OK || captured.Result.Ref != "INBOX-1" {
		t.Fatalf("capture = %+v", captured)
	}
	promoted := svc.Inbox(context.Background(), "promote", "", "INBOX-1", "acme-web", "Billing reminder")
	if !promoted.OK || promoted.Result.Ref != "CORE-1" {
		t.Fatalf("promote = %+v", promoted)
	}
	raw, err := os.ReadFile(captured.Result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "promoted_to:\n  - CORE-1") || mem.created[0].SourceInbox != "INBOX-1" {
		t.Fatalf("promotion record = %s created %#v", raw, mem.created[0])
	}
	if mem.created[0].Acceptance != "" {
		t.Fatalf("the raw note must not be copied into the acceptance criteria: %#v", mem.created[0])
	}

	// One capture decomposes into a second task: promote again on the same
	// ref, both tasks stay linked, and manager_task exposes the link back.
	again := svc.Inbox(context.Background(), "promote", "", "INBOX-1", "acme-web", "Second billing task")
	if !again.OK || again.Result.Ref != "CORE-2" {
		t.Fatalf("second promote = %+v", again)
	}
	raw, _ = os.ReadFile(captured.Result.Path)
	if !strings.Contains(string(raw), "promoted_to:\n  - CORE-1\n  - CORE-2") {
		t.Fatalf("second task not linked onto the same capture:\n%s", raw)
	}
	if mem.tasks[1].SourceInbox != "INBOX-1" {
		t.Fatalf("created task did not carry source_inbox: %#v", mem.tasks[1])
	}

	shown := svc.Inbox(context.Background(), "show", "", "INBOX-1", "", "")
	if !shown.OK {
		t.Fatalf("show = %+v", shown)
	}
	detail := shown.Result.Detail.(map[string]any)
	if detail["body"] != "look at billing later" || fmt.Sprint(detail["promoted_to"]) != "[CORE-1 CORE-2]" {
		t.Fatalf("show detail = %#v", detail)
	}

	listed := svc.Inbox(context.Background(), "list", "", "", "", "")
	if !listed.OK {
		t.Fatalf("list = %+v", listed)
	}
	items := listed.Result.Detail.([]map[string]any)
	if len(items) != 1 || items[0]["status"] != "promoted" || items[0]["preview"] != "look at billing later" {
		t.Fatalf("list detail = %#v", items)
	}

	if resp := svc.Alias("acme-web", "billing"); resp.OK {
		t.Fatal("wrote a card for a project that has none")
	}
	cardPath := filepath.Join(root, "Work", "acme-web", "PROJECT.md")
	if err := os.MkdirAll(filepath.Dir(cardPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cardPath, []byte("---\nid: acme-web\ntitle: Acme Web\naliases: []\n---\n\n# Acme Web\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := svc.Alias("acme-web", "billing")
	second := svc.Alias("acme-web", "BILLING")
	if !first.OK || !second.OK || first.Result.Detail.(map[string]any)["wrote"] != true || second.Result.Detail.(map[string]any)["wrote"] != false {
		t.Fatalf("project writes = %+v / %+v", first, second)
	}
	card, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(card), "  - billing") != 1 || strings.Contains(string(card), "## Activity Log") || strings.Contains(string(card), "aliases: []") {
		t.Fatalf("project card = %s", card)
	}
	resolve := NewService(mem, BoardView{Workspaces: []board.Workspace{{ID: "acme-web", Title: "Acme Web"}}})
	resolve.DataRoot = root
	if got := resolve.ResolveProject("Billing").Result.Detail.(map[string]any); got["verdict"] != "confident" || got["project"] != "acme-web" {
		t.Fatalf("alias does not resolve: %#v", got)
	}

	mem.tasks = append(mem.tasks, taskflow.Task{Ref: "CORE-3", ID: "CORE-3", Locator: "CORE-3", Title: "Blocked", Status: taskflow.StatusBlocked})
	answered := svc.Answer(context.Background(), "CORE-3", "use paid and open", "")
	if !answered.OK || answered.Result.Status != "todo" {
		t.Fatalf("answer = %+v", answered.Result)
	}

	mem.tasks = append(mem.tasks, taskflow.Task{Ref: "CORE-4", ID: "CORE-4", Locator: "CORE-4", Title: "Review me", Status: "needs_review", Assignee: "claude"})
	accepted := svc.Review(context.Background(), "CORE-4", "accept", "")
	if !accepted.OK || accepted.Result.Status != "done" {
		t.Fatalf("accept = %+v", accepted.Result)
	}
	mem.tasks = append(mem.tasks, taskflow.Task{Ref: "CORE-5", ID: "CORE-5", Locator: "CORE-5", Title: "Again", Status: "needs_review"})
	rework := svc.Review(context.Background(), "CORE-5", "rework", "status is still hidden")
	if !rework.OK || rework.Result.Status != "needs_rework" {
		t.Fatalf("rework = %+v", rework.Result)
	}
	if !strings.Contains(mem.tasks[len(mem.tasks)-1].Comments[0].Text, "## Rework") {
		t.Fatalf("rework comment = %#v", mem.tasks[len(mem.tasks)-1].Comments)
	}
}

func TestEventsReadsAuditCursor(t *testing.T) {
	root := t.TempDir()
	if err := audit.AppendEvent(root, audit.Event{Type: "task.needs_attention", TaskRef: "CORE-1", Message: "which states"}); err != nil {
		t.Fatal(err)
	}
	if err := audit.AppendEvent(root, audit.Event{Type: "session.started", Message: "ignore"}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(&memTasks{}, BoardView{})
	svc.RuntimeRoot = root
	resp := svc.Events(0)
	if !resp.OK {
		t.Fatal(resp.Failure)
	}
	events := resp.Result.Detail.([]audit.Event)
	if len(events) != 1 || events[0].Type != "task.needs_attention" || events[0].ID == 0 {
		t.Fatalf("events = %#v", events)
	}
	later := svc.Events(events[0].ID)
	if len(later.Result.Detail.([]audit.Event)) != 0 {
		t.Fatalf("cursor = %#v", later.Result.Detail)
	}
}

func TestRouteRejectsUnknownWorkerAndFlagsBusyOverride(t *testing.T) {
	mem := &memTasks{tasks: []taskflow.Task{
		{Ref: "CORE-1", ID: "CORE-1", Locator: "CORE-1", Title: "Busy", Status: taskflow.StatusDoing, Assignee: "codex"},
	}}
	svc := NewService(mem, BoardView{})
	root := t.TempDir()
	svc.DataRoot = root
	if err := os.MkdirAll(filepath.Join(root, "Work", "acme"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Work", "acme", "PROJECT.md"), []byte("---\nid: acme\ndefault_assignee: codex\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	unknown := svc.Route(context.Background(), Intent{Assignee: "alex"})
	if unknown.OK || unknown.Failure.Code != FailureValidation || len(unknown.Failure.Suggestions) == 0 {
		t.Fatalf("unknown worker = %+v", unknown)
	}
	if err := os.MkdirAll(filepath.Join(root, "Fleet"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Fleet", "alex.md"), []byte("# Alex\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	human := svc.Route(context.Background(), Intent{Assignee: "alex"})
	if !human.OK || human.Result.Detail.(map[string]any)["worker"] != "alex" {
		t.Fatalf("human worker = %+v", human)
	}
	owner := svc.Route(context.Background(), Intent{Assignee: "unassigned", Project: "acme"})
	detail := owner.Result.Detail.(map[string]any)
	if detail["worker"] != "codex" || detail["confident"] != false {
		t.Fatalf("busy override = %#v", detail)
	}
}

func TestSimilarCountsRunesNotBytes(t *testing.T) {
	mem := &memTasks{tasks: []taskflow.Task{
		{Ref: "CORE-1", ID: "CORE-1", Locator: "CORE-1", Title: "рахунок на сплату", Status: "todo"},
		{Ref: "CORE-2", ID: "CORE-2", Locator: "CORE-2", Title: "зі та на до", Status: "todo"},
	}}
	svc := NewService(mem, BoardView{})
	hits := svc.Similar(context.Background(), "зі та на до").Result.Detail.([]map[string]any)
	if len(hits) != 0 {
		t.Fatalf("two-letter words counted as significant: %#v", hits)
	}
	hits = svc.Similar(context.Background(), "рахунок").Result.Detail.([]map[string]any)
	if len(hits) != 1 || hits[0]["ref"] != "CORE-1" {
		t.Fatalf("similar = %#v", hits)
	}
}

func TestMissingTaskKeepsSuggestionsForWrappedFailure(t *testing.T) {
	mem := &memTasks{tasks: []taskflow.Task{{Ref: "CORE-1", ID: "CORE-1", Locator: "CORE-1", Title: "One", Status: "todo"}}}
	svc := NewService(mem, BoardView{})
	wrapped := fmt.Errorf("patch: %w", Failure{Code: FailureNotFound, Message: "missing"})
	resp := svc.missingTask(context.Background(), "CORE-9", wrapped)
	if resp.OK || resp.Failure.Code != FailureNotFound || len(resp.Failure.Suggestions) == 0 {
		t.Fatalf("missing = %+v", resp.Failure)
	}
}

func TestEventsFailsWithoutRuntimeRootAndWarnsWhenAuditIsDegraded(t *testing.T) {
	svc := NewService(&memTasks{}, BoardView{})
	if resp := svc.Events(0); resp.OK || resp.Failure.Code != FailureProviderError {
		t.Fatalf("events without runtime root = %+v", resp)
	}
	svc.RuntimeRoot = t.TempDir()
	svc.Health = health.New()
	svc.Health.Fail("audit", fmt.Errorf("disk full"))
	resp := svc.Events(0)
	if !resp.OK || len(resp.Result.Warnings) != 1 || !strings.Contains(resp.Result.Warnings[0], "disk full") {
		t.Fatalf("degraded events = %+v", resp.Result)
	}
	svc.Health.OK("audit")
	if resp := svc.Events(0); len(resp.Result.Warnings) != 0 {
		t.Fatalf("recovered events still warn: %v", resp.Result.Warnings)
	}
}

func TestRouteWarnsAboutUnknownDefaultAssigneeInsteadOfFailing(t *testing.T) {
	svc := NewService(&memTasks{}, BoardView{})
	root := t.TempDir()
	svc.DataRoot = root
	if err := os.MkdirAll(filepath.Join(root, "Work", "acme"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Work", "acme", "PROJECT.md"), []byte("---\nid: acme\ndefault_assignee: codx\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resp := svc.Route(context.Background(), Intent{Project: "acme"})
	detail := resp.Result.Detail.(map[string]any)
	if !resp.OK || detail["worker"] != "claude" || detail["confident"] != false || len(resp.Result.Warnings) != 1 {
		t.Fatalf("route = %+v %#v", resp.Result, detail)
	}
}

func TestValidateReportsUnknownDependenciesAndUnreadableBoard(t *testing.T) {
	mem := &memTasks{tasks: []taskflow.Task{{Ref: "CORE-1", ID: "CORE-1", Locator: "CORE-1", Title: "One", Status: "todo"}}}
	svc := NewService(mem, BoardView{})
	resp := svc.Validate(context.Background(), Intent{Kind: KindTask, Title: "Two", Description: "d", Acceptance: "a", DependsOn: []string{"CORE-1", "CORE-9"}})
	if resp.OK || !strings.Contains(strings.Join(resp.Failure.Suggestions, "|"), "CORE-9 is not on the board") {
		t.Fatalf("validate = %+v", resp.Failure)
	}
	broken := NewService(nil, BoardView{})
	resp = broken.Validate(context.Background(), Intent{Kind: KindTask, Title: "Two", Description: "d", Acceptance: "a", DependsOn: []string{"CORE-1"}})
	if resp.OK || !strings.Contains(strings.Join(resp.Failure.Suggestions, "|"), "could not check depends_on") {
		t.Fatalf("unreadable board = %+v", resp.Failure)
	}
}

func TestBoardPagesFiltersAndSummarizes(t *testing.T) {
	mem := &memTasks{}
	for i := 1; i <= 5; i++ {
		project := "acme"
		if i > 3 {
			project = "other"
		}
		mem.tasks = append(mem.tasks, taskflow.Task{
			Ref: "CORE-" + itoa(i), ID: "CORE-" + itoa(i), Locator: "CORE-" + itoa(i), Title: "Task " + itoa(i),
			Status: "todo", Project: project, Body: strings.Repeat("word ", 100),
		})
	}
	svc := NewService(mem, BoardView{})
	first := svc.ShowBoard(context.Background(), BoardQuery{Project: "acme", Detail: "summary", Limit: 2})
	detail := first.Result.Detail.(map[string]any)
	cards := detail["tasks"].([]map[string]any)
	if detail["total"] != 3 || len(cards) != 2 || detail["next_offset"] != 2 {
		t.Fatalf("first page = %#v", detail)
	}
	if summary := cards[0]["summary"].(string); !strings.HasSuffix(summary, "…") || len([]rune(summary)) > summaryRunes+1 {
		t.Fatalf("summary = %q", summary)
	}
	second := svc.ShowBoard(context.Background(), BoardQuery{Project: "acme", Limit: 2, Offset: 2}).Result.Detail.(map[string]any)
	if len(second["tasks"].([]map[string]any)) != 1 || second["next_offset"] != nil {
		t.Fatalf("second page = %#v", second)
	}
	if _, has := second["tasks"].([]map[string]any)[0]["summary"]; has {
		t.Fatal("cards mode returned summaries")
	}
	if resp := svc.ShowBoard(context.Background(), BoardQuery{Detail: "everything"}); resp.OK {
		t.Fatal("unknown detail accepted")
	}
	full := svc.Task(context.Background(), "CORE-1").Result.Detail.(map[string]any)
	if full["body"] != mem.tasks[0].Body {
		t.Fatal("manager_task does not return the body")
	}
}

func TestUpdateChecksThenPatches(t *testing.T) {
	mem := &memTasks{tasks: []taskflow.Task{
		{Ref: "CORE-1", ID: "CORE-1", Locator: "CORE-1", Title: "One", Status: "todo", Project: "acme-web"},
		{Ref: "CORE-2", ID: "CORE-2", Locator: "CORE-2", Title: "Two", Status: "backlog", DependsOn: []string{"CORE-1"}},
	}}
	svc := NewService(mem, BoardView{Workspaces: []board.Workspace{
		{ID: "acme-web", Title: "Acme Web", Repositories: []string{"acme/web"}},
		{ID: "acme-api", Title: "Acme API"},
	}})
	ctx := context.Background()
	if resp := svc.Update(ctx, UpdateInput{Ref: "CORE-1"}); resp.OK {
		t.Fatal("empty update accepted")
	}
	cycle := []string{"CORE-2"}
	if resp := svc.Update(ctx, UpdateInput{Ref: "CORE-1", DependsOn: &cycle}); resp.OK || !strings.Contains(strings.Join(resp.Failure.Suggestions, "|"), "cycle") {
		t.Fatalf("cycle = %+v", resp.Failure)
	}
	if resp := svc.Update(ctx, UpdateInput{Ref: "CORE-1", Project: "acme"}); resp.OK {
		t.Fatalf("ambiguous project accepted: %+v", resp)
	}
	if resp := svc.Update(ctx, UpdateInput{Ref: "CORE-1", Repository: "nope/repo"}); resp.OK {
		t.Fatal("unknown repository accepted")
	}
	if resp := svc.Update(ctx, UpdateInput{Ref: "CORE-9", Body: ptr("x")}); resp.OK || resp.Failure.Code != FailureNotFound {
		t.Fatalf("missing task = %+v", resp)
	}
	none := []string{}
	resp := svc.Update(ctx, UpdateInput{Ref: "CORE-2", Project: "Acme API", DependsOn: &none, Body: ptr("new plan")})
	if !resp.OK {
		t.Fatalf("update = %+v", resp.Failure)
	}
	got := mem.tasks[1]
	if got.Project != "acme-api" || len(got.DependsOn) != 0 || got.Body != "new plan" {
		t.Fatalf("patched task = %#v", got)
	}
}

func ptr[T any](v T) *T { return &v }

// TestDirectTaskCreationLinksSourceInboxWithoutPromote proves the link back
// to the raw capture does not depend on the promote convenience action: a
// plain kind=task command with source_inbox set is enough. This is the path
// intake uses when it decomposes one capture into several tasks.
func TestDirectTaskCreationLinksSourceInboxWithoutPromote(t *testing.T) {
	mem := &memTasks{}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "_registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_registry", "counters.json"), []byte(`{"next_work_ref":1,"next_inbox_ref":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(mem, BoardView{})
	svc.DataRoot = root

	captured := svc.Inbox(context.Background(), "capture", "redesign the whole onboarding flow", "", "", "")
	if !captured.OK {
		t.Fatalf("capture = %+v", captured)
	}
	ref := captured.Result.Ref

	for _, title := range []string{"Design the empty state", "Wire up the welcome email"} {
		resp := svc.SubmitCommand(context.Background(), CommandRequest{Intent: Intent{
			Kind: KindTask, Project: "acme-web", Title: title, Description: title,
			Acceptance: "- [ ] done", SourceInbox: ref, Status: "backlog", Type: "feature",
		}})
		if !resp.OK {
			t.Fatalf("create %q = %+v", title, resp)
		}
	}
	raw, err := os.ReadFile(captured.Result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "promoted_to:\n  - CORE-1\n  - CORE-2") {
		t.Fatalf("both direct tasks must link back to %s:\n%s", ref, raw)
	}
	for i, want := range []string{"CORE-1", "CORE-2"} {
		if mem.tasks[i].SourceInbox != ref {
			t.Fatalf("task %s missing source_inbox: %#v", want, mem.tasks[i])
		}
	}
}

func TestProjectFactsAndDescribe(t *testing.T) {
	root := t.TempDir()
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("# Acme\n\nA tool that reconciles invoices.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "_registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	repos := `{"repositories": [{"relative_path": "acme", "summary": "No short prose summary was detected.", "summary_source": "readme-title", "readme_path": "` + readme + `"}]}`
	if err := os.WriteFile(filepath.Join(root, "_registry", "repositories.json"), []byte(repos), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "Work", "acme"), 0o755); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(root, "Work", "acme", "PROJECT.md")
	if err := os.WriteFile(card, []byte("---\nid: \"acme\"\nsummary_source: \"generated\"\n---\n\n# Acme\n\nNo short prose summary was detected.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewService(&memTasks{}, BoardView{Workspaces: []board.Workspace{
		{ID: "acme", Title: "Acme", Kind: "standalone_repository", Repositories: []string{"acme"}, Summary: "No short prose summary was detected.", SummarySource: "generated"},
	}})
	svc.DataRoot = root

	facts := svc.ProjectFacts("acme")
	if !facts.OK {
		t.Fatalf("facts = %+v", facts)
	}
	detail := facts.Result.Detail.(map[string]any)
	if detail["summary_source"] != "generated" {
		t.Fatalf("facts must show the project is still unconfirmed: %#v", detail)
	}
	repoFacts := detail["repositories"].([]map[string]any)
	if len(repoFacts) != 1 || repoFacts[0]["readme_excerpt"] != "# Acme\n\nA tool that reconciles invoices." {
		t.Fatalf("facts did not read the real README: %#v", repoFacts)
	}

	if resp := svc.Describe("acme", "   "); resp.OK {
		t.Fatal("wrote an empty description")
	}
	described := svc.Describe("acme", "A tool that reconciles invoices.")
	if !described.OK {
		t.Fatalf("describe = %+v", described)
	}
	raw, _ := os.ReadFile(card)
	if !strings.Contains(string(raw), "\n# Acme\n\nA tool that reconciles invoices.\n") {
		t.Fatalf("card not updated:\n%s", raw)
	}
	if !strings.Contains(string(raw), `summary_source: "confirmed"`) {
		t.Fatalf("summary_source not confirmed:\n%s", raw)
	}
}
