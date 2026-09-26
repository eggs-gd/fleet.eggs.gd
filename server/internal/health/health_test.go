package health

import (
	"errors"
	"testing"
)

func TestFailAndRecover(t *testing.T) {
	m := New()
	boom := errors.New("disk full")
	if got := m.Fail("audit", boom); got != boom {
		t.Fatalf("Fail must return the error unchanged, got %v", got)
	}
	first, ok := m.Has("audit")
	if !ok || first.Error != "disk full" {
		t.Fatalf("problem = %+v %v", first, ok)
	}
	m.Fail("audit", errors.New("disk full"))
	again, _ := m.Has("audit")
	if again.Since != first.Since {
		t.Fatal("same failure restarted the clock")
	}
	if len(m.Problems()) != 1 {
		t.Fatalf("problems = %v", m.Problems())
	}
	m.OK("audit")
	if _, ok := m.Has("audit"); ok {
		t.Fatal("recovered component still failing")
	}
}

func TestNilMonitorIsHarmless(t *testing.T) {
	var m *Monitor
	err := errors.New("x")
	if m.Fail("a", err) != err {
		t.Fatal("nil monitor changed the error")
	}
	m.OK("a")
	if len(m.Problems()) != 0 {
		t.Fatal("nil monitor has problems")
	}
}
