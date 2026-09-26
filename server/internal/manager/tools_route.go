package manager

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/eggs-gd/fleet.eggs.gd/internal/workfiles"
)

// ResolveProject ranks project candidates for a phrase.
func (s *Service) ResolveProject(phrase string) Response {
	phrase = strings.TrimSpace(phrase)
	exact, fuzzy, err := s.projectCandidates(phrase)
	if err != nil {
		return failureFromErr(err)
	}
	verdict := "none"
	switch {
	case len(exact) == 1:
		verdict = "confident"
	case len(exact) > 1 || len(fuzzy) > 0:
		verdict = "ambiguous"
	}
	candidates := exact
	if len(candidates) == 0 {
		candidates = fuzzy
	}
	return Response{OK: true, Result: &Result{Action: "resolve_project", Detail: map[string]any{
		"verdict":    verdict,
		"project":    firstOrEmpty(candidates),
		"candidates": candidates,
	}}}
}

// Similar lists tasks that share words with text.
func (s *Service) Similar(ctx context.Context, text string) Response {
	tasks, err := s.flowTasks(ctx)
	if err != nil {
		return failureFromErr(err)
	}
	type hit struct {
		ref, title string
		score      int
	}
	words := significantWords(text)
	var hits []hit
	for _, task := range tasks {
		score := overlap(words, significantWords(task.Title+" "+task.Summary+" "+task.Body))
		if score == 0 {
			continue
		}
		hits = append(hits, hit{ref: task.Ref, title: task.Title, score: score})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score == hits[j].score {
			return hits[i].ref < hits[j].ref
		}
		return hits[i].score > hits[j].score
	})
	if len(hits) > 5 {
		hits = hits[:5]
	}
	out := make([]map[string]any, 0, len(hits))
	for _, item := range hits {
		out = append(out, map[string]any{"ref": item.ref, "title": item.title, "score": item.score})
	}
	return Response{OK: true, Result: &Result{Action: "similar", Detail: out}}
}

// Route recommends a worker. An explicit assignee wins. A project card with
// a default_assignee nobody knows is a data error the Manager cannot fix, so it
// is reported as a warning and the recommendation stops being confident.
func (s *Service) Route(ctx context.Context, draft Intent) Response {
	tasks, err := s.flowTasks(ctx)
	if err != nil {
		return failureFromErr(err)
	}
	busy := map[string]string{}
	for _, held := range ownershipOf(tasks) {
		busy[held.Assignee] = held.Ref
	}
	pick := func(worker string, confident bool, reason string, warnings ...string) Response {
		return Response{OK: true, Result: &Result{
			Action:   "route",
			Detail:   map[string]any{"worker": worker, "confident": confident && len(warnings) == 0, "reason": reason},
			Warnings: warnings,
		}}
	}
	assignee := strings.ToLower(strings.TrimSpace(draft.Assignee))
	if assignee != "" && assignee != "unassigned" {
		switch {
		case isWorker(assignee):
			return pick(assignee, true, "explicit instruction")
		case workfiles.HumanWorker(s.DataRoot, assignee):
			return pick(assignee, true, "explicit instruction (human worker)")
		default:
			return failResponse(Failure{
				Code:        FailureValidation,
				Message:     fmt.Sprintf("unknown worker %q", assignee),
				Suggestions: workerAgents,
			})
		}
	}
	var warnings []string
	if override := workfiles.DefaultAssignee(s.DataRoot, draft.Project); override != "" {
		if isWorker(override) || workfiles.HumanWorker(s.DataRoot, override) {
			return pick(override, busy[override] == "", "project override")
		}
		warnings = append(warnings, fmt.Sprintf("project %s has default_assignee %q, which is not a known worker; fix PROJECT.md", draft.Project, override))
	}
	if busy["claude"] == "" {
		return pick("claude", true, "category "+strings.TrimSpace(draft.Type), warnings...)
	}
	for _, agent := range []string{"cursor", "codex"} {
		if busy[agent] == "" {
			return pick(agent, false, "preferred worker is busy; this one is free", warnings...)
		}
	}
	return pick("", false, "no free worker", warnings...)
}

// Validate checks a draft and writes nothing.
func (s *Service) Validate(ctx context.Context, draft Intent) Response {
	var fixes []string
	if err := ValidateIntent(draft); err != nil {
		fixes = append(fixes, err.Error())
	}
	if draft.Kind == KindTask && strings.TrimSpace(draft.Acceptance) == "" {
		fixes = append(fixes, "add acceptance_criteria")
	}
	if status := strings.TrimSpace(draft.Status); status != "" && !knownStatus(status) {
		fixes = append(fixes, "status "+status+" is not allowed")
	}
	known := s.knownRepositories()
	for _, repo := range draftRepos(draft) {
		if len(known) > 0 && !repoKnown(known, repo) {
			fixes = append(fixes, "unknown repository "+repo)
		}
	}
	fixes = append(fixes, s.checkDependencies(ctx, draft.Ref, draft.DependsOn)...)
	ok := len(fixes) == 0
	resp := Response{OK: ok, Result: &Result{Action: "validate", Detail: map[string]any{"fixes": fixes}}}
	if !ok {
		resp.Failure = &Failure{Code: FailureValidation, Message: "draft needs changes", Suggestions: fixes}
	}
	return resp
}

