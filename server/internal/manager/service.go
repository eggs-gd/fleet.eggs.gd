package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
)

// Transcriber turns audio into text. Production wiring uses gpt-4o-transcribe.
type Transcriber interface {
	Transcribe(ctx context.Context, audio []byte, contentType string, vocabulary Vocabulary) (string, error)
}

// IntentClassifier turns free text into a structured Intent when fast-path misses.
type IntentClassifier interface {
	Classify(ctx context.Context, text string, vocabulary Vocabulary) (Intent, error)
}

// UnconfiguredTranscriber is the MVP stub until STT credentials are wired.
type UnconfiguredTranscriber struct{}

func (UnconfiguredTranscriber) Transcribe(context.Context, []byte, string, Vocabulary) (string, error) {
	return "", Failure{Code: FailureSTTNotConfigured, Message: "speech-to-text is not configured; submit text or wire gpt-4o-transcribe"}
}

// UnconfiguredClassifier is the MVP stub until a cheap structured-output model is wired.
type UnconfiguredClassifier struct{}

func (UnconfiguredClassifier) Classify(context.Context, string, Vocabulary) (Intent, error) {
	return Intent{}, Failure{Code: FailureLLMNotConfigured, Message: "manager LLM is not configured; use a deterministic command or wire structured-output model"}
}

// Service is Contour 1 (task management): Human → Manager → TaskService.
// It ends after TaskService accepts the command and returns an operator
// confirmation. It must not watch changes, launch agents, wait for
// executions, finalize, own change-detection walkers, or call providers
// directly (_docs/TASK_FLOW_CONTOURS.md §3, CORE-110).
type Service struct {
	Tasks      TaskManagement
	Board      BoardReader
	STT        Transcriber
	Classifier IntentClassifier
}

func NewService(tasks TaskManagement, board BoardReader) *Service {
	return &Service{
		Tasks:      tasks,
		Board:      board,
		STT:        UnconfiguredTranscriber{},
		Classifier: UnconfiguredClassifier{},
	}
}

func (s *Service) Vocabulary() Vocabulary {
	if s == nil || s.Board == nil {
		return BuildVocabulary(BoardView{})
	}
	return BuildVocabulary(s.Board.Board())
}

// SubmitText classifies and executes a text submission.
func (s *Service) SubmitText(ctx context.Context, req TextRequest) Response {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return failResponse(Failure{Code: FailureValidation, Message: "text is required"})
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "text"
	}

	intent, fail := ParseFastPath(text)
	if fail != nil {
		return failResponse(*fail)
	}
	if intent != nil {
		intent.RawTranscript = text
		return s.execute(ctx, *intent, PathDeterministic, source)
	}

	if s.Classifier == nil {
		s.Classifier = UnconfiguredClassifier{}
	}
	classified, err := s.Classifier.Classify(ctx, text, s.Vocabulary())
	if err != nil {
		return failureFromErr(err)
	}
	classified.RawTranscript = text
	if err := ValidateIntent(classified); err != nil {
		return failResponse(Failure{Code: FailureMalformedModelOutput, Message: err.Error()})
	}
	return s.execute(ctx, classified, PathLLM, source)
}

// SubmitAudio transcribes then runs the text pipeline.
func (s *Service) SubmitAudio(ctx context.Context, audio []byte, contentType string, source string) Response {
	if len(audio) == 0 {
		return failResponse(Failure{Code: FailureValidation, Message: "audio is required"})
	}
	if s.STT == nil {
		s.STT = UnconfiguredTranscriber{}
	}
	transcript, err := s.STT.Transcribe(ctx, audio, contentType, s.Vocabulary())
	if err != nil {
		return failureFromErr(err)
	}
	if source == "" {
		source = "voice"
	}
	resp := s.SubmitText(ctx, TextRequest{Text: transcript, Source: source})
	resp.Transcript = transcript
	return resp
}

// SubmitCommand executes an already-structured intent.
func (s *Service) SubmitCommand(ctx context.Context, req CommandRequest) Response {
	if err := ValidateIntent(req.Intent); err != nil {
		return failResponse(Failure{Code: FailureValidation, Message: err.Error()})
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "command"
	}
	return s.execute(ctx, req.Intent, PathCommand, source)
}

func (s *Service) execute(ctx context.Context, intent Intent, path, source string) Response {
	if s == nil || s.Tasks == nil {
		return failResponse(Failure{Code: FailureProviderError, Message: "task service is not configured"})
	}

	switch intent.Kind {
	case KindTask:
		return s.createTask(ctx, intent, path, source)
	case KindStatusChange:
		return s.patchStatus(ctx, intent, path)
	case KindComment:
		return s.addComment(ctx, intent, path)
	case KindAssigneeChange:
		return s.patchAssignee(ctx, intent, path)
	case KindPriorityChange:
		return s.patchPriority(ctx, intent, path)
	case KindCancel:
		return s.cancelTask(ctx, intent, path)
	case KindBoardCommand, KindQuestion:
		return s.lookup(ctx, intent, path)
	default:
		return failResponse(Failure{Code: FailureMalformedModelOutput, Message: fmt.Sprintf("unsupported kind %q", intent.Kind)})
	}
}

