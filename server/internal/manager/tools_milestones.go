package manager

import (
	"context"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/workfiles"
)

// MilestoneCreate adds a new, parked milestone to a project card.
func (s *Service) MilestoneCreate(project, title, version string) Response {
	if strings.TrimSpace(s.DataRoot) == "" {
		return failResponse(Failure{Code: FailureProviderError, Message: "data root is not configured"})
	}
	if !workfiles.ValidProjectID(project) {
		return failResponse(Failure{Code: FailureValidation, Message: "project id is required"})
	}
	milestone, err := workfiles.CreateMilestone(s.DataRoot, project, title, version)
	if err != nil {
		return failureFromErr(err)
	}
	s.projectCardChanged(project)
	return Response{OK: true, Result: &Result{Action: "milestone_create", Status: milestone.State, Detail: milestone}}
}

// MilestoneActivate flips one milestone's state to active. Other milestones
// on the same card are left as they are — any number can be active at once.
func (s *Service) MilestoneActivate(project, milestoneID string) Response {
	if strings.TrimSpace(s.DataRoot) == "" {
		return failResponse(Failure{Code: FailureProviderError, Message: "data root is not configured"})
	}
	if !workfiles.ValidProjectID(project) {
		return failResponse(Failure{Code: FailureValidation, Message: "project id is required"})
	}
	milestone, err := workfiles.SetMilestoneState(s.DataRoot, project, milestoneID, "active")
	if err != nil {
		return failureFromErr(err)
	}
	s.projectCardChanged(project)
	return Response{OK: true, Result: &Result{Action: "milestone_activate", Status: milestone.State, Detail: milestone}}
}

// AssignTaskMilestone sets, or clears with an empty milestoneID, the
// milestone on one task — the same field a dashboard drag-and-drop writes.
func (s *Service) AssignTaskMilestone(ctx context.Context, ref, milestoneID string) Response {
	milestoneID = strings.TrimSpace(milestoneID)
	task, err := patchViaService(ctx, s.Tasks, ref, taskflow.PatchInput{Milestone: &milestoneID, Actor: "manager"})
	if err != nil {
		return s.missingTask(ctx, ref, err)
	}
	return Response{OK: true, Result: &Result{Action: "assign_milestone", Ref: task.Ref, Status: task.Status, Detail: task}}
}
