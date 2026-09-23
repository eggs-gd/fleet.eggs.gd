package plane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// stateRef is a work item's workflow state. Plane's documented list-issues
// response nests a full object ({id, name, group}); the create/update
// response examples show only a bare state UUID. Both are handled so the
// adapter does not depend on which shape a given Plane deployment/endpoint
// actually returns.
type stateRef struct {
	ID    string
	Name  string
	Group string
}

func (s *stateRef) UnmarshalJSON(data []byte) error {
	var obj struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Group string `json:"group"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && (obj.ID != "" || obj.Name != "") {
		s.ID, s.Name, s.Group = obj.ID, obj.Name, obj.Group
		return nil
	}
	var id string
	if err := json.Unmarshal(data, &id); err == nil {
		s.ID = id
		return nil
	}
	return nil
}

// labelRefs is a work item's label list. Plane's list-issues response uses
// bare label ID strings; the single-item get_work_item response instead
// nests full label objects ({id, name, color, ...}). Both are handled here
// the same way stateRef already handles state, so callers only ever see
// plain label IDs regardless of which shape a given endpoint returned.
type labelRefs []string

func (l *labelRefs) UnmarshalJSON(data []byte) error {
	var ids []string
	if err := json.Unmarshal(data, &ids); err == nil {
		*l = ids
		return nil
	}
	var objs []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &objs); err != nil {
		return err
	}
	ids = make([]string, 0, len(objs))
	for _, obj := range objs {
		ids = append(ids, obj.ID)
	}
	*l = ids
	return nil
}

type workItem struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	DescriptionHTML     string    `json:"description_html,omitempty"`
	DescriptionStripped string    `json:"description_stripped,omitempty"`
	Priority            string    `json:"priority,omitempty"`
	State               *stateRef `json:"state,omitempty"`
	Assignees           []string  `json:"assignees,omitempty"`
	Labels              labelRefs `json:"labels,omitempty"`
	ExternalID          string    `json:"external_id,omitempty"`
	ExternalSource      string    `json:"external_source,omitempty"`
	SequenceID          int       `json:"sequence_id,omitempty"`
	CreatedAt           string    `json:"created_at,omitempty"`
	UpdatedAt           string    `json:"updated_at,omitempty"`
}

type workItemListResponse struct {
	Results         []workItem `json:"results"`
	NextCursor      string     `json:"next_cursor"`
	NextPageResults bool       `json:"next_page_results"`
}

type stateObj struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
}

type stateListResponse struct {
	Results []stateObj `json:"results"`
}

type labelObj struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type labelListResponse struct {
	Results []labelObj `json:"results"`
}

// projectObj is one entry from Plane's workspace-level project list — used
// only to resolve a project's UUID by name/identifier when Core's config
// names a Core project instead of pinning a raw UUID (Provider.ensureProjectID).
type projectObj struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

type projectListResponse struct {
	Results []projectObj `json:"results"`
}

type workComment struct {
	ID          string `json:"id"`
	CommentHTML string `json:"comment_html"`
	CreatedAt   string `json:"created_at"`
}

type commentListResponse struct {
	Results []workComment `json:"results"`
}

func (c *Client) ListWorkItems(ctx context.Context) ([]workItem, error) {
	var all []workItem
	cursor := ""
	for {
		query := url.Values{"per_page": {"100"}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		var page workItemListResponse
		if err := c.do(ctx, http.MethodGet, c.projectPath("work-items/"), query, nil, &page, "list_work_items"); err != nil {
			return nil, err
		}
		all = append(all, page.Results...)
		if !page.NextPageResults || page.NextCursor == "" || page.NextCursor == cursor {
			break
		}
		cursor = page.NextCursor
	}
	return all, nil
}

// FindWorkItemByExternalID looks up the (at most one) work item Core created
// for a given CORE-N ref, via Plane's external_id/external_source filter.
func (c *Client) FindWorkItemByExternalID(ctx context.Context, coreRef string) (*workItem, error) {
	query := url.Values{"external_id": {coreRef}, "external_source": {externalSource}, "per_page": {"1"}}
	var page workItemListResponse
	if err := c.do(ctx, http.MethodGet, c.projectPath("work-items/"), query, nil, &page, "find_work_item"); err != nil {
		return nil, err
	}
	if len(page.Results) == 0 {
		return nil, nil
	}
	return &page.Results[0], nil
}

func (c *Client) GetWorkItem(ctx context.Context, id string) (*workItem, error) {
	var item workItem
	if err := c.do(ctx, http.MethodGet, c.projectPath("work-items/"+url.PathEscape(id)+"/"), nil, nil, &item, "get_work_item"); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *Client) CreateWorkItem(ctx context.Context, payload map[string]any) (*workItem, error) {
	var item workItem
	if err := c.do(ctx, http.MethodPost, c.projectPath("work-items/"), nil, payload, &item, "create_work_item"); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *Client) UpdateWorkItem(ctx context.Context, id string, payload map[string]any) (*workItem, error) {
	var item workItem
	if err := c.do(ctx, http.MethodPatch, c.projectPath("work-items/"+url.PathEscape(id)+"/"), nil, payload, &item, "update_work_item"); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *Client) AddComment(ctx context.Context, workItemID string, commentHTML string) error {
	payload := map[string]any{"comment_html": commentHTML, "external_source": externalSource}
	return c.do(ctx, http.MethodPost, c.projectPath("work-items/"+url.PathEscape(workItemID)+"/comments/"), nil, payload, nil, "add_comment")
}

func (c *Client) ListComments(ctx context.Context, workItemID string) ([]workComment, error) {
	var page commentListResponse
	if err := c.do(ctx, http.MethodGet, c.projectPath("work-items/"+url.PathEscape(workItemID)+"/comments/"), url.Values{"per_page": {"100"}}, nil, &page, "list_comments"); err != nil {
		return nil, err
	}
	return page.Results, nil
}

// ListProjects returns every project in the configured workspace. Used only
// to resolve a project UUID by name/identifier (Provider.ensureProjectID) —
// not part of the per-task request path, so it is not paginated beyond one
// page of 100 (a workspace with more projects than that is not this repo's
// scale problem yet).
func (c *Client) ListProjects(ctx context.Context) ([]projectObj, error) {
	var page projectListResponse
	if err := c.do(ctx, http.MethodGet, c.workspacePath("projects/"), url.Values{"per_page": {"100"}}, nil, &page, "list_projects"); err != nil {
		return nil, err
	}
	return page.Results, nil
}

func (c *Client) ListStates(ctx context.Context) ([]stateObj, error) {
	var page stateListResponse
	if err := c.do(ctx, http.MethodGet, c.projectPath("states/"), url.Values{"per_page": {"100"}}, nil, &page, "list_states"); err != nil {
		return nil, err
	}
	return page.Results, nil
}

func (c *Client) ListLabels(ctx context.Context) ([]labelObj, error) {
	var page labelListResponse
	if err := c.do(ctx, http.MethodGet, c.projectPath("labels/"), url.Values{"per_page": {"100"}}, nil, &page, "list_labels"); err != nil {
		return nil, err
	}
	return page.Results, nil
}

func (c *Client) CreateLabel(ctx context.Context, name string) (*labelObj, error) {
	var label labelObj
	if err := c.do(ctx, http.MethodPost, c.projectPath("labels/"), nil, map[string]any{"name": name, "color": "#6366f1"}, &label, "create_label"); err != nil {
		return nil, err
	}
	return &label, nil
}