func (s *Service) createTask(ctx context.Context, intent Intent, path, source string) Response {
	if resolved, fail := s.resolveProject(intent.Project); fail != nil {
		return failResponse(*fail)
	} else {
		intent.Project = resolved
	}
	if intent.Repository != "" {
		if resolved, fail := s.resolveRepository(intent.Repository, intent.Project); fail != nil {
			return failResponse(*fail)
		} else {
			intent.Repository = resolved
		}
	}

	reason := "Created by Core Manager API."
	if source == "voice" {
		reason = "Created by Core Manager API from voice input."
	}
	task, err := createViaService(ctx, s.Tasks, intent.Title, intent.Description, intent.Project, intent.Repository, intent.Status, intent.Type, intent.Assignee, reason, source, intent.Priority)
	if err != nil {
		return failureFromErr(err)
	}
	return Response{
		OK:     true,
		Path:   path,
		Intent: &intent,
		Result: &Result{Action: "create_task", Ref: task.Ref, Path: task.RelativePath, Status: task.Status, Detail: task},
	}
}

func (s *Service) patchStatus(ctx context.Context, intent Intent, path string) Response {
	task, err := patchViaService(ctx, s.Tasks, intent.Ref, taskflow.PatchInput{
		Status: taskflow.Status(intent.Status),
		Actor:  "manager",
	})
	if err != nil {
		return failureFromErr(err)
	}
	return Response{
		OK:     true,
		Path:   path,
		Intent: &intent,
		Result: &Result{Action: "status_change", Ref: task.Ref, Path: task.RelativePath, Status: task.Status, Detail: task},
	}
}

func (s *Service) addComment(ctx context.Context, intent Intent, path string) Response {
	author := intent.CommentAuthor
	if author == "" {
		author = "alex"
	}
	task, err := patchViaService(ctx, s.Tasks, intent.Ref, taskflow.PatchInput{
		Comment:       intent.Comment,
		CommentAuthor: author,
		Actor:         "manager",
	})
	if err != nil {
		return failureFromErr(err)
	}
	return Response{
		OK:     true,
		Path:   path,
		Intent: &intent,
		Result: &Result{Action: "add_comment", Ref: task.Ref, Path: task.RelativePath, Status: task.Status, Detail: task},
	}
}

func (s *Service) patchAssignee(ctx context.Context, intent Intent, path string) Response {
	task, err := patchViaService(ctx, s.Tasks, intent.Ref, taskflow.PatchInput{
		Assignee: intent.Assignee,
		Actor:    "manager",
	})
	if err != nil {
		return failureFromErr(err)
	}
	return Response{
		OK:     true,
		Path:   path,
		Intent: &intent,
		Result: &Result{Action: "assignee_change", Ref: task.Ref, Path: task.RelativePath, Status: task.Status, Detail: task},
	}
}

func (s *Service) patchPriority(ctx context.Context, intent Intent, path string) Response {
	task, err := patchViaService(ctx, s.Tasks, intent.Ref, taskflow.PatchInput{
		Priority: intent.Priority,
		Actor:    "manager",
	})
	if err != nil {
		return failureFromErr(err)
	}
	return Response{
		OK:     true,
		Path:   path,
		Intent: &intent,
		Result: &Result{Action: "priority_change", Ref: task.Ref, Path: task.RelativePath, Status: task.Status, Detail: task},
	}
}

func (s *Service) cancelTask(ctx context.Context, intent Intent, path string) Response {
	if !intent.Confirm {
		return failResponse(Failure{
			Code:    FailureUnsafeCommand,
			Message: fmt.Sprintf("refusing to archive %s without confirm=true", intent.Ref),
		})
	}
	task, err := patchViaService(ctx, s.Tasks, intent.Ref, taskflow.PatchInput{
		Status: taskflow.StatusArchived,
		Actor:  "manager",
	})
	if err != nil {
		return failureFromErr(err)
	}
	return Response{
		OK:     true,
		Path:   path,
		Intent: &intent,
		Result: &Result{Action: "cancel_task", Ref: task.Ref, Path: task.RelativePath, Status: task.Status, Detail: task},
	}
}

func (s *Service) lookup(ctx context.Context, intent Intent, path string) Response {
	if intent.BoardAction == BoardShowTask || intent.Kind == KindQuestion {
		if intent.Ref != "" {
			task, err := findByRef(ctx, s.Tasks, intent.Ref)
			if err != nil {
				return failureFromErr(err)
			}
			return Response{
				OK:     true,
				Path:   path,
				Intent: &intent,
				Result: &Result{Action: "show_task", Ref: task.Ref, Path: task.RelativePath, Status: task.Status, Detail: task},
			}
		}
	}
	tasks, err := listViaService(ctx, s.Tasks, intent.Project, "", intent.Ref)
	if err != nil {
		return failureFromErr(err)
	}
	return Response{
		OK:     true,
		Path:   path,
		Intent: &intent,
		Result: &Result{Action: "show_board", Detail: tasks},
	}
}

