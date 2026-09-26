package plane

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/providerconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// fakePlane is a minimal in-memory stand-in for the Plane REST API,
// covering exactly the endpoints Provider calls: states, labels,
// work-items (list/get/create/update), and work-item comments
// (list/create). It exists so provider_test.go can exercise real HTTP
// request/response marshaling against Provider without live Plane
// credentials (none are available to this worker — see the architecture
// note for what that means for acceptance-criteria coverage).
type fakePlane struct {
	mu sync.Mutex

	states   []stateObj
	labels   []labelObj
	projects []projectObj
	items    map[string]*workItem
	comments map[string][]workComment
	nextID   int

	// forceStatus, if non-zero, makes the next matching request return that
	// HTTP status instead of the normal response — used to test the
	// rate-limit/auth failure path.
	forceStatusForPath map[string]int
}

func newFakePlane() *fakePlane {
	return &fakePlane{
		states: []stateObj{
			{ID: "s-backlog", Name: "backlog", Group: "backlog"},
			{ID: "s-todo", Name: "todo", Group: "unstarted"},
			{ID: "s-doing", Name: "doing", Group: "started"},
			{ID: "s-blocked", Name: "blocked", Group: "unstarted"},
			{ID: "s-needs_review", Name: "needs_review", Group: "started"},
			{ID: "s-needs_rework", Name: "needs_rework", Group: "unstarted"},
			{ID: "s-done", Name: "done", Group: "completed"},
			{ID: "s-archived", Name: "archived", Group: "cancelled"},
		},
		items:              map[string]*workItem{},
		comments:           map[string][]workComment{},
		forceStatusForPath: map[string]int{},
	}
}

func (f *fakePlane) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	// Workspace-level (not project-scoped) project list — used only by
	// ensureProjectID to resolve a project UUID when ProjectID is unset.
	mux.HandleFunc("/api/v1/workspaces/eggs_gd/projects/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		writeJSON(w, http.StatusOK, projectListResponse{Results: f.projects})
	})
	mux.HandleFunc("/api/v1/workspaces/eggs_gd/projects/proj-1/states/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		writeJSON(w, http.StatusOK, stateListResponse{Results: f.states})
	})
	mux.HandleFunc("/api/v1/workspaces/eggs_gd/projects/proj-1/labels/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method == http.MethodPost {
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.nextID++
			label := labelObj{ID: fmt.Sprintf("label-%d", f.nextID), Name: body.Name}
			f.labels = append(f.labels, label)
			writeJSON(w, http.StatusCreated, label)
			return
		}
		writeJSON(w, http.StatusOK, labelListResponse{Results: f.labels})
	})
	// One handler covers both the collection (list/create) and item-scoped
	// (get/update/comments) routes, dispatching on the path tail after
	// "work-items/" — http.ServeMux panics on duplicate pattern
	// registration, so this cannot be split into separate HandleFunc calls
	// per route the way the other, non-parameterized endpoints above are.
	mux.HandleFunc("/api/v1/workspaces/eggs_gd/projects/proj-1/work-items/", f.itemRouter(t))

	return httptest.NewServer(mux)
}

// itemRouter overrides the collection handler registered above (ServeMux
// keeps the last registration for an identical pattern) and dispatches both
// collection-level (list/create) and item-scoped (get/update/comments)
// requests by inspecting the path tail.
func (f *fakePlane) itemRouter(t *testing.T) http.HandlerFunc {
	const prefix = "/api/v1/workspaces/eggs_gd/projects/proj-1/work-items/"
	return func(w http.ResponseWriter, r *http.Request) {
		tail := strings.TrimPrefix(r.URL.Path, prefix)
		if tail == "" {
			f.handleCollection(w, r)
			return
		}
		parts := strings.Split(strings.Trim(tail, "/"), "/")
		id := parts[0]
		if len(parts) >= 2 && parts[1] == "comments" {
			f.handleComments(w, r, id)
			return
		}
		f.handleItem(w, r, id)
	}
}

func (f *fakePlane) handleCollection(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if code, ok := f.forceStatusForPath["list_or_create"]; ok {
		delete(f.forceStatusForPath, "list_or_create")
		http.Error(w, `{"error":"forced"}`, code)
		return
	}
	if r.Method == http.MethodPost {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		f.nextID++
		item := &workItem{ID: fmt.Sprintf("item-%d", f.nextID)}
		applyPayload(item, payload)
		f.items[item.ID] = item
		writeJSON(w, http.StatusCreated, item)
		return
	}
	externalID := r.URL.Query().Get("external_id")
	results := make([]workItem, 0, len(f.items))
	for _, item := range f.items {
		if externalID != "" && item.ExternalID != externalID {
			continue
		}
		results = append(results, *item)
	}
	writeJSON(w, http.StatusOK, workItemListResponse{Results: results})
}

