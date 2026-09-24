package markdown

import (
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// ObserveFileChange reloads a task card after a filesystem change and returns
// a normalized TaskEvent for Contour 2 (CORE-103 / CORE-111).
//
// FsWalker detects the path; this method owns the Markdown reload and event
// shape. Generic launcher code must not take filesystem paths as input.
// Before is nil here — callers fill it from canonical state when observing.
func (p *Provider) ObserveFileChange(path string) (taskflow.TaskEvent, tasklifecycle.Task, error) {
	task, err := p.Reload(path)
	if err != nil {
		return taskflow.TaskEvent{}, tasklifecycle.Task{}, err
	}
	return TaskEventFor(nil, task, taskflow.TaskEventSourceMarkdownFS), task, nil
}

// TaskEventFor builds a normalized TaskEvent from a loaded Markdown task.
// TaskID is the opaque locator (relative path) used with TaskService.Get.
// Kind is derived from Before/After for migration compatibility.
func TaskEventFor(before *tasklifecycle.Task, after tasklifecycle.Task, source taskflow.TaskEventSource) taskflow.TaskEvent {
	if source == "" {
		source = taskflow.TaskEventSourceMarkdownFS
	}
	var beforeFlow *taskflow.Task
	if before != nil {
		b := toFlowTask(*before)
		beforeFlow = &b
	}
	return taskflow.NewTaskEvent(taskEventID(after), beforeFlow, toFlowTask(after), source)
}

// ClassifyTaskEventKind picks the minimal TaskEventKind for an observed
// Markdown upsert. Prefer TaskEvent.StatusChanged / DeriveTaskEventKind;
// kept for call sites that still pass raw statuses during migration.
func ClassifyTaskEventKind(previousStatus string, hadPrevious bool, current tasklifecycle.Task) taskflow.TaskEventKind {
	if hadPrevious && previousStatus != "" && previousStatus != current.Status {
		return taskflow.TaskEventStatusChanged
	}
	return taskflow.TaskEventUpserted
}

func taskEventID(task tasklifecycle.Task) string {
	if id := strings.TrimSpace(task.RelativePath); id != "" {
		return id
	}
	return strings.TrimSpace(task.Path)
}
