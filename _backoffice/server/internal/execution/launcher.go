package execution

import (
	"fmt"
	"strings"

	"github.com/eggs-gd/core.eggs.gd/lib/chain"
)

// NewAgentLauncher plans the provider-neutral launch command for a validated task.
func NewAgentLauncher(in <-chan *Task, out chan<- *Task, planner Planner) chain.Processor {
	return chain.NewDecorator(in, out, &AgentLauncher{Planner: planner})
}

// AgentLauncher resolves agent command/plan gates. It must stay provider-neutral.
type AgentLauncher struct {
	Planner Planner
}

type Planner interface {
	Plan(name string, task TaskContext) (Plan, error)
}

func (launcher *AgentLauncher) Decorate(task *Task) (*Task, error) {
	if task.HasLaunchGateFailures() {
		return task, nil
	}

	agent := firstNonEmpty(task.Launch.Agent, task.Assignee)
	if agent == "" || agent == "unassigned" {
		task.FailLaunchGate("command", "agent is not set")
		task.ResolveLaunchEvaluation()
		return task, nil
	}

	task.LaunchEvaluation.Agent = agent
	if launcher.Planner == nil {
		return nil, fmt.Errorf("agent planner is not configured")
	}
	plan, err := launcher.Planner.Plan(agent, agentTaskContext(*task))
	if err != nil {
		task.FailLaunchGate("command", fmt.Sprintf("launch command is not configured for agent %q", agent))
		task.ResolveLaunchEvaluation()
		return task, nil
	}

	visibility := VisibilityClass(firstNonEmpty(plan.VisibilityClass, string(VisibilityUnknown)))
	if visibility == "" {
		visibility = VisibilityUnknown
	}
	task.LaunchEvaluation.VisibilityMode = string(visibility)
	task.LaunchEvaluation.Backend = plan.Backend
	task.LaunchEvaluation.WorkingDir = plan.WorkingDir
	task.LaunchEvaluation.Prompt = plan.Prompt
	task.LaunchEvaluation.InitialInput = plan.InitialInput
	task.LaunchEvaluation.Command = append([]string{}, plan.Command...)

	if !plan.LiveReady {
		gate := "agent_live_ready"
		switch visibility {
		case VisibilityCoreVisible, VisibilityHeadless:
			gate = "agent_visibility"
		}
		// Missing CLI binaries are setup failures that should block the task
		// with an actionable provider diagnostic (CORE-150), not a routine skip.
		notesLower := strings.ToLower(plan.LiveNotes)
		if strings.Contains(notesLower, "binary could not be resolved") ||
			strings.Contains(notesLower, "is not available on the daemon path") {
			gate = "agent_executable"
		}
		task.FailLaunchGate(gate, plan.LiveNotes)
		task.ResolveLaunchEvaluation()
		return task, nil
	}
	if !visibility.AllowsDaemonAutoLaunch() && !strings.EqualFold(strings.TrimSpace(task.Launch.Mode), LaunchModeAllowCoreVisible) {
		task.FailLaunchGate("agent_visibility", fmt.Sprintf(
			"backend %q visibility is %q; daemon auto-launch requires app_visible or cli_visible (override with launch.mode=%s)",
			plan.Backend, visibility, LaunchModeAllowCoreVisible,
		))
		task.ResolveLaunchEvaluation()
		return task, nil
	}

	task.Pass("command")
	task.ResolveLaunchEvaluation()
	return task, nil
}

func (launcher *AgentLauncher) Stop() {}

func agentTaskContext(task Task) TaskContext {
	return TaskContext{
		ID:           task.ID,
		Ref:          task.Ref,
		Title:        task.Title,
		Status:       task.Status,
		Type:         task.Type,
		Priority:     task.Priority,
		ProjectID:    task.ProjectID,
		WorkspaceID:  task.WorkspaceID,
		Repository:   task.LaunchEvaluation.Repository,
		RelativePath: task.RelativePath,
		WorkingDir:   task.LaunchEvaluation.WorkingDir,
		Assignee:     task.Assignee,
		LaunchMode:   task.Launch.Mode,
		Body:         task.Body,
	}
}