func (s *Service) resolveProject(project string) (string, *Failure) {
	project = strings.TrimSpace(project)
	if project == "" {
		return "", &Failure{Code: FailureValidation, Message: "project is required"}
	}
	view := BoardView{}
	if s.Board != nil {
		view = s.Board.Board()
	}
	var matches []string
	for _, ws := range view.Workspaces {
		if strings.EqualFold(ws.ID, project) || strings.EqualFold(ws.Title, project) {
			matches = append(matches, ws.ID)
		}
	}
	for _, p := range view.Projects {
		if strings.EqualFold(p.ID, project) || strings.EqualFold(p.Title, project) {
			matches = append(matches, firstNonEmpty(p.WorkspaceID, p.ID))
		}
	}
	matches = uniqueFold(matches)
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", &Failure{Code: FailureAmbiguousProject, Message: fmt.Sprintf("project %q matches %v", project, matches)}
	}
	// Allow creating into a known Work folder id even if projection is empty in tests.
	return project, nil
}

func (s *Service) resolveRepository(repo, project string) (string, *Failure) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", nil
	}
	view := BoardView{}
	if s.Board != nil {
		view = s.Board.Board()
	}
	var matches []string
	consider := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if strings.EqualFold(candidate, repo) || strings.EqualFold(repoBaseName(candidate), repo) || strings.Contains(strings.ToLower(candidate), strings.ToLower(repo)) {
			matches = append(matches, candidate)
		}
	}
	for _, ws := range view.Workspaces {
		if project != "" && !strings.EqualFold(ws.ID, project) {
			continue
		}
		for _, path := range ws.Repositories {
			consider(path)
		}
	}
	for _, p := range view.Projects {
		if project != "" && !strings.EqualFold(p.WorkspaceID, project) && !strings.EqualFold(p.ID, project) {
			continue
		}
		for _, path := range p.Repositories {
			consider(path)
		}
	}
	for _, r := range view.Registry.Repositories {
		consider(r.RelativePath)
		consider(r.Name)
	}
	matches = uniqueFold(matches)
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", &Failure{Code: FailureAmbiguousRepository, Message: fmt.Sprintf("repository %q matches %v", repo, matches)}
	}
	return repo, nil
}

// ValidateIntent checks required fields for a structured intent.
func ValidateIntent(intent Intent) error {
	switch intent.Kind {
	case KindTask:
		if strings.TrimSpace(intent.Project) == "" {
			return fmt.Errorf("task intent requires project")
		}
		if strings.TrimSpace(intent.Title) == "" {
			return fmt.Errorf("task intent requires title")
		}
		if strings.TrimSpace(intent.Description) == "" {
			return fmt.Errorf("task intent requires description")
		}
	case KindStatusChange:
		if intent.Ref == "" || intent.Status == "" {
			return fmt.Errorf("status_change requires ref and status")
		}
	case KindComment:
		if intent.Ref == "" || strings.TrimSpace(intent.Comment) == "" {
			return fmt.Errorf("comment requires ref and comment")
		}
	case KindBoardCommand:
		if intent.BoardAction == "" {
			return fmt.Errorf("board_command requires board_action")
		}
	case KindAssigneeChange:
		if intent.Ref == "" || intent.Assignee == "" {
			return fmt.Errorf("assignee_change requires ref and assignee")
		}
	case KindPriorityChange:
		if intent.Ref == "" || intent.Priority == nil {
			return fmt.Errorf("priority_change requires ref and priority")
		}
	case KindCancel:
		if intent.Ref == "" {
			return fmt.Errorf("cancel requires ref")
		}
	case KindQuestion:
		if intent.Ref == "" && strings.TrimSpace(intent.Query) == "" {
			return fmt.Errorf("question requires ref or query")
		}
	default:
		return fmt.Errorf("unknown kind %q", intent.Kind)
	}
	return nil
}

func failResponse(fail Failure) Response {
	return Response{OK: false, Failure: &fail}
}

func failureFromErr(err error) Response {
	if fail, ok := err.(Failure); ok {
		return failResponse(fail)
	}
	return failResponse(Failure{Code: FailureProviderError, Message: err.Error()})
}

func uniqueFold(values []string) []string {
	seen := map[string]string{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = value
		out = append(out, value)
	}
	return out
}

// SchemaObject returns the intent schema as parsed JSON for HTTP responses.
func SchemaObject() (any, error) {
	var raw any
	if err := json.Unmarshal([]byte(IntentJSONSchema), &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
