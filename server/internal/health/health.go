// Package health records which parts of the running system are failing. A
// failing write returns its error to whoever asked for it and lands here too,
// so the failure stays visible after the request is gone. The next success for
// the same component clears it.
package health

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// Problem is one failing component.
type Problem struct {
	Component string `json:"component"`
	Error     string `json:"error"`
	Since     string `json:"since"`
}

// Monitor is safe for concurrent use. A nil Monitor ignores everything.
type Monitor struct {
	mu       sync.Mutex
	problems map[string]Problem
}

// New returns an empty monitor.
func New() *Monitor {
	return &Monitor{problems: map[string]Problem{}}
}

// Fail marks component as failing and returns err unchanged. The first report
// for a component, or a changed error, is also printed to stderr.
func (m *Monitor) Fail(component string, err error) error {
	if m == nil || err == nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	prev, had := m.problems[component]
	if had && prev.Error == err.Error() {
		return err
	}
	since := time.Now().UTC().Format(time.RFC3339)
	if had {
		since = prev.Since
	}
	m.problems[component] = Problem{Component: component, Error: err.Error(), Since: since}
	fmt.Fprintf(os.Stderr, "degraded: %s: %v\n", component, err)
	return err
}

// OK clears component.
func (m *Monitor) OK(component string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, had := m.problems[component]; had {
		delete(m.problems, component)
		fmt.Fprintf(os.Stderr, "recovered: %s\n", component)
	}
}

// Has reports whether component is failing.
func (m *Monitor) Has(component string) (Problem, bool) {
	if m == nil {
		return Problem{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.problems[component]
	return p, ok
}

// Problems lists every failing component, sorted by name.
func (m *Monitor) Problems() []Problem {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Problem, 0, len(m.problems))
	for _, p := range m.problems {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Component < out[j].Component })
	return out
}
