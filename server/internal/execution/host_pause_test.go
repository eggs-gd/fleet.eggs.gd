package execution

import (
	"context"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

type pauseTaskService struct{}

func (pauseTaskService) Create(context.Context, taskflow.CreateTask) (taskflow.Task, error) {
	return taskflow.Task{}, nil
}
func (pauseTaskService) Get(_ context.Context, id string) (taskflow.Task, error) {
	return taskflow.Task{ID: id, Locator: id, Status: taskflow.StatusDoing}, nil
}
func (pauseTaskService) List(context.Context, taskflow.TaskFilter) ([]taskflow.Task, error) {
	return nil, nil
}
func (pauseTaskService) Claim(_ context.Context, id string) (taskflow.Task, error) {
	return taskflow.Task{ID: id, Status: taskflow.StatusDoing}, nil
}
func (pauseTaskService) Transition(context.Context, string, taskflow.Status, taskflow.TransitionMeta) error {
	return nil
}
func (pauseTaskService) Update(context.Context, taskflow.Task) error { return nil }
func (pauseTaskService) Patch(context.Context, string, taskflow.PatchInput) (taskflow.Task, error) {
	return taskflow.Task{}, nil
}
func (pauseTaskService) AddComment(context.Context, string, string) error { return nil }
func (pauseTaskService) ReportExecution(context.Context, taskflow.ExecutionResult) error {
	return nil
}

func TestPauseSessionNotifiesOperatorOnce(t *testing.T) {
	var got []string
	svc := &Service{
		tasks: pauseTaskService{},
		hub:   newSessionHub(t.TempDir(), nil, nil),
		onAttention: func(ref, question string) {
			got = append(got, ref+"|"+question)
		},
	}
	session := &RuntimeSession{
		ClaimID: "claim-1",
		Result:  &tasklifecycle.WorkerResult{Question: "How should the worker obtain go_diagnostics?"},
	}
	task := Task{Ref: "CORE-152", RelativePath: "tasks/core-152.md", Status: "doing"}
	svc.pauseSessionForOperatorInput(session, task)
	svc.pauseSessionForOperatorInput(session, task)
	if len(got) != 1 || got[0] != "CORE-152|How should the worker obtain go_diagnostics?" {
		t.Fatalf("notices = %#v", got)
	}
	if session.ExecutionStatus != "waiting_input" {
		t.Fatalf("execution status = %q", session.ExecutionStatus)
	}
}

func TestOpenOperatorQuestionReadsLatestPause(t *testing.T) {
	question, ok := openOperatorQuestion(Task{
		Status: "doing",
		Comments: []tasklifecycle.Comment{{
			Text: "Agent execution reported it is waiting on operator input. Question: How should this Cursor worker obtain go_diagnostics? Artifacts: - `artifacts/x.md` Session log: `_registry/sessions/a.log`",
		}},
	})
	if !ok || question != "How should this Cursor worker obtain go_diagnostics?" {
		t.Fatalf("question = %q ok=%v", question, ok)
	}
	if _, ok := openOperatorQuestion(Task{Status: "needs_review", Comments: []tasklifecycle.Comment{{Text: "waiting on operator input"}}}); ok {
		t.Fatal("closed task should not announce a pause")
	}
}

func TestAnnounceOpenPausesReachesManagerOnce(t *testing.T) {
	var notices []string
	svc := &Service{onAttention: func(ref, question string) {
		notices = append(notices, ref+"|"+question)
	}}
	task := Task{
		Status: "doing",
		Ref:    "CORE-152",
		Comments: []tasklifecycle.Comment{{
			Text: "Agent execution reported it is waiting on operator input. Question: obtain go_diagnostics?",
		}},
	}
	svc.AnnounceOpenPauses([]Task{task})
	svc.AnnounceOpenPauses([]Task{task})
	if len(notices) != 1 || !strings.Contains(notices[0], "CORE-152|obtain go_diagnostics?") {
		t.Fatalf("notices = %#v", notices)
	}
}

func TestAttachedRecoveryKeepsWaitingStatus(t *testing.T) {
	session, question, waiting := applyAttachedRecovery(RuntimeSession{
		ExecutionStatus: "waiting_input",
		Status:          "waiting_input",
		BlockingReason:  "Need a decision.",
		LastEvent:       "waiting_operator_input",
	}, sessionRecoveryDecision{AttachAsSession: true, ExecutionStatus: "resumable", Status: "resumable"})
	if !waiting || question != "Need a decision." {
		t.Fatalf("waiting=%v question=%q", waiting, question)
	}
	if session.ExecutionStatus != "waiting_input" || session.Status != "waiting_input" {
		t.Fatalf("status = %s/%s", session.ExecutionStatus, session.Status)
	}
}
