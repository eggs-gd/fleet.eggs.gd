package settings

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/projectscan"
)

type watchSeen struct {
	ready       bool
	fingerprint string
}

// Watch polls the saved scan roots while serve is running. A full registry
// rewrite runs once at start and again whenever the set of git checkouts changes.
func (s *Scanner) Watch(ctx context.Context, coreRoot string, interval time.Duration) {
	if s == nil || ctx == nil {
		return
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	var seen watchSeen
	s.watchOnce(coreRoot, nil, &seen)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.watchOnce(coreRoot, nil, &seen)
		}
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
		return
	}
	scanRoots, _ := EffectiveScanRoots(overlay)
	if len(scanRoots) == 0 {
		return
	}
	found, err := paths(scanRoots)
	if err != nil {
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
	if s.Status().State != ScanSucceeded {
		return
	}
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
