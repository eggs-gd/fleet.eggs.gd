package projecttag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateReadableFirst(t *testing.T) {
	cases := map[string]string{
		"core-eggs-gd": "CORE",
		"career":       "CARE",
		"hdzp-site":    "HDZP",
		"Fleet":        "FLEE",
	}
	for name, want := range cases {
		if got := Generate(name, nil); got != want {
			t.Errorf("Generate(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestGenerateAvoidsTakenTagsWithReadableFallbacks(t *testing.T) {
	taken := map[string]bool{"EGGS": true}
	if got := Generate("eggs-gd-prod", taken); got != "EGPG" {
		t.Fatalf("initials fallback = %q", got)
	}
	taken["EGPG"] = true
	got := Generate("eggs-gd-prod", taken)
	if got == "" || taken[got] || !Valid(got) {
		t.Fatalf("second fallback = %q", got)
	}
}

func TestGenerateAlwaysFindsAFreeTagAndIsStable(t *testing.T) {
	taken := map[string]bool{}
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		tag := Generate("project", taken)
		if tag == "" || seen[tag] || !Valid(tag) || len(tag) != Length {
			t.Fatalf("round %d: tag %q (duplicate=%v)", i, tag, seen[tag])
		}
		seen[tag] = true
		taken[tag] = true
	}
	if a, b := Generate("same", map[string]bool{"SAME": true}), Generate("same", map[string]bool{"SAME": true}); a != b {
		t.Fatalf("not stable: %q vs %q", a, b)
	}
}

func TestGenerateShortAndOddNames(t *testing.T) {
	for _, name := range []string{"pfd", "z2m", "123", "", "a", "гтд"} {
		tag := Generate(name, nil)
		if !Valid(tag) || len(tag) != Length {
			t.Errorf("Generate(%q) = %q", name, tag)
		}
	}
	if got := Generate("pfd", nil); !strings.HasPrefix(got, "PFD") {
		t.Errorf("short name lost its letters: %q", got)
	}
}

func TestValid(t *testing.T) {
	for tag, want := range map[string]bool{"FLET": true, "CORE": true, "A1": true, "AB12CD": true, "INBOX": false, "F": false, "1ABC": false, "flet": false, "TOOLONGX": false, "AB-C": false} {
		if Valid(tag) != want {
			t.Errorf("Valid(%q) = %v", tag, !want)
		}
	}
}

func TestForProject(t *testing.T) {
	root := t.TempDir()
	write := func(project, body string) {
		dir := filepath.Join(root, "Work", project)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "PROJECT.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a", "---\nid: a\ntag: \"FLET\"\n---\n")
	write("b", "---\nid: b\n---\n")
	write("c", "---\nid: c\ntag: flet\n---\n")
	if tag, err := ForProject(root, "a"); err != nil || tag != "FLET" {
		t.Fatalf("a = %q, %v", tag, err)
	}
	if _, err := ForProject(root, "b"); err == nil || !strings.Contains(err.Error(), "no tag yet") {
		t.Fatalf("b = %v", err)
	}
	if _, err := ForProject(root, "c"); err == nil || !strings.Contains(err.Error(), "capital letters") {
		t.Fatalf("c = %v", err)
	}
	if _, err := ForProject(root, "missing"); err == nil {
		t.Fatal("missing card accepted")
	}
}
