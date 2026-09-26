package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/health"
)

func TestMaintainReportsBrokenCountersAndRecovers(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "_registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	counters := filepath.Join(root, "_registry", "counters.json")
	if err := os.WriteFile(counters, []byte("{oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	mon := health.New()
	scanner := NewScanner(nil, nil)
	scanner.Health = mon

	scanner.maintain(root, true)
	if _, failing := mon.Has("counters"); !failing {
		t.Fatal("invalid counters.json was not reported")
	}
	if _, failing := mon.Has("workspace"); failing {
		t.Fatal("an unscanned registry must not count as a failure")
	}

	if err := os.WriteFile(counters, []byte(`{"next_work_ref": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	scanner.maintain(root, true)
	if _, failing := mon.Has("counters"); failing {
		t.Fatal("counters still failing after the file was fixed")
	}
}

func TestMaintainSkipsCounterCheckBetweenFullPasses(t *testing.T) {
	root := t.TempDir()
	scanner := NewScanner(nil, nil)
	scanner.maintain(root, false)
	if _, err := os.Stat(filepath.Join(root, "_registry", "counters.json")); err == nil {
		t.Fatal("counters were checked on a light pass")
	}
	scanner.maintain(root, true)
	if _, err := os.Stat(filepath.Join(root, "_registry", "counters.json")); err != nil {
		t.Fatalf("full pass did not create counters: %v", err)
	}
}