// projectCandidates matches phrase against every workspace and project id,
// title, and the aliases on its project card.
func (s *Service) projectCandidates(phrase string) (exact, fuzzy []string, err error) {
	view := BoardView{}
	if s != nil && s.Board != nil {
		view = s.Board.Board()
	}
	needle := strings.ToLower(strings.TrimSpace(phrase))
	consider := func(id string, names ...string) error {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil
		}
		if s.DataRoot != "" {
			folder, _, _ := strings.Cut(id, "/")
			aliases, err := workfiles.ProjectAliases(s.DataRoot, folder)
			if err != nil {
				return err
			}
			names = append(names, aliases...)
		}
		for _, name := range names {
			if strings.EqualFold(strings.TrimSpace(name), phrase) {
				exact = appendFold(exact, id)
				return nil
			}
		}
		for _, name := range names {
			if needle != "" && strings.Contains(strings.ToLower(name), needle) {
				fuzzy = appendFold(fuzzy, id)
				return nil
			}
		}
		return nil
	}
	for _, ws := range view.Workspaces {
		if err := consider(ws.ID, ws.ID, ws.Title); err != nil {
			return nil, nil, err
		}
	}
	for _, project := range view.Projects {
		id := firstNonEmpty(project.WorkspaceID, project.ID)
		if err := consider(id, project.ID, id, project.Title); err != nil {
			return nil, nil, err
		}
	}
	return exact, fuzzy, nil
}

func appendFold(list []string, id string) []string {
	for _, have := range list {
		if strings.EqualFold(have, id) {
			return list
		}
	}
	return append(list, id)
}

func (s *Service) knownRepositories() []string {
	view := BoardView{}
	if s != nil && s.Board != nil {
		view = s.Board.Board()
	}
	var repos []string
	for _, ws := range view.Workspaces {
		repos = append(repos, ws.Repositories...)
	}
	for _, project := range view.Projects {
		repos = append(repos, project.Repositories...)
	}
	for _, repo := range view.Registry.Repositories {
		repos = append(repos, repo.RelativePath, repo.Name)
	}
	return uniqueFold(repos)
}

// checkDependencies returns the problems in a depends_on list for the task
// ref (empty for a draft): refs that are not on the board, a task waiting on
// itself, and cycles. A board that cannot be read is reported, not skipped.
func (s *Service) checkDependencies(ctx context.Context, ref string, deps []string) []string {
	deps = upperAll(deps)
	if len(deps) == 0 {
		return nil
	}
	tasks, err := s.flowTasks(ctx)
	if err != nil {
		return []string{"could not check depends_on: " + err.Error()}
	}
	start := strings.ToUpper(strings.TrimSpace(ref))
	if start == "" {
		start = "DRAFT"
	}
	graph := map[string][]string{}
	for _, task := range tasks {
		graph[strings.ToUpper(task.Ref)] = upperAll(task.DependsOn)
	}
	var fixes []string
	for _, dep := range deps {
		if dep == start {
			fixes = append(fixes, "a task cannot depend on itself")
		} else if _, ok := graph[dep]; !ok {
			fixes = append(fixes, "depends_on "+dep+" is not on the board")
		}
	}
	graph[start] = deps
	seen := map[string]int{}
	var visit func(string) bool
	visit = func(node string) bool {
		switch seen[node] {
		case 1:
			return true
		case 2:
			return false
		}
		seen[node] = 1
		for _, next := range graph[node] {
			if visit(next) {
				return true
			}
		}
		seen[node] = 2
		return false
	}
	if visit(start) {
		fixes = append(fixes, "depends_on contains a cycle")
	}
	return fixes
}

func knownStatus(status string) bool {
	switch status {
	case "backlog", "needs_rework", "todo", "doing", "blocked", "needs_review", "done", "archived":
		return true
	default:
		return false
	}
}

func draftRepos(draft Intent) []string {
	var repos []string
	if strings.TrimSpace(draft.Repository) != "" {
		repos = append(repos, draft.Repository)
	}
	repos = append(repos, draft.Repositories...)
	return repos
}

func repoKnown(known []string, repo string) bool {
	for _, candidate := range known {
		if strings.EqualFold(candidate, repo) || strings.EqualFold(repoBaseName(candidate), repo) {
			return true
		}
	}
	return false
}

func upperAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func significantWords(text string) map[string]bool {
	words := map[string]bool{}
	for _, word := range strings.Fields(strings.ToLower(text)) {
		word = strings.Trim(word, ".,:;!?\"'`")
		if utf8.RuneCountInString(word) < 4 {
			continue
		}
		words[word] = true
	}
	return words
}

func overlap(left, right map[string]bool) int {
	score := 0
	for word := range left {
		if right[word] {
			score++
		}
	}
	return score
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// knownAssignee reports whether name may hold a task: unassigned, an agent, or
// a person in the Fleet roster.
func (s *Service) knownAssignee(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "unassigned" || isWorker(name) || workfiles.HumanWorker(s.DataRoot, name)
}

func isWorker(name string) bool {
	for _, agent := range workerAgents {
		if agent == name {
			return true
		}
	}
	return false
}
