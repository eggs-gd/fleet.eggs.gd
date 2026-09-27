package tasklifecycle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKnownAssigneeUsesTheFleetRoster(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Fleet"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Sam.md", "ROUTING.md", "README.md", "claude.md"} {
		if err := os.WriteFile(filepath.Join(root, "Fleet", name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, want := range map[string]bool{
		"unassigned": true, "claude": true, "codex": true, "sam": true, "SAM": true,
		"routing": false, "readme": false, "launch_policy": false, "alex": false, "": false, "../x": false,
	} {
		if got := KnownAssignee(root, name); got != want {
			t.Errorf("KnownAssignee(%q) = %v, want %v", name, got, want)
		}
	}
	if RosterPerson(root, "claude") {
		t.Error("an agent is not a roster person")
	}
}

func TestRosterPeopleListsOnlyPeople(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Fleet", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Sam.md", "alex.md", "claude.md", "README.md", "ROUTING.md", "LAUNCH_POLICY.md", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(root, "Fleet", name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := RosterPeople(root)
	if len(got) != 2 || got[0] != "alex" || got[1] != "sam" {
		t.Fatalf("people = %v", got)
	}
	if got := RosterPeople(t.TempDir()); got == nil || len(got) != 0 {
		t.Fatalf("no Fleet folder must give an empty list, got %#v", got)
	}
}
