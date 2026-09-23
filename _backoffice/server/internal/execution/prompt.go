package execution

import "github.com/eggs-gd/core.eggs.gd/internal/executionapi"

func BuildPrompt(task TaskContext) string {
	return executionapi.BuildPrompt(task)
}