func (f *fakePlane) handleItem(w http.ResponseWriter, r *http.Request, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if code, ok := f.forceStatusForPath["item:"+id]; ok {
		delete(f.forceStatusForPath, "item:"+id)
		http.Error(w, `{"error":"forced"}`, code)
		return
	}
	item, ok := f.items[id]
	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if r.Method == http.MethodPatch {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		applyPayload(item, payload)
	}
	writeJSON(w, http.StatusOK, item)
}

func (f *fakePlane) handleComments(w http.ResponseWriter, r *http.Request, itemID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Method == http.MethodPost {
		var payload struct {
			CommentHTML string `json:"comment_html"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		f.nextID++
		c := workComment{ID: fmt.Sprintf("comment-%d", f.nextID), CommentHTML: payload.CommentHTML, CreatedAt: fmt.Sprintf("2026-08-0%dT00:00:00Z", len(f.comments[itemID])+1)}
		f.comments[itemID] = append(f.comments[itemID], c)
		writeJSON(w, http.StatusCreated, c)
		return
	}
	writeJSON(w, http.StatusOK, commentListResponse{Results: f.comments[itemID]})
}

func applyPayload(item *workItem, payload map[string]any) {
	if name, ok := payload["name"].(string); ok {
		item.Name = name
	}
	if desc, ok := payload["description_stripped"].(string); ok {
		item.DescriptionStripped = desc
	}
	if priority, ok := payload["priority"].(string); ok {
		item.Priority = priority
	}
	if stateID, ok := payload["state"].(string); ok {
		item.State = &stateRef{ID: stateID}
	}
	if externalID, ok := payload["external_id"].(string); ok {
		item.ExternalID = externalID
	}
	if externalSourceValue, ok := payload["external_source"].(string); ok {
		item.ExternalSource = externalSourceValue
	}
	if labels, ok := payload["labels"].([]any); ok {
		item.Labels = make([]string, 0, len(labels))
		for _, l := range labels {
			if s, ok := l.(string); ok {
				item.Labels = append(item.Labels, s)
			}
		}
	}
	item.CreatedAt = "2026-08-03T00:00:00Z"
	item.UpdatedAt = "2026-08-03T00:00:00Z"
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func newTestProvider(t *testing.T, baseURL string) *Provider {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "_registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	counters := `{"work_ref_prefix":"CORE","next_work_ref":100,"inbox_ref_prefix":"INBOX","next_inbox_ref":1,"life_ref_prefix":"LIFE","next_life_ref":1,"notes":[]}`
	if err := os.WriteFile(filepath.Join(root, "_registry", "counters.json"), []byte(counters), 0o644); err != nil {
		t.Fatal(err)
	}
	writeProjectCard(t, root, "CORE")

	settings := providerconfig.PlaneSettings{
		Workspace:   "eggs_gd",
		BaseURL:     baseURL,
		ProjectID:   "proj-1",
		CoreProject: "core-eggs-gd",
		Repository:  "core.eggs.gd",
	}
	return New(settings, "test-token", root)
}

func TestProvider_CreateListLoadMutateClaimHappyPath(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)

	created, err := provider.CreateFromRequest(tasklifecycle.TaskCreateRequest{
		Title:    "Integrate Plane",
		Request:  "Add Plane as a second task provider.",
		Project:  "core-eggs-gd",
		Assignee: "claude",
		Type:     "feature",
	})
	if err != nil {
		t.Fatalf("CreateFromRequest: %v", err)
	}
	if created.Status != "backlog" {
		t.Fatalf("expected default status backlog, got %q", created.Status)
	}
	if !strings.HasPrefix(created.Ref, "CORE-") {
		t.Fatalf("expected CORE-* ref, got %q", created.Ref)
	}
	if created.Assignee != "claude" {
		t.Fatalf("expected assignee claude, got %q", created.Assignee)
	}
	if created.Type != "feature" {
		t.Fatalf("expected type feature, got %q", created.Type)
	}
	if len(created.Repositories) != 1 || created.Repositories[0] != "core.eggs.gd" {
		t.Fatalf("expected configured repository, got %v", created.Repositories)
	}

	listed, err := provider.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 1 || listed[0].Ref != created.Ref {
		t.Fatalf("expected List to return the created task, got %+v", listed)
	}

	loaded, err := provider.Load(created.RelativePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Ref != created.Ref {
		t.Fatalf("Load returned a different task: %+v", loaded)
	}

	// backlog -> todo
	moved, err := provider.Mutate(tasklifecycle.TaskPatch{Path: created.RelativePath, Status: "todo"})
	if err != nil {
		t.Fatalf("Mutate backlog->todo: %v", err)
	}
	if moved.Status != "todo" {
		t.Fatalf("expected status todo, got %q", moved.Status)
	}

	// todo -> doing via Claim
	claimed, err := provider.Claim(created.RelativePath)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed.Status != "doing" {
		t.Fatalf("expected status doing after claim, got %q", claimed.Status)
	}

	// A second claim must be rejected: task already left the pickup status.
	if _, err := provider.Claim(created.RelativePath); err == nil {
		t.Fatal("expected second Claim to fail")
	} else if !strings.Contains(err.Error(), tasklifecycle.ErrTaskAlreadyClaimed.Error()) {
		t.Fatalf("expected ErrTaskAlreadyClaimed, got %v", err)
	}

	// doing -> blocked with a review comment.
	blocked, err := provider.Mutate(tasklifecycle.TaskPatch{
		Path:          created.RelativePath,
		Status:        "blocked",
		Comment:       "waiting on Plane API token",
		CommentAuthor: "claude",
	})
	if err != nil {
		t.Fatalf("Mutate doing->blocked: %v", err)
	}
	if blocked.Status != "blocked" {
		t.Fatalf("expected status blocked, got %q", blocked.Status)
	}
	if len(blocked.Comments) != 1 || blocked.Comments[0].Author != "claude" || blocked.Comments[0].Text != "waiting on Plane API token" {
		t.Fatalf("expected round-tripped comment, got %+v", blocked.Comments)
	}
	if blocked.BlockedReason != "waiting on Plane API token" {
		t.Fatalf("expected derived blocked reason, got %q", blocked.BlockedReason)
	}

	// List() must surface the same blocked_reason/comment for a blocked
	// task (the one status where List fetches comments — see List's doc
	// comment on the Plane rate-limit tradeoff).
	relisted, err := provider.List()
	if err != nil {
		t.Fatalf("List after blocking: %v", err)
	}
	if len(relisted) != 1 || relisted[0].Status != "blocked" || relisted[0].BlockedReason == "" {
		t.Fatalf("expected relisted blocked task with a reason, got %+v", relisted)
	}
}

func TestProvider_RejectsIllegalStatusTransition(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)

	created, err := provider.CreateFromRequest(tasklifecycle.TaskCreateRequest{
		Title:   "Illegal transition test",
		Request: "n/a",
		Project: "core-eggs-gd",
	})
	if err != nil {
		t.Fatalf("CreateFromRequest: %v", err)
	}

	// backlog -> done is not a legal transition (see status_transition.go).
	_, err = provider.Mutate(tasklifecycle.TaskPatch{Path: created.RelativePath, Status: "done"})
	if err == nil {
		t.Fatal("expected illegal transition to be rejected")
	}
	if !strings.Contains(err.Error(), tasklifecycle.ErrTaskTransitionRejected.Error()) {
		t.Fatalf("expected ErrTaskTransitionRejected, got %v", err)
	}
}

func TestProvider_RateLimitedFailureMode(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)

	fake.mu.Lock()
	fake.forceStatusForPath["list_or_create"] = http.StatusTooManyRequests
	fake.mu.Unlock()

	_, err := provider.List()
	if err == nil {
		t.Fatal("expected List to fail when Plane returns 429")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *plane.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", apiErr.StatusCode)
	}
	if apiErr.Classified == nil || apiErr.Classified.Kind != "rate_limited" {
		t.Fatalf("expected classified rate_limited provider error, got %+v", apiErr.Classified)
	}
}

// newTestProviderNoProjectID builds a provider like newTestProvider but with
// ProjectID left empty, so ensureProjectID must resolve it via
// fakePlane's workspace-level project list on first use.
func newTestProviderNoProjectID(t *testing.T, baseURL string) *Provider {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "_registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	counters := `{"work_ref_prefix":"CORE","next_work_ref":100,"inbox_ref_prefix":"INBOX","next_inbox_ref":1,"life_ref_prefix":"LIFE","next_life_ref":1,"notes":[]}`
	if err := os.WriteFile(filepath.Join(root, "_registry", "counters.json"), []byte(counters), 0o644); err != nil {
		t.Fatal(err)
	}
	writeProjectCard(t, root, "CORE")

	settings := providerconfig.PlaneSettings{
		Workspace:   "eggs_gd",
		BaseURL:     baseURL,
		CoreProject: "core-eggs-gd",
		Repository:  "core.eggs.gd",
	}
	return New(settings, "test-token", root)
}

func TestProvider_ResolvesProjectIDByCoreProjectMatch(t *testing.T) {
	fake := newFakePlane()
	fake.projects = []projectObj{
		{ID: "other-proj", Identifier: "OTH", Name: "some-other-repo"},
		{ID: "proj-1", Identifier: "COR", Name: "core-eggs-gd"},
	}
	server := fake.server(t)
	defer server.Close()
	provider := newTestProviderNoProjectID(t, server.URL)

	if _, err := provider.List(); err != nil {
		t.Fatalf("List with unresolved ProjectID: %v", err)
	}
	if provider.client.projectID != "proj-1" {
		t.Fatalf("expected ensureProjectID to resolve to proj-1, got %q", provider.client.projectID)
	}
	if provider.settings.ProjectID != "proj-1" {
		t.Fatalf("expected settings.ProjectID kept in sync, got %q", provider.settings.ProjectID)
	}
}

func TestProvider_ResolveProjectIDNoMatchReturnsError(t *testing.T) {
	fake := newFakePlane()
	fake.projects = []projectObj{
		{ID: "other-proj", Identifier: "OTH", Name: "some-other-repo"},
	}
	server := fake.server(t)
	defer server.Close()
	provider := newTestProviderNoProjectID(t, server.URL)

	_, err := provider.List()
	if err == nil {
		t.Fatal("expected List to fail when no workspace project matches coreProject")
	}
	if !strings.Contains(err.Error(), "no project in workspace") {
		t.Fatalf("expected a clear no-match error, got: %v", err)
	}
}

func TestLabelRefsUnmarshalsBothShapes(t *testing.T) {
	var bare labelRefs
	if err := json.Unmarshal([]byte(`["label-1","label-2"]`), &bare); err != nil {
		t.Fatalf("bare id array: %v", err)
	}
	if !strings.Contains(strings.Join(bare, ","), "label-1") || len(bare) != 2 {
		t.Fatalf("expected 2 bare ids, got %v", bare)
	}

	var nested labelRefs
	if err := json.Unmarshal([]byte(`[{"id":"label-1","name":"core:assignee:claude","color":"#fff"},{"id":"label-2","name":"core:type:feature"}]`), &nested); err != nil {
		t.Fatalf("nested label objects (real Plane get_work_item shape): %v", err)
	}
	if len(nested) != 2 || nested[0] != "label-1" || nested[1] != "label-2" {
		t.Fatalf("expected ids extracted from nested objects, got %v", nested)
	}
}

func TestWorkItemIDFromLocator(t *testing.T) {
	id, err := workItemIDFromLocator("plane://eggs_gd/proj-1/item-42")
	if err != nil {
		t.Fatal(err)
	}
	if id != "item-42" {
		t.Fatalf("expected item-42, got %q", id)
	}
	if _, err := workItemIDFromLocator(""); err == nil {
		t.Fatal("expected empty locator to error")
	}
}

// writeProjectCard gives the mapped Core project a card with a ref tag, which is
// where the Plane provider takes the prefix of new refs from.
func writeProjectCard(t *testing.T, root, tag string) {
	t.Helper()
	dir := filepath.Join(root, "Work", "core-eggs-gd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	card := "---\nid: core-eggs-gd\ntag: \"" + tag + "\"\n---\n\n# Core\n"
	if err := os.WriteFile(filepath.Join(dir, "PROJECT.md"), []byte(card), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProvider_CreateTakesTheTagOfTheMappedCoreProject(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)
	writeProjectCard(t, provider.coreRoot, "PLNE")

	created, err := provider.CreateFromRequest(tasklifecycle.TaskCreateRequest{
		Title: "Tagged", Request: "r", Project: "core-eggs-gd", Assignee: "claude", Type: "feature",
	})
	if err != nil {
		t.Fatalf("CreateFromRequest: %v", err)
	}
	if created.Ref != "PLNE-1" {
		t.Fatalf("ref = %q, want PLNE-1", created.Ref)
	}
}

func TestProvider_CreateFailsWhenTheMappedProjectHasNoTag(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)
	if err := os.WriteFile(filepath.Join(provider.coreRoot, "Work", "core-eggs-gd", "PROJECT.md"), []byte("---\nid: core-eggs-gd\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := provider.CreateFromRequest(tasklifecycle.TaskCreateRequest{
		Title: "Untagged", Request: "r", Project: "core-eggs-gd", Assignee: "claude", Type: "feature",
	})
	if err == nil || !strings.Contains(err.Error(), "no tag yet") {
		t.Fatalf("err = %v", err)
	}
	if len(fake.items) != 0 {
		t.Fatal("a Plane work item was created for a task with no ref")
	}
}
