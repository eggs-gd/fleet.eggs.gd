package settings

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/eventbus"
	"github.com/eggs-gd/fleet.eggs.gd/internal/health"
	"github.com/eggs-gd/fleet.eggs.gd/internal/projectscan"
)

var (
	ErrScanBusy         = errors.New("scan already running")
	ErrScanRootRequired = errors.New("scan root is required")
)

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
	// Health, when set, is told about failing scans and workspace repairs.
	Health *health.Monitor
	// Publish, when set, is told when a project appears, its repositories
	// change, or it drops out of the registry. In-process only: it is not
	// replayed, so a project missing when serve starts is not reported.
	Publish func(eventbus.Event) error

	mu     sync.Mutex
	status ScanStatus
	sniff  func(scanRoots []string, registryDir string) error
	reload func() error
	// known is the project id set from the previous maintain pass, used only
	// to notice one going missing. It is memory, not a durable record.
	known map[string]bool
}

func NewScanner(sniff func(scanRoots []string, registryDir string) error, reload func() error) *Scanner {
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

func (s *Scanner) Start(scanRoots []string, registryDir string) error {
	if s == nil {
		return fmt.Errorf("scanner is not configured")
	}
	scanRoots = normalizeScanRoots(scanRoots)
	if len(scanRoots) == 0 {
		return ErrScanRootRequired
	}
	s.mu.Lock()
	if s.status.State == ScanScanning {
		s.mu.Unlock()
		return ErrScanBusy
	}
	s.status = ScanStatus{
		State:     ScanScanning,
		Root:      strings.Join(scanRoots, "\n"),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	sniff := s.sniff
	reload := s.reload
	s.mu.Unlock()

	go s.run(scanRoots, registryDir, sniff, reload)
	return nil
}

func (s *Scanner) run(scanRoots []string, registryDir string, sniff func(scanRoots []string, registryDir string) error, reload func() error) {
	var err error
	if sniff == nil {
		err = fmt.Errorf("sniff runner is not configured")
	} else {
		err = sniff(scanRoots, registryDir)
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

func DefaultSniff(coreRoot string) func(scanRoots []string, registryDir string) error {
	return func(scanRoots []string, registryDir string) error {
		_, err := projectscan.Scan(projectscan.ScanOptions{
			ScanRoots:   scanRoots,
			RegistryDir: registryDir,
			WorkDir:     filepath.Join(coreRoot, "Work"),
		})
		return err
	}
}
