package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
)

func TestParseFastPathCommands(t *testing.T) {
	t.Parallel()

	priority := 2
	cases := []struct {
		name    string
		input   string
		want    *Intent
		failCode string
	}{
		{
			name:  "show board",
			input: "show board",
			want:  &Intent{Kind: KindBoardCommand, BoardAction: BoardShowBoard},
		},
		{
			name:  "show task ref",
			input: "show CORE-57",
			want:  &Intent{Kind: KindBoardCommand, BoardAction: BoardShowTask, Ref: "CORE-57"},
		},
		{
			name:  "show loose Ukrainian ref",
			input: "show задача 57",
			want:  &Intent{Kind: KindBoardCommand, BoardAction: BoardShowTask, Ref: "CORE-57"},
		},
		{
			name:  "move status",
			input: "move CORE-57 to todo",
			want:  &Intent{Kind: KindStatusChange, Ref: "CORE-57", Status: "todo"},
		},
		{
			name:  "move needs review",
			input: "move CORE-10 to needs review",
			want:  &Intent{Kind: KindStatusChange, Ref: "CORE-10", Status: "needs_review"},
		},
		{
			name:  "add comment",
			input: "add comment to CORE-57: please check the API shape",
			want:  &Intent{Kind: KindComment, Ref: "CORE-57", Comment: "please check the API shape", CommentAuthor: "alex"},
		},
		{
			name:  "assign",
			input: "assign CORE-57 to claude",
			want:  &Intent{Kind: KindAssigneeChange, Ref: "CORE-57", Assignee: "claude"},
		},
		{
			name:  "priority",
			input: "set priority of CORE-57 to 2",
			want:  &Intent{Kind: KindPriorityChange, Ref: "CORE-57", Priority: &priority},
		},
		{
			name:     "cancel without confirm",
			input:    "cancel CORE-57",
			failCode: FailureUnsafeCommand,
		},
		{
			name:  "cancel with confirm",
			input: "cancel CORE-57 confirm",
			want:  &Intent{Kind: KindCancel, Ref: "CORE-57", Confirm: true, Status: "archived"},
		},
		{
			name:  "non command falls through",
			input: "please add a Career Wizard vacancy history filter",
			want:  nil,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, fail := ParseFastPath(tc.input)
			if tc.failCode != "" {
				if fail == nil || fail.Code != tc.failCode {
					t.Fatalf("fail = %#v, want code %q", fail, tc.failCode)
				}
				return
			}
			if fail != nil {
				t.Fatalf("unexpected failure: %#v", fail)
			}
			if tc.want == nil {
				if got != nil {
					t.Fatalf("got %#v, want nil fallthrough", got)
				}
				return
			}
			if got == nil {
				t.Fatal("got nil intent")
			}
			if got.Kind != tc.want.Kind || got.BoardAction != tc.want.BoardAction || got.Ref != tc.want.Ref ||
				got.Status != tc.want.Status || got.Assignee != tc.want.Assignee || got.Comment != tc.want.Comment ||
				got.Confirm != tc.want.Confirm {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
			if tc.want.Priority != nil {
				if got.Priority == nil || *got.Priority != *tc.want.Priority {
					t.Fatalf("priority got %#v, want %d", got.Priority, *tc.want.Priority)
				}
			}
		})
	}
}

