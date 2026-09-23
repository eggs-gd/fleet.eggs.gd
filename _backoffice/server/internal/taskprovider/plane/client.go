// Package plane is the Plane-backed TaskProvider (CORE-106).
//
// It owns every Plane-coupled concern for tasks:
//
//   - REST API calls (client.go / entities.go)
//   - Plane work-item IDs and Core CORE-N ref mapping (external_id)
//   - state/status and priority mapping (mapping.go)
//   - comments, labels (assignee/type), pagination, API errors
//   - provider-specific metadata placement (see _docs/PLANE_TASK_PROVIDER.md)
//   - change observation: ObservePoll and ObserveWebhook →
//     TaskEvent{Before,After,Source} (CORE-111)
//
// *Provider satisfies taskprovider.Provider for Runtime bootstrap/sync.
// Provider.Flow() satisfies taskflow.TaskProvider for TaskService — the same
// application contract Markdown exposes. Generic manager/launcher/finalizer
// code must not import this package or branch on Plane.
//
// client.go is the thin HTTP layer: request/response JSON, auth, pagination,
// and error classification. provider.go and mapping.go own what a Plane work
// item means to Core.
package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

// externalSource tags every Core-created work item/comment/label so Core can
// find its own records via Plane's external_id/external_source filter
// (list-issues supports external_id + external_source query params) without
// colliding with issues created by other integrations.
const externalSource = "core"

const defaultBaseURL = "https://api.plane.so"

// ClientConfig configures a low-level Plane REST client scoped to one
// workspace/project — the shape Core needs, since one Provider maps to
// exactly one Core project (see providerconfig.PlaneSettings.CoreProject).
type ClientConfig struct {
	BaseURL    string
	Workspace  string
	ProjectID  string
	APIToken   string
	HTTPClient *http.Client
}

// Client is the low-level Plane REST client. It knows Plane's URL shape,
// auth header, and pagination/error conventions; it does not know what a
// Core Task is.
type Client struct {
	baseURL   string
	workspace string
	projectID string
	token     string
	http      *http.Client
}

func NewClient(cfg ClientConfig) *Client {
	baseURL := strings.TrimSuffix(firstNonEmpty(cfg.BaseURL, defaultBaseURL), "/")
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		baseURL:   baseURL,
		workspace: cfg.Workspace,
		projectID: cfg.ProjectID,
		token:     cfg.APIToken,
		http:      httpClient,
	}
}

func (c *Client) projectPath(suffix string) string {
	return fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/%s",
		c.baseURL, url.PathEscape(c.workspace), url.PathEscape(c.projectID), suffix)
}

// workspacePath builds a workspace-scoped URL (not project-scoped) — for
// endpoints like listing every project in the workspace, used to resolve
// projectID by name when it is not pinned in config (see
// Provider.ensureProjectID).
func (c *Client) workspacePath(suffix string) string {
	return fmt.Sprintf("%s/api/v1/workspaces/%s/%s", c.baseURL, url.PathEscape(c.workspace), suffix)
}

// APIError is a Plane HTTP failure. Classified is the same
// tasklifecycle.ProviderError taxonomy corechain already uses for agent
// process/provider failures (auth_failed, rate_limited, quota_exceeded, ...)
// — reused here rather than inventing a parallel one, since a Plane task
// mutation failing for "rate limited" means the same thing operationally as
// an agent launch failing for "rate limited".
type APIError struct {
	Operation  string
	StatusCode int
	Body       string
	Classified *tasklifecycle.ProviderError
}

func (e *APIError) Error() string {
	if e.Classified != nil {
		reason := e.Classified.Reason
		if e.Classified.Detail != "" {
			reason += ": " + e.Classified.Detail
		}
		return fmt.Sprintf("plane API %s failed (%d): %s", e.Operation, e.StatusCode, reason)
	}
	body := strings.TrimSpace(e.Body)
	if len(body) > 300 {
		body = body[:300] + "..."
	}
	return fmt.Sprintf("plane API %s failed (%d): %s", e.Operation, e.StatusCode, body)
}

func (c *Client) do(ctx context.Context, method string, path string, query url.Values, body any, out any, operation string) error {
	fullURL := path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("plane: encode %s request: %w", operation, err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return fmt.Errorf("plane: build %s request: %w", operation, err)
	}
	req.Header.Set("X-API-Key", c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if classified := tasklifecycle.ClassifyProviderError("plane", "plane-api", operation, err, ""); classified != nil {
			return &APIError{Operation: operation, Classified: classified}
		}
		return fmt.Errorf("plane: %s request failed: %w", operation, err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("plane: read %s response: %w", operation, readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			Operation:  operation,
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
			Classified: classifyHTTPFailure(operation, resp.StatusCode, respBody),
		}
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("plane: decode %s response: %w", operation, err)
	}
	return nil
}

// classifyHTTPFailure maps a non-2xx Plane response onto
// tasklifecycle.ProviderError. It first tries the shared text-based
// classifier (which already recognizes "rate limit", "unauthorized", etc in
// a response body), then falls back to a status-code default for the two
// failure modes Plane's documented rate limiting/auth guarantees make certain
// enough to classify without keyword matching.
func classifyHTTPFailure(operation string, statusCode int, body []byte) *tasklifecycle.ProviderError {
	if classified := tasklifecycle.ClassifyProviderError("plane", "plane-api", operation, fmt.Errorf("http %d", statusCode), string(body)); classified != nil {
		return classified
	}
	switch statusCode {
	case http.StatusTooManyRequests:
		return tasklifecycle.NewProviderError("plane", "rate_limited", operation, "Plane API rate limit was hit (60 requests/minute).", string(body))
	case http.StatusUnauthorized, http.StatusForbidden:
		return tasklifecycle.NewProviderError("plane", "auth_failed", operation, "Plane API authentication failed.", string(body))
	default:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
