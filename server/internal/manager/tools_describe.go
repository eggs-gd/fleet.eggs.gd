package manager

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/projectscan"
	"github.com/eggs-gd/fleet.eggs.gd/internal/workfiles"
)

// readmeExcerptMaxBytes caps how much of a repository's README ProjectFacts
// reads back, so one call stays cheap even for a workspace group with many
// members.
const readmeExcerptMaxBytes = 1500

// ProjectFacts is what a Manager needs to write a project's one-paragraph
// description: what is on the card now, whether that is still just the
// scanner's own guess, and enough of the project's repository (or, for a
// group, each member's) README to write a real one from. It reads only; it
// changes nothing.
func (s *Service) ProjectFacts(project string) Response {
	if !workfiles.ValidProjectID(project) {
		return failResponse(Failure{Code: FailureValidation, Message: "project id is required"})
	}
	if strings.TrimSpace(s.DataRoot) == "" {
		return failResponse(Failure{Code: FailureProviderError, Message: "data root is not configured"})
	}
	view := BoardView{}
	if s.Board != nil {
		view = s.Board.Board()
	}
	var ws *board.Workspace
	for i := range view.Workspaces {
		if strings.EqualFold(view.Workspaces[i].ID, project) {
			ws = &view.Workspaces[i]
			break
		}
	}
	if ws == nil {
		return failResponse(Failure{Code: FailureNotFound, Message: fmt.Sprintf("project %q not found", project)})
	}
	registryDir := filepath.Join(s.DataRoot, "_registry")
	facts, err := projectscan.RepositoryFactsFor(registryDir, ws.Repositories, readmeExcerptMaxBytes)
	if err != nil {
		return failureFromErr(err)
	}
	repos := make([]map[string]any, 0, len(facts))
	for _, fact := range facts {
		repos = append(repos, map[string]any{
			"relative_path":  fact.RelativePath,
			"summary":        fact.Summary,
			"summary_source": fact.SummarySource,
			"readme_excerpt": fact.ReadmeExcerpt,
		})
	}
	return Response{OK: true, Result: &Result{Action: "project_facts", Detail: map[string]any{
		"project":        ws.ID,
		"kind":           ws.Kind,
		"summary":        ws.Summary,
		"summary_source": firstNonEmpty(ws.SummarySource, "generated"),
		"repositories":   repos,
	}}}
}

// Describe records the project's one-paragraph description and marks it
// confirmed, so the scanner leaves it alone from now on. Call it only after
// the person has approved the wording — this is the only write ProjectFacts
// leads to.
func (s *Service) Describe(project, summary string) Response {
	if strings.TrimSpace(s.DataRoot) == "" {
		return failResponse(Failure{Code: FailureProviderError, Message: "data root is not configured"})
	}
	path, err := workfiles.SetProjectSummary(s.DataRoot, project, summary)
	if err != nil {
		return failureFromErr(err)
	}
	return Response{OK: true, Result: &Result{Action: "describe", Path: path}}
}
