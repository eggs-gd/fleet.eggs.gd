package settings

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/projectscan"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// countersEvery is how many ticks pass between full checks of the ref
// counters against the refs already in the tree.
const countersEvery = 30

type watchSeen struct {
	ready       bool
	fingerprint string
}

// Watch keeps the workspace in order while serve is running. Every tick it
// syncs the project cards with the registry. Every countersEvery ticks it also
// checks the ref counters. It polls the saved scan roots too: a full registry
// rewrite runs once at start and again whenever the set of git checkouts
// changes. A failing part is reported to Health and retried on the next tick.
func (s *Scanner) Watch(ctx context.Context, coreRoot string, interval time.Duration) {
	if s == nil || ctx == nil {
		return
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	var seen watchSeen
	s.maintain(coreRoot, true)
	s.watchOnce(coreRoot, nil, &seen)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for tick := 1; ; tick++ {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.maintain(coreRoot, tick%countersEvery == 0)
			s.watchOnce(coreRoot, nil, &seen)
		}
	}
}

// maintain repairs the project cards and, when full is set, the ref counters.
func (s *Scanner) maintain(coreRoot string, full bool) {
	if s == nil {
		return
	}
	_, err := projectscan.MaintainCards(filepath.Join(coreRoot, "_registry"), filepath.Join(coreRoot, "Work"))
	if err != nil {
		s.Health.Fail("workspace", err)
	} else {
		s.Health.OK("workspace")
	}
	if !full {
		return
	}
	if _, err := tasklifecycle.EnsureCounters(coreRoot); err != nil {
		s.Health.Fail("counters", err)
	} else {
		s.Health.OK("counters")
	}
}

func (s *Scanner) watchOnce(coreRoot string, paths func(scanRoots []string) ([]string, error), seen *watchSeen) {
	if s == nil || seen == nil {
		return
	}
	if paths == nil {
		paths = repoPaths
	}
	overlay, err := LoadOverlay(coreRoot)
	if err != nil {
		s.Health.Fail("scan", err)
		return
	}
	scanRoots, _ := EffectiveScanRoots(overlay)
	if len(scanRoots) == 0 {
		return
	}
	found, err := paths(scanRoots)
	if err != nil {
		s.Health.Fail("scan", err)
		return
	}
	sort.Strings(found)
	fingerprint := strings.Join(scanRoots, "\n") + "\n\n" + strings.Join(found, "\n")
	if seen.ready && seen.fingerprint == fingerprint {
		return
	}

	s.mu.Lock()
	if s.status.State == ScanScanning {
		s.mu.Unlock()
		return
	}
	s.status = ScanStatus{
		State:     ScanScanning,
		Root:      strings.Join(scanRoots, "\n"),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	sniff := s.sniff
	reload := s.reload
	s.mu.Unlock()

	s.run(scanRoots, filepath.Join(coreRoot, "_registry"), sniff, reload)
	if status := s.Status(); status.State != ScanSucceeded {
		msg := status.Error
		if msg == "" {
			msg = "scan ended in state " + status.State
		}
		s.Health.Fail("scan", errors.New(msg))
		return
	}
	s.Health.OK("scan")
	seen.ready = true
	seen.fingerprint = fingerprint
}

func repoPaths(scanRoots []string) ([]string, error) {
	var found []string
	for _, root := range scanRoots {
		paths, err := projectscan.RepoPaths(root)
		if err != nil {
			return nil, err
		}
		found = append(found, paths...)
	}
	return found, nil
}