func TestIntentJSONSchemaParses(t *testing.T) {
	t.Parallel()
	raw, err := SchemaObject()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"CoreManagerIntent"`) {
		t.Fatalf("schema missing title: %s", encoded)
	}
}

func TestBuildVocabularyIsDynamicWithSummariesAndTechnologies(t *testing.T) {
	t.Parallel()
	vocab := BuildVocabulary(BoardView{
		Workspaces: []board.Workspace{{
			ID:      "eggs-gd-prod",
			Title:   "eGGs.gd Prod",
			Summary: "Product workspace for eGGs.gd apps and career tooling.",
			Technology: board.TechnologySummary{
				EffectiveTags: []string{"typescript", "svelte"},
			},
		}},
		Projects: []board.Project{{
			ID:          "career-wizard",
			Title:       "Career Wizard",
			WorkspaceID: "eggs-gd-prod",
			Summary:     "Personal career-materials workspace: CV, vacancy analysis, Telegram control.",
			Repositories: []string{"eGGs.gd.prod/career-wizard"},
			Technology: board.TechnologySummary{
				EffectiveTags: []string{"python", "docker"},
				DetectedTags:  []string{"go"},
			},
		}},
		Registry: board.RegistryInfo{
			Repositories: []board.RepositoryTechnology{
				{
					Name:         "career-wizard",
					RelativePath: "eGGs.gd.prod/career-wizard",
					EffectiveTags: []string{"python"},
				},
			},
		},
	})

	joined := strings.Join(vocab.Terms, " ")
	for _, term := range []string{"Core", "Codex", "Career Wizard", "eggs-gd-prod", "career-wizard", "Python", "TypeScript", "Svelte", "Docker", "Go"} {
		if !strings.Contains(joined, term) {
			t.Fatalf("vocabulary missing %q in %#v", term, vocab.Terms)
		}
	}

	if len(vocab.Projects) != 1 {
		t.Fatalf("projects = %#v", vocab.Projects)
	}
	project := vocab.Projects[0]
	if project.ID != "career-wizard" || project.Title != "Career Wizard" {
		t.Fatalf("project entity = %#v", project)
	}
	if !strings.Contains(project.Summary, "career-materials") {
		t.Fatalf("project summary = %q", project.Summary)
	}
	if !containsFold(project.Aliases, "eGGs.gd.prod/career-wizard") {
		t.Fatalf("project aliases = %#v", project.Aliases)
	}

	if len(vocab.Workspaces) != 1 || vocab.Workspaces[0].ID != "eggs-gd-prod" {
		t.Fatalf("workspaces = %#v", vocab.Workspaces)
	}
	if !strings.Contains(vocab.Workspaces[0].Summary, "Product workspace") {
		t.Fatalf("workspace summary = %q", vocab.Workspaces[0].Summary)
	}

	for _, tech := range []string{"Python", "TypeScript", "Svelte", "Docker", "Go"} {
		if !containsFold(vocab.Technologies, tech) {
			t.Fatalf("technologies missing %q in %#v", tech, vocab.Technologies)
		}
	}

	if !strings.Contains(vocab.Prompt, "generated dynamically") {
		t.Fatalf("prompt missing dynamic marker: %q", vocab.Prompt)
	}
	if !strings.Contains(vocab.Prompt, "Career Wizard") || !strings.Contains(vocab.Prompt, "career-materials") {
		t.Fatalf("prompt missing project explanation: %q", vocab.Prompt)
	}
	if !strings.Contains(vocab.Prompt, "Technologies in context:") {
		t.Fatalf("prompt missing technologies section: %q", vocab.Prompt)
	}
	// Hardcoded product names that belong in state should not be injected when absent.
	if strings.Contains(joined, "LMSense") || strings.Contains(joined, "JiveMax") {
		t.Fatalf("unexpected hardcoded project terms in %#v", vocab.Terms)
	}
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

type fakeBoard struct {
	view BoardView
}

func (f *fakeBoard) Board() BoardView {
	if f == nil {
		return BoardView{}
	}
	return f.view
}

// recordingProvider is a taskflow.TaskProvider that records writes for tests.
type recordingProvider struct {
	mu       sync.Mutex
	tasks    map[string]taskflow.Task
	created  []taskflow.CreateTask
	patched  []taskflow.Task
	comments []string
	nextID   int
}

func newRecordingProvider(seed ...taskflow.Task) *recordingProvider {
	p := &recordingProvider{tasks: map[string]taskflow.Task{}}
	for _, task := range seed {
		locator := task.Locator
		if locator == "" {
			locator = task.ID
		}
		task.Locator = locator
		p.tasks[locator] = task
	}
	return p
}

func (p *recordingProvider) Create(_ context.Context, input taskflow.CreateTask) (taskflow.Task, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.created = append(p.created, input)
	p.nextID++
	// Unique per call so a seeded fixture and a task created in the same
	// project during the same test never collide on locator/map key (they
	// used to both resolve to "Work/<project>/tasks/example.md").
	id := fmt.Sprintf("Work/%s/tasks/example-%d.md", input.Project, p.nextID)
	priority := input.Priority
	if priority == 0 {
		priority = 5
	}
	status := input.Status
	if status == "" {
		status = taskflow.StatusBacklog
	}
	task := taskflow.Task{
		Ref:          "CORE-100",
		ID:           id,
		Title:        input.Title,
		Status:       status,
		Priority:     priority,
		Project:      input.Project,
		Repositories: []string{input.Repository},
		Assignee:     input.Assignee,
		Locator:      id,
		Body:         input.Description,
	}
	p.tasks[id] = task
	return task, nil
}

func (p *recordingProvider) Get(_ context.Context, id string) (taskflow.Task, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	task, ok := p.tasks[id]
	if !ok {
		return taskflow.Task{}, fmt.Errorf("task %q not found", id)
	}
	return task, nil
}

func (p *recordingProvider) List(_ context.Context, filter taskflow.TaskFilter) ([]taskflow.Task, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]taskflow.Task, 0, len(p.tasks))
	for _, task := range p.tasks {
		if filter.Project != "" && task.Project != filter.Project && task.ProjectID != filter.Project {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		if filter.Assignee != "" && task.Assignee != filter.Assignee {
			continue
		}
		if filter.Ref != "" && !strings.EqualFold(task.Ref, filter.Ref) {
			continue
		}
		out = append(out, task)
	}
	return out, nil
}

func (p *recordingProvider) Update(_ context.Context, task taskflow.Task) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.patched = append(p.patched, task)
	locator := task.Locator
	if locator == "" {
		locator = task.ID
	}
	existing, ok := p.tasks[locator]
	if !ok {
		return fmt.Errorf("task %q not found", locator)
	}
	if task.Status != "" {
		existing.Status = task.Status
	}
	if task.Assignee != "" {
		existing.Assignee = task.Assignee
	}
	if task.Priority > 0 {
		existing.Priority = task.Priority
	}
	if task.Body != "" {
		existing.Body = task.Body
	}
	p.tasks[locator] = existing
	return nil
}

func (p *recordingProvider) AddComment(ctx context.Context, id string, text string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	author := taskflow.CommentAuthorFrom(ctx)
	if author == "" {
		author = "core"
	}
	p.comments = append(p.comments, author+": "+text)
	task, ok := p.tasks[id]
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}
	task.Comments = append(task.Comments, taskflow.Comment{Author: author, Text: text})
	p.tasks[id] = task
	return nil
}

func TestServiceDeterministicStatusChangeAndCreate(t *testing.T) {
	t.Parallel()
	locator := "Work/core-eggs-gd/tasks/example.md"
	provider := newRecordingProvider(taskflow.Task{
		Ref:      "CORE-57",
		Title:    "Example",
		Status:   taskflow.StatusBacklog,
		Project:  "core-eggs-gd",
		Locator:  locator,
		ID:       locator,
		Priority: 5,
	})
	svc := taskflow.NewService(provider)
	defer svc.Close()

	board := &fakeBoard{view: BoardView{
		Workspaces: []board.Workspace{{ID: "core-eggs-gd", Title: "Core"}},
	}}
	service := NewService(svc, board)

	move := service.SubmitText(context.Background(), TextRequest{Text: "move CORE-57 to todo"})
	if !move.OK {
		t.Fatalf("move failed: %#v", move.Failure)
	}
	if move.Path != PathDeterministic || move.Result == nil || move.Result.Action != "status_change" {
		t.Fatalf("unexpected move response: %#v", move)
	}
	if len(provider.patched) != 1 || provider.patched[0].Status != taskflow.StatusTodo {
		t.Fatalf("patched = %#v", provider.patched)
	}

	create := service.SubmitCommand(context.Background(), CommandRequest{Intent: Intent{
		Kind:        KindTask,
		Project:     "core-eggs-gd",
		Repository:  "core.eggs.gd",
		Title:       "Wire manager audio",
		Description: "Add STT wiring for the Manager API.",
		Status:      "backlog",
	}})
	if !create.OK {
		t.Fatalf("create failed: %#v", create.Failure)
	}
	if len(provider.created) != 1 {
		t.Fatalf("created = %#v", provider.created)
	}
	if provider.created[0].Title != "Wire manager audio" || provider.created[0].Project != "core-eggs-gd" {
		t.Fatalf("create request = %#v", provider.created[0])
	}
}

func TestManagerWritesGoThroughTaskService(t *testing.T) {
	t.Parallel()
	locator := "Work/core-eggs-gd/tasks/example.md"
	provider := newRecordingProvider(taskflow.Task{
		Ref:      "CORE-57",
		Title:    "Example",
		Status:   taskflow.StatusTodo,
		Project:  "core-eggs-gd",
		Locator:  locator,
		ID:       locator,
		Assignee: "unassigned",
		Priority: 5,
	})
	svc := taskflow.NewService(provider)
	defer svc.Close()
	service := NewService(svc, &fakeBoard{})

	comment := service.SubmitText(context.Background(), TextRequest{Text: "add comment to CORE-57: check the boundary"})
	if !comment.OK {
		t.Fatalf("comment failed: %#v", comment.Failure)
	}
	if len(provider.comments) != 1 || !strings.Contains(provider.comments[0], "alex:") {
		t.Fatalf("comments = %#v", provider.comments)
	}

	assign := service.SubmitText(context.Background(), TextRequest{Text: "assign CORE-57 to cursor"})
	if !assign.OK {
		t.Fatalf("assign failed: %#v", assign.Failure)
	}
	got, err := svc.Get(context.Background(), locator)
	if err != nil {
		t.Fatal(err)
	}
	if got.Assignee != "cursor" {
		t.Fatalf("assignee = %q, want cursor", got.Assignee)
	}

	show := service.SubmitText(context.Background(), TextRequest{Text: "show CORE-57"})
	if !show.OK || show.Result == nil || show.Result.Action != "show_task" {
		t.Fatalf("show failed: %#v", show)
	}
	if show.Result.Ref != "CORE-57" {
		t.Fatalf("show ref = %q", show.Result.Ref)
	}
}

func TestServiceFallsThroughToUnconfiguredLLM(t *testing.T) {
	t.Parallel()
	svc := taskflow.NewService(newRecordingProvider())
	defer svc.Close()
	service := NewService(svc, &fakeBoard{})
	resp := service.SubmitText(context.Background(), TextRequest{Text: "build a new vacancy ranking feature in Career Wizard"})
	if resp.OK {
		t.Fatal("expected failure when LLM is not configured")
	}
	if resp.Failure == nil || resp.Failure.Code != FailureLLMNotConfigured {
		t.Fatalf("failure = %#v", resp.Failure)
	}
}

func TestServiceAudioRequiresSTT(t *testing.T) {
	t.Parallel()
	svc := taskflow.NewService(newRecordingProvider())
	defer svc.Close()
	service := NewService(svc, &fakeBoard{})
	resp := service.SubmitAudio(context.Background(), []byte("fake-audio"), "audio/webm", "voice")
	if resp.OK || resp.Failure == nil || resp.Failure.Code != FailureSTTNotConfigured {
		t.Fatalf("response = %#v", resp)
	}
}

// trackingManagement records Contour 1 TaskManagement calls.
type trackingManagement struct {
	inner   TaskManagement
	mu      sync.Mutex
	creates int
	patches int
	lists   int
}

func (t *trackingManagement) Create(ctx context.Context, input taskflow.CreateTask) (taskflow.Task, error) {
	t.mu.Lock()
	t.creates++
	t.mu.Unlock()
	return t.inner.Create(ctx, input)
}

func (t *trackingManagement) Get(ctx context.Context, id string) (taskflow.Task, error) {
	return t.inner.Get(ctx, id)
}

func (t *trackingManagement) List(ctx context.Context, filter taskflow.TaskFilter) ([]taskflow.Task, error) {
	t.mu.Lock()
	t.lists++
	t.mu.Unlock()
	return t.inner.List(ctx, filter)
}

func (t *trackingManagement) Patch(ctx context.Context, id string, patch taskflow.PatchInput) (taskflow.Task, error) {
	t.mu.Lock()
	t.patches++
	t.mu.Unlock()
	return t.inner.Patch(ctx, id, patch)
}

func TestContour1CreateUpdateCommentStopAtTaskService(t *testing.T) {
	t.Parallel()
	locator := "Work/core-eggs-gd/tasks/example.md"
	provider := newRecordingProvider(taskflow.Task{
		Ref:      "CORE-110",
		Title:    "Contour isolation",
		Status:   taskflow.StatusBacklog,
		Project:  "core-eggs-gd",
		Locator:  locator,
		ID:       locator,
		Priority: 5,
	})
	svc := taskflow.NewService(provider)
	defer svc.Close()
	track := &trackingManagement{inner: svc}
	service := NewService(track, &fakeBoard{view: BoardView{
		Workspaces: []board.Workspace{{ID: "core-eggs-gd", Title: "Core"}},
	}})

	create := service.SubmitCommand(context.Background(), CommandRequest{Intent: Intent{
		Kind:        KindTask,
		Project:     "core-eggs-gd",
		Title:       "Isolate Contour 1",
		Description: "Manager must end at TaskService.",
		Status:      "backlog",
	}})
	if !create.OK {
		t.Fatalf("create: %#v", create.Failure)
	}

	update := service.SubmitText(context.Background(), TextRequest{Text: "move CORE-110 to todo"})
	if !update.OK {
		t.Fatalf("update: %#v", update.Failure)
	}

	comment := service.SubmitText(context.Background(), TextRequest{Text: "add comment to CORE-110: contour ends here"})
	if !comment.OK {
		t.Fatalf("comment: %#v", comment.Failure)
	}

	// Contour 1 returns operator confirmations; Claim is not on TaskManagement.
	track.mu.Lock()
	defer track.mu.Unlock()
	if track.creates != 1 {
		t.Fatalf("creates = %d, want 1", track.creates)
	}
	if track.patches < 2 {
		t.Fatalf("patches = %d, want >= 2 (status + comment)", track.patches)
	}
	if track.lists < 2 {
		t.Fatalf("lists = %d, want >= 2 (resolve refs for patch)", track.lists)
	}
	if create.Result == nil || create.Result.Action != "create_task" {
		t.Fatalf("create result = %#v", create.Result)
	}
	if update.Result == nil || update.Result.Action != "status_change" {
		t.Fatalf("update result = %#v", update.Result)
	}
	if comment.Result == nil || comment.Result.Action != "add_comment" {
		t.Fatalf("comment result = %#v", comment.Result)
	}
}

func TestTaskManagementOmitsExecutionMethods(t *testing.T) {
	t.Parallel()
	if ContourName != "task_management" {
		t.Fatalf("ContourName = %q", ContourName)
	}
	// Service field is TaskManagement, not full TaskService — compile-time
	// Contour 1 isolation (Claim / ReportExecution unavailable).
	var s Service
	_ = s.Tasks
}
