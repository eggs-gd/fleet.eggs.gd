package execution

import (
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	l "github.com/eggs-gd/fleet.eggs.gd/lib/logger"
)

// Config is host-level execution settings (not provider-specific).
type Config struct {
	Root             string
	RuntimeRoot      string
	DryRun           bool
	SessionTimeout   time.Duration
	LauncherIdentity string
	Logger           *l.Logger
}

func (cfg Config) AllowsProcessStart() bool {
	return !cfg.DryRun
}

func (cfg Config) EffectiveSessionTimeout() time.Duration {
	if cfg.SessionTimeout <= 0 {
		return 10 * time.Minute
	}
	return cfg.SessionTimeout
}

// BoardHooks is the task-board surface execution needs without owning
// board projection. Session/orphan state lives on Service (sessionHub).
type BoardHooks struct {
	UpsertTask       func(task tasklifecycle.Task)
	ListTasks        func() []tasklifecycle.Task
	TaskByPath       func(locator string) (tasklifecycle.Task, bool)
	EmitTaskEvent    func(eventType string, task tasklifecycle.Task, message string, details map[string]any)
	EmitRuntimeEvent func(typ, path, message string, details map[string]any)
}
