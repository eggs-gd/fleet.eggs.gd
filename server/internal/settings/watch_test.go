package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWatchScansWhenRepositoriesChange(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, OverlayFileName), []byte("scanRoot: /proj\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var scans int
	scanner := NewScanner(func([]string, string) error {
		scans++
		return nil
	}, func() error { return nil })

	calls := 0
	paths := func([]string) ([]string, error) {
		calls++
		if calls == 1 {
			return []string{"/proj/a"}, nil
		}
		return []string{"/proj/b", "/proj/a"}, nil
	}
	var seen watchSeen
	scanner.watchOnce(root, paths, &seen)
	scanner.watchOnce(root, paths, &seen)
	if scans != 2 {
		t.Fatalf("scans = %d, want 2", scans)
	}

	scanner.watchOnce(root, func([]string) ([]string, error) {
		return []string{"/proj/a", "/proj/b"}, nil
	}, &seen)
	if scans != 2 {
		t.Fatalf("unchanged scans = %d, want 2", scans)
	}
}

func TestWatchRetriesAfterFailedScan(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, OverlayFileName), []byte("scanRoot: /proj\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fail := true
	var scans int
	scanner := NewScanner(func([]string, string) error {
		scans++
		if fail {
			return fmt.Errorf("boom")
		}
		return nil
	}, nil)
	paths := func([]string) ([]string, error) {
		return []string{"/proj/a"}, nil
	}
	var seen watchSeen
	scanner.watchOnce(root, paths, &seen)
	if scans != 1 || seen.ready {
		t.Fatalf("scans = %d ready = %v", scans, seen.ready)
	}
	fail = false
	scanner.watchOnce(root, paths, &seen)
	if scans != 2 || !seen.ready {
		t.Fatalf("scans = %d ready = %v", scans, seen.ready)
	}
	scanner.watchOnce(root, paths, &seen)
	if scans != 2 {
		t.Fatalf("scans = %d, want 2", scans)
	}
}

func TestWatchScansWhenSecondRootChanges(t *testing.T) {
	root := t.TempDir()
	body := "scanRoots:\n  - /one\n  - /two\n"
	if err := os.WriteFile(filepath.Join(root, OverlayFileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var scans int
	scanner := NewScanner(func([]string, string) error {
		scans++
		return nil
	}, nil)
	current := []string{"/one/a", "/two/b"}
	paths := func([]string) ([]string, error) {
		return append([]string{}, current...), nil
	}
	var seen watchSeen
	scanner.watchOnce(root, paths, &seen)
	scanner.watchOnce(root, paths, &seen)
	if scans != 1 {
		t.Fatalf("scans = %d, want 1", scans)
	}
	current = []string{"/one/a", "/two/c"}
	scanner.watchOnce(root, paths, &seen)
	if scans != 2 {
		t.Fatalf("scans = %d, want 2", scans)
	}
}

func TestWatchSkipsMissingScanRoot(t *testing.T) {
	root := t.TempDir()
	var scans int
	scanner := NewScanner(func([]string, string) error {
		scans++
		return nil
	}, nil)
	var seen watchSeen
	scanner.watchOnce(root, func([]string) ([]string, error) {
		return nil, fmt.Errorf("missing")
	}, &seen)
	if scans != 0 || seen.ready {
		t.Fatalf("scans = %d ready = %v", scans, seen.ready)
	}
}
