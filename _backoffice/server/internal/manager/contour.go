package manager

import (
	"context"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
)

// ContourName is the independent Contour 1 entry point for task management
// (_docs/TASK_FLOW_CONTOURS.md §3). Unlike change-detection and execution
// contours, Contour 1 is request/response: Human → Manager → TaskService.
const ContourName = "task_management"

// TaskManagement is the Contour 1 surface of TaskService.
//
// Manager/command flows may create, patch (status/assignee/priority/comment),
// and query tasks. This interface deliberately omits launch claims and
// execution-result finalization so Contour 1 cannot synchronously continue
// into agent launch or finalization.
type TaskManagement interface {
	Create(ctx context.Context, input taskflow.CreateTask) (taskflow.Task, error)
	Get(ctx context.Context, id string) (taskflow.Task, error)
	List(ctx context.Context, filter taskflow.TaskFilter) ([]taskflow.Task, error)
	Patch(ctx context.Context, id string, patch taskflow.PatchInput) (taskflow.Task, error)
}

// Compile-time: the serialized TaskService implementation satisfies Contour 1.
var _ TaskManagement = (*taskflow.Service)(nil)
