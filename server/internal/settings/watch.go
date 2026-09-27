package settings

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/eventbus"
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
	report, err := projectscan.MaintainCards(filepath.Join(coreRoot, "_registry"), filepath.Join(coreRoot, "Work"))
	switch {
	case err != nil:
		s.Health.Fail("workspace", err)
	case len(report.Problems) > 0:
		s.Health.Fail("workspace", errors.New(strings.Join(report.Problems, "; ")))
	default:
		s.Health.OK("workspace")
	}
	if err == nil {
		s.publishProjectEvents(coreRoot, report)
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

// publishProjectEvents tells the bus about a project appearing, its
// repositories changing, or it dropping out of the registry. "Missing" is
// edge-triggered against s.known, kept only in memory: it fires once, when a
// project first drops out, not on every later tick it stays gone.
func (s *Scanner) publishProjectEvents(coreRoot string, report projectscan.CardReport) {
	if s.Publish == nil {
		return
	}
	now := make(map[string]bool, len(report.Known))
	for _, id := range report.Known {
		now[id] = true
	}
	s.mu.Lock()
	previous := s.known
	s.known = now
	s.mu.Unlock()

	publish := func(kind string, id string) {
		_ = s.Publish(eventbus.Event{
			Channel: eventbus.ChannelProject,
			Type:    "project." + kind,
			Text:    "Project " + id + " " + strings.ReplaceAll(kind, "_", " ") + ".",
			Fields: map[string]string{
				"project_id":   id,
				"project_path": filepath.Join(coreRoot, "Work", id, "PROJECT.md"),
			},
		})
	}
	for _, id := range report.Created {
		publish("discovered", id)
	}
	for _, id := range report.ReposChanged {
		publish("repos_changed", id)
	}
	if previous != nil {
		for id := range previous {
			if !now[id] {
				publish("missing", id)
			}
		}
	}
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
