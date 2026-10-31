package manager

import (
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
)

func TestValidateIntent(t *testing.T) {
	five := 5
	cases := []struct {
		name    string
		intent  Intent
		message string // empty means valid
	}{
		{"task ok", Intent{Kind: KindTask, Project: "p", Title: "t", Description: "d"}, ""},
		{"task no project", Intent{Kind: KindTask, Title: "t", Description: "d"}, "task intent requires project"},
		{"task no title", Intent{Kind: KindTask, Project: "p", Description: "d"}, "task intent requires title"},
		{"task no description", Intent{Kind: KindTask, Project: "p", Title: "t"}, "task intent requires description"},
		{"status ok", Intent{Kind: KindStatusChange, Ref: "R-1", Status: "done"}, ""},
		{"status missing", Intent{Kind: KindStatusChange, Ref: "R-1"}, "status_change requires ref and status"},
		{"comment ok", Intent{Kind: KindComment, Ref: "R-1", Comment: "hi"}, ""},
		{"comment blank", Intent{Kind: KindComment, Ref: "R-1", Comment: "  "}, "comment requires ref and comment"},
		{"board ok", Intent{Kind: KindBoardCommand, BoardAction: "show"}, ""},
		{"board missing", Intent{Kind: KindBoardCommand}, "board_command requires board_action"},
		{"assignee ok", Intent{Kind: KindAssigneeChange, Ref: "R-1", Assignee: "ann"}, ""},
		{"assignee missing", Intent{Kind: KindAssigneeChange, Ref: "R-1"}, "assignee_change requires ref and assignee"},
		{"priority ok", Intent{Kind: KindPriorityChange, Ref: "R-1", Priority: &five}, ""},
		{"priority missing", Intent{Kind: KindPriorityChange, Ref: "R-1"}, "priority_change requires ref and priority"},
		{"cancel ok", Intent{Kind: KindCancel, Ref: "R-1"}, ""},
		{"cancel missing", Intent{Kind: KindCancel}, "cancel requires ref"},
		{"question by query", Intent{Kind: KindQuestion, Query: "what"}, ""},
		{"question by ref", Intent{Kind: KindQuestion, Ref: "R-1"}, ""},
		{"question missing", Intent{Kind: KindQuestion, Query: " "}, "question requires ref or query"},
		{"unknown", Intent{Kind: "teleport"}, `unknown kind "teleport"`},
	}
	for _, c := range cases {
		err := ValidateIntent(c.intent)
		switch {
		case c.message == "" && err != nil:
			t.Errorf("%s: unexpected %v", c.name, err)
		case c.message != "" && (err == nil || err.Error() != c.message):
			t.Errorf("%s: err = %v, want %q", c.name, err, c.message)
		}
	}
}

func testView() BoardView {
	return BoardView{
		Workspaces: []board.Workspace{{ID: "mono", Title: "Mono Repo", Repositories: []string{"mono/api", "mono/web"}, Summary: "The monorepo"}},
		Projects: []board.Project{
			{ID: "mono-web", WorkspaceID: "mono", Title: "Web", Repositories: []string{"mono/web"}},
			{ID: "solo", Title: "Solo", Repositories: []string{"solo/tool"}},
		},
		Registry: board.RegistryInfo{Repositories: []board.RepositoryTechnology{{ID: "r1", Name: "tool", RelativePath: "solo/tool", EffectiveTags: []string{"go"}}}},
	}
}

func TestRepositoryMatches(t *testing.T) {
	view := testView()
	if got := repositoryMatches(view, "web", ""); len(got) != 2 || got[0] != "mono/web" {
		t.Fatalf("any project: %v", got)
	}
	if got := repositoryMatches(view, "web", "solo"); len(got) != 0 {
		t.Fatalf("project solo has no web: %v", got)
	}
	if got := uniqueFold(repositoryMatches(view, "tool", "")); len(got) != 2 {
		t.Fatalf("tool by path and by name: %v", got)
	}
	svc := &Service{Board: &fakeBoard{view: view}}
	if got, fail := svc.resolveRepository("api", "mono"); got != "mono/api" || fail != nil {
		t.Fatalf("resolve: %q %v", got, fail)
	}
	if _, fail := svc.resolveRepository("mono", ""); fail == nil || fail.Code != FailureAmbiguousRepository {
		t.Fatalf("ambiguous: %v", fail)
	}
	if got, fail := svc.resolveRepository("unknown", ""); got != "unknown" || fail != nil {
		t.Fatalf("unknown repos pass through: %q %v", got, fail)
	}
	if got, _ := svc.resolveRepository("  ", ""); got != "" {
		t.Fatalf("blank: %q", got)
	}
}

