package execution

import "strings"

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func taskLabel(task *Task) string {
	if task == nil {
		return "unknown task"
	}
	if task.Ref != "" {
		return task.Ref
	}
	return firstNonEmpty(task.ID, task.RelativePath, task.Path, "unknown task")
}

func (s *Service) emitTaskEvent(eventType string, task Task, message string, details map[string]any) {
	if s == nil || s.board.EmitTaskEvent == nil {
		return
	}
	s.board.EmitTaskEvent(eventType, task, message, details)
}

func (s *Service) emitRuntimeEvent(typ, path, message string, details map[string]any) {
	if s == nil || s.board.EmitRuntimeEvent == nil {
		return
	}
	s.board.EmitRuntimeEvent(typ, path, message, details)
}
