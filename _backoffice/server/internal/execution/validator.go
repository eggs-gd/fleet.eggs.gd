package execution

import (
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/core.eggs.gd/lib/chain"
)

// NewValidator gates launch using tasklifecycle.EvaluateLaunch.
// knownTasks supplies the board snapshot for depends_on resolution (CORE-148).
func NewValidator(root string, knownTasks func() []Task, in <-chan *Task, out chan<- *Task) chain.Processor {
	return chain.NewDecorator(in, out, &Validator{Root: root, KnownTasks: knownTasks})
}

// Validator wires pure launch-gate rules into the execution processor chain.
type Validator struct {
	Root       string
	KnownTasks func() []Task
}

func (validator *Validator) Decorate(task *Task) (*Task, error) {
	tasklifecycle.EvaluateLaunch(task, validator.Root, validator.knownTasks())
	return task, nil
}

func (validator *Validator) knownTasks() []Task {
	if validator == nil || validator.KnownTasks == nil {
		return nil
	}
	return validator.KnownTasks()
}

func (validator *Validator) Stop() {}
