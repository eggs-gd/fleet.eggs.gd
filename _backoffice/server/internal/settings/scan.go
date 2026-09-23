package settings

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrScanBusy = errors.New("scan already running")

const (
	ScanIdle      = "idle"
	ScanScanning  = "scanning"
	ScanSucceeded = "succeeded"
	ScanFailed    = "failed"
)

type ScanStatus struct {
	State           string `json:"state"`
	Root            string `json:"root,omitempty"`
	StartedAt       string `json:"started_at,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
	Error           string `json:"error,omitempty"`
	RepositoryCount int    `json:"repository_count,omitempty"`
}

type Scanner struct {
	mu     sync.Mutex
	status ScanStatus
	sniff  func(scanRoot, registryDir string) error
	reload func() error
}

func NewScanner(sniff func(scanRoot, registryDir string) error, reload func() error) *Scanner {
	return &Scanner{
		status: ScanStatus{State: ScanIdle},
		sniff:  sniff,
		reload: reload,
	}
}

func (s *Scanner) Status() ScanStatus {
	if s == nil {
		return ScanStatus{State: ScanIdle}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Scanner) Start(scanRoot, registryDir string) error {
	if s == nil {
		return fmt.Errorf("scanner is not configured")
	}
	s.mu.Lock()
	if s.status.State == ScanScanning {
		s.mu.Unlock()
		return ErrScanBusy
	}
	s.status = ScanStatus{
		State:     ScanScanning,
		Root:      scanRoot,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	sniff := s.sniff
	reload := s.reload
	s.mu.Unlock()

	go s.run(scanRoot, registryDir, sniff, reload)
	return nil
}

func (s *Scanner) run(scanRoot, registryDir string, sniff func(scanRoot, registryDir string) error, reload func() error) {
	var err error
	if sniff == nil {
		err = fmt.Errorf("sniff runner is not configured")
	} else {
		err = sniff(scanRoot, registryDir)
	}
	if err == nil && reload != nil {
		err = reload()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		s.status.State = ScanFailed
		s.status.Error = err.Error()
		return
	}
	s.status.State = ScanSucceeded
	s.status.Error = ""
}

func DefaultSniff(coreRoot string) func(scanRoot, registryDir string) error {
	script := filepath.Join(coreRoot, "_backoffice", "scripts", "sniff_projects.py")
	return func(scanRoot, registryDir string) error {
		cmd := exec.Command("python3", script, "--root", scanRoot, "--registry-dir", registryDir)
		cmd.Dir = coreRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if len(msg) > 2000 {
				msg = msg[:2000] + "…"
			}
			if msg == "" {
				return fmt.Errorf("sniff_projects.py: %w", err)
			}
			return fmt.Errorf("sniff_projects.py: %w\n%s", err, msg)
		}
		return nil
	}
}
