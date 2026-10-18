package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/workfiles"
)

func TestMilestoneRoutesUseDashboardSurface(t *testing.T) {
	dash := &recordingDashboard{}
	mux := http.NewServeMux()
	registerMilestoneRoutes(mux, dash)

	t.Run("get_lists_project_milestones", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/projects/acme/milestones", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		if dash.milestonesCalls != 1 || dash.lastMilestoneProj != "acme" {
			t.Fatalf("Milestones calls = %d, project = %q", dash.milestonesCalls, dash.lastMilestoneProj)
		}
		if !strings.Contains(rec.Body.String(), "v1-launch") {
			t.Fatalf("body missing milestone: %s", rec.Body.String())
		}
	})

	t.Run("post_creates_a_milestone", func(t *testing.T) {
		body := `{"title":"v1 launch","version":"v1.0.0"}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/acme/milestones", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		if dash.createMilestoneCall != 1 || dash.lastMilestoneTitle != "v1 launch" || dash.lastMilestoneVer != "v1.0.0" {
			t.Fatalf("create = %d %q %q", dash.createMilestoneCall, dash.lastMilestoneTitle, dash.lastMilestoneVer)
		}
	})

	t.Run("patch_updates_one_milestone", func(t *testing.T) {
		body := `{"id":"v1-launch","state":"active"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/projects/acme/milestones", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		if dash.patchMilestoneCall != 1 || dash.lastMilestonePatch.id != "v1-launch" {
			t.Fatalf("patch call = %d, id = %q", dash.patchMilestoneCall, dash.lastMilestonePatch.id)
		}
		if dash.lastMilestonePatch.state == nil || *dash.lastMilestonePatch.state != "active" {
			t.Fatalf("state = %v", dash.lastMilestonePatch.state)
		}
	})

	t.Run("patch_without_id_is_rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/projects/acme/milestones", strings.NewReader(`{"state":"active"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("delete_removes_a_milestone_and_reports_cleared_tasks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/projects/acme/milestones", strings.NewReader(`{"id":"v1-launch"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"cleared":2`) {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		if dash.deleteMilestoneCall != 1 || dash.lastMilestonePatch.id != "v1-launch" {
			t.Fatalf("delete call = %d id = %q", dash.deleteMilestoneCall, dash.lastMilestonePatch.id)
		}
		bad := httptest.NewRecorder()
		mux.ServeHTTP(bad, httptest.NewRequest(http.MethodDelete, "/api/projects/acme/milestones", strings.NewReader(`{}`)))
		if bad.Code != http.StatusBadRequest {
			t.Fatalf("missing id status = %d, want 400", bad.Code)
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/projects/acme/milestones", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})
}

func TestMilestoneWriteReachesLiveBoardWithoutRestart(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "acme", "PROJECT.md"), "---\nid: acme\ntitle: Acme\nkind: standalone_repository\nstatus: draft\nreview_status: draft\nrepositories:\n  - acme\n---\n\n# Acme\n")
	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	if _, err := app.Surface().CreateMilestone("acme", "v1 launch", ""); err != nil {
		t.Fatal(err)
	}
	project := findProjectByID(app.State().Projects, "acme")
	if project == nil || len(project.Milestones) != 1 || project.Milestones[0].ID != "v1-launch" {
		t.Fatalf("board after create = %#v", project)
	}

	active := "active"
	if _, err := app.Surface().PatchMilestone("acme", "v1-launch", &active, nil, nil); err != nil {
		t.Fatal(err)
	}
	project = findProjectByID(app.State().Projects, "acme")
	if project.Milestones[0].State != "active" {
		t.Fatalf("board after patch = %#v", project.Milestones)
	}
}

func TestDeletingAMilestoneClearsItFromItsTasks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "acme", "PROJECT.md"), "---\nid: acme\ntitle: Acme\nkind: standalone_repository\nstatus: draft\nreview_status: draft\nrepositories:\n  - acme\n---\n\n# Acme\n")
	writeTestFile(t, filepath.Join(root, "Work", "acme", "tasks", "2026-08-01-a.md"), strings.Replace(testTaskForProjectMarkdown("ACME-1", "In it", "todo", "acme", "acme"), "id: work-test", "id: work-a", 1))
	writeTestFile(t, filepath.Join(root, "Work", "acme", "tasks", "2026-08-01-b.md"), strings.Replace(testTaskForProjectMarkdown("ACME-2", "Not in it", "todo", "acme", "acme"), "id: work-test", "id: work-b", 1))
	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	surface := app.Surface()
	m, err := surface.CreateMilestone("acme", "v1 launch", "")
	if err != nil {
		t.Fatal(err)
	}
	var assigned string
	for _, task := range app.State().Tasks {
		if task.Ref == "ACME-1" {
			assigned = task.Path
		}
	}
	if _, err := surface.PatchTask(tasklifecycle.TaskPatch{Path: assigned, Milestone: &m.ID}); err != nil {
		t.Fatal(err)
	}

	cleared, err := surface.DeleteMilestone("acme", m.ID)
	if err != nil || cleared != 1 {
		t.Fatalf("DeleteMilestone = %d, %v; want 1 task cleared", cleared, err)
	}
	for _, task := range app.State().Tasks {
		if task.Milestone != "" {
			t.Fatalf("task %s still points at %q", task.Ref, task.Milestone)
		}
	}
	if project := findProjectByID(app.State().Projects, "acme"); project == nil || len(project.Milestones) != 0 {
		t.Fatalf("board still lists the milestone: %#v", project)
	}
	if _, err := surface.DeleteMilestone("acme", m.ID); err == nil {
		t.Fatal("deleted the same milestone twice")
	}
}

func TestDeleteMilestoneFinishesASweepThatFailedHalfway(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "acme", "PROJECT.md"), "---\nid: acme\ntitle: Acme\nkind: standalone_repository\nstatus: draft\nreview_status: draft\nrepositories:\n  - acme\n---\n\n# Acme\n")
	writeTestFile(t, filepath.Join(root, "Work", "acme", "tasks", "2026-08-01-a.md"), strings.Replace(testTaskForProjectMarkdown("ACME-1", "In it", "todo", "acme", "acme"), "id: work-test", "id: work-a", 1))
	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	surface := app.Surface()
	m, _ := surface.CreateMilestone("acme", "v1", "")
	path := app.State().Tasks[0].Path
	if _, err := surface.PatchTask(tasklifecycle.TaskPatch{Path: path, Milestone: &m.ID}); err != nil {
		t.Fatal(err)
	}

	// Simulate a delete that removed the card entry but died before the sweep.
	if err := workfiles.DeleteMilestone(root, "acme", m.ID); err != nil {
		t.Fatal(err)
	}
	cleared, err := surface.DeleteMilestone("acme", m.ID)
	if err != nil || cleared != 1 {
		t.Fatalf("retry = %d, %v; want the dangling task cleared", cleared, err)
	}
	if got := app.State().Tasks[0].Milestone; got != "" {
		t.Fatalf("task still names %q", got)
	}
	if _, err := surface.DeleteMilestone("acme", m.ID); err == nil {
		t.Fatal("deleting a milestone nothing references reported success")
	}
}

func TestConcurrentAssignmentsNeverSurviveADelete(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "acme", "PROJECT.md"), "---\nid: acme\ntitle: Acme\nkind: standalone_repository\nstatus: draft\nreview_status: draft\nrepositories:\n  - acme\n---\n\n# Acme\n")
	const tasks = 8
	for i := 0; i < tasks; i++ {
		name := fmt.Sprintf("work-%d", i)
		writeTestFile(t, filepath.Join(root, "Work", "acme", "tasks", fmt.Sprintf("2026-08-01-%d.md", i)),
			strings.Replace(testTaskForProjectMarkdown(fmt.Sprintf("ACME-%d", i+1), "T", "todo", "acme", "acme"), "id: work-test", "id: "+name, 1))
	}
	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	surface := app.Surface()
	m, _ := surface.CreateMilestone("acme", "v1", "")

	var wg sync.WaitGroup
	for _, task := range app.State().Tasks {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			// Rejected once the milestone is gone; either outcome is fine.
			_, _ = surface.PatchTask(tasklifecycle.TaskPatch{Path: path, Milestone: &m.ID})
		}(task.Path)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = surface.DeleteMilestone("acme", m.ID)
	}()
	wg.Wait()

	// A delete finishes the sweep for anything it raced past.
	_, _ = surface.DeleteMilestone("acme", m.ID)
	for _, task := range app.State().Tasks {
		if task.Milestone != "" {
			t.Fatalf("task %s was left pointing at the deleted milestone %q", task.Ref, task.Milestone)
		}
	}
}
