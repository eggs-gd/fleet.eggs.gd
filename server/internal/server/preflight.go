package server

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/managerskills"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// preflight checks what the server cannot run without and repairs what it can.
// It runs once before anything starts and reports every problem together, so
// one restart is enough to see the whole list. After startup a failing part is
// reported through Health and retried instead of stopping the process.
func preflight(cfg Config) error {
	var problems []error
	check := func(name string, err error) {
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", name, err))
		}
	}
	check("dashboard", dashboardPresent(cfg))
	dataErr := writable(cfg.CoreRoot)
	check("data root is not writable", dataErr)
	runtimeRoot := cfg.RuntimeRoot
	if runtimeRoot == "" {
		runtimeRoot = cfg.CoreRoot
	}
	check("runtime database is not writable", audit.AppendEvent(runtimeRoot, audit.Event{Type: "server.started", Message: cfg.Version}))
	if dataErr == nil {
		// These write into the data root, so they only make sense when it works.
		if _, err := tasklifecycle.EnsureCounters(cfg.CoreRoot); err != nil {
			check("ref counters", err)
		}
		check("manager skills", managerskills.Install(cfg.CoreRoot))
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("startup checks failed:\n%w", errors.Join(problems...))
}

func writable(dir string) error {
	probe, err := os.CreateTemp(dir, ".fleet-probe-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	probe.Close()
	return os.Remove(filepath.Clean(name))
}

func dashboardPresent(cfg Config) error {
	if cfg.Backoffice == nil {
		return errors.New("no dashboard is configured")
	}
	if _, err := fs.Stat(cfg.Backoffice, "index.html"); err != nil {
		return fmt.Errorf("%s has no index.html: build the dashboard with `make build`, or pass --backoffice-dir", cfg.BackofficeSource)
	}
	return nil
}