func TestProjectMatcher(t *testing.T) {
	m := &projectMatcher{s: &Service{}, phrase: "Web", needle: "web"}
	if err := m.consider("mono-web", "mono-web", "Web"); err != nil {
		t.Fatal(err)
	}
	if err := m.consider("website", "website", "Site"); err != nil {
		t.Fatal(err)
	}
	if err := m.consider(" ", "x"); err != nil {
		t.Fatal(err)
	}
	if err := m.consider("website", "website"); err != nil { // repeats are not listed twice
		t.Fatal(err)
	}
	if len(m.exact) != 1 || m.exact[0] != "mono-web" || len(m.fuzzy) != 1 || m.fuzzy[0] != "website" {
		t.Fatalf("exact %v fuzzy %v", m.exact, m.fuzzy)
	}

	svc := &Service{Board: &fakeBoard{view: testView()}}
	exact, fuzzy, err := svc.projectCandidates("web")
	if err != nil || len(exact) != 1 || exact[0] != "mono" || len(fuzzy) != 0 {
		t.Fatalf("a project title match files its workspace: exact %v fuzzy %v err %v", exact, fuzzy, err)
	}
	exact, fuzzy, _ = svc.projectCandidates("mono-w")
	if len(exact) != 0 || len(fuzzy) != 1 || fuzzy[0] != "mono" {
		t.Fatalf("exact %v fuzzy %v", exact, fuzzy)
	}
}

func TestBuildVocabularyFromBoard(t *testing.T) {
	v := BuildVocabulary(testView())
	if len(v.Workspaces) != 1 || len(v.Projects) != 2 || len(v.Repositories) != 1 {
		t.Fatalf("entities: %d %d %d", len(v.Workspaces), len(v.Projects), len(v.Repositories))
	}
	if v.Projects[0].Title != "Solo" || v.Projects[1].Title != "Web" {
		t.Fatalf("projects are sorted by title: %+v", v.Projects)
	}
	has := func(term string) bool {
		for _, t := range v.Terms {
			if strings.EqualFold(t, term) {
				return true
			}
		}
		return false
	}
	if !has("Mono Repo") || !has("tool") || !has("Go") {
		t.Fatalf("terms = %v", v.Terms)
	}
	seen := map[string]bool{}
	for _, term := range v.Terms {
		if seen[strings.ToLower(term)] {
			t.Fatalf("duplicate term %q", term)
		}
		seen[strings.ToLower(term)] = true
	}
	if len(v.Technologies) == 0 || v.Prompt == "" {
		t.Fatalf("technologies %v prompt %q", v.Technologies, v.Prompt)
	}
	if empty := BuildVocabulary(BoardView{}); len(empty.Terms) == 0 {
		t.Fatal("the fixed operational terms are always there")
	}
}

func TestTasksInViewAndFilters(t *testing.T) {
	tasks := []taskflow.Task{
		{Ref: "A-1", Status: taskflow.StatusBlocked, ProjectID: "p1", Milestone: "m1"},
		{Ref: "A-2", Status: taskflow.StatusTodo, BlockedReason: "waits", Project: "p2"},
		{Ref: "A-3", Status: "needs_review", WorkspaceID: "p1"},
		{Ref: "A-4", Status: taskflow.StatusDone, ProjectID: "p1"},
	}
	count := func(view string) int {
		got, ok := tasksInView(tasks, view)
		if !ok {
			t.Fatalf("view %s is known", view)
		}
		return len(got)
	}
	if count("all") != 4 || count("needs_attention") != 2 || count("blocked") != 1 || count("in_review") != 1 {
		t.Fatal("view sizes")
	}
	if _, ok := tasksInView(tasks, "nope"); ok {
		t.Fatal("unknown view")
	}
	if got := filterBoardTasks(tasks, BoardQuery{Project: "P1"}); len(got) != 3 {
		t.Fatalf("project: %d", len(got))
	}
	if got := filterBoardTasks(tasks, BoardQuery{Status: "DONE"}); len(got) != 1 || got[0].Ref != "A-4" {
		t.Fatalf("status: %v", got)
	}
	if got := filterBoardTasks(tasks, BoardQuery{Milestone: "m1"}); len(got) != 1 {
		t.Fatalf("milestone: %v", got)
	}
	if got := filterBoardTasks(tasks, BoardQuery{Milestone: "none"}); len(got) != 3 {
		t.Fatalf("no milestone: %v", got)
	}
}

func TestBoardPage(t *testing.T) {
	cases := []struct{ total, offset, limit, start, end int }{
		{10, 0, 0, 0, 10},
		{500, 0, 0, 0, defaultBoardLimit},
		{500, 0, 9999, 0, maxBoardLimit},
		{10, 4, 3, 4, 7},
		{10, 8, 5, 8, 10},
		{10, -1, 5, 10, 10},
		{10, 99, 5, 10, 10},
	}
	for _, c := range cases {
		if s, e := boardPage(c.total, c.offset, c.limit); s != c.start || e != c.end {
			t.Errorf("%+v: got %d..%d", c, s, e)
		}
	}
}
