package tasklifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readCountersFile(t *testing.T, root string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "_registry", "counters.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestEnsureCountersCreatesFromExistingRefsOfEveryPrefix(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"Work/acme/tasks/a.md":      "---\nid: a\nref: CORE-41\n---\n",
		"Work/acme/tasks/b.md":      "---\nid: b\nref: \"FLET-7\"\n---\n",
		"Inbox/items/2026-a.md":     "---\nref: INBOX-9\n---\n",
		"Work/acme/tasks/README.md": "no frontmatter ref: CORE-999\n",
		"Work/acme/notes/n.md":      "---\ntitle: mentions CORE-500 in body\n---\nref: CORE-800\n",
	})
	changed, err := EnsureCounters(root)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	for prefix, want := range map[string]string{"CORE": "CORE-42", "FLET": "FLET-8", "INBOX": "INBOX-10"} {
		if ref, err := AllocateNextRef(root, prefix); err != nil || ref != want {
			t.Fatalf("%s = %q, %v", prefix, ref, err)
		}
	}
	if changed, _ := EnsureCounters(root); changed {
		t.Fatal("a second pass rewrote a healthy file")
	}
}

func TestAllocateNextRefStartsUnknownPrefixesAboveTheTree(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"_registry/counters.json": `{"next":{"CORE":102}}`,
		"Work/acme/tasks/a.md":    "---\nref: NEWP-6\n---\n",
	})
	if ref, err := AllocateNextRef(root, "CORE"); err != nil || ref != "CORE-102" {
		t.Fatalf("known prefix = %q, %v", ref, err)
	}
	if ref, err := AllocateNextRef(root, "NEWP"); err != nil || ref != "NEWP-7" {
		t.Fatalf("unknown prefix = %q, %v", ref, err)
	}
	if ref, err := AllocateNextRef(root, "FRSH"); err != nil || ref != "FRSH-1" {
		t.Fatalf("fresh prefix = %q, %v", ref, err)
	}
	if _, err := AllocateNextRef(root, "bad-prefix"); err == nil {
		t.Fatal("invalid prefix accepted")
	}
	raw := readCountersFile(t, root)
	if !strings.Contains(raw, `"CORE": 103`) {
		t.Fatalf("counters = %s", raw)
	}
}

func TestEnsureCountersLeavesInvalidJSONAlone(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"_registry/counters.json": `{"next": `})
	if _, err := EnsureCounters(root); err == nil {
		t.Fatal("invalid JSON was accepted")
	}
	if _, err := AllocateNextRef(root, "FLET"); err == nil {
		t.Fatal("allocation went ahead on an invalid file")
	}
	if raw := readCountersFile(t, root); raw != `{"next": ` {
		t.Fatalf("invalid file was rewritten: %s", raw)
	}
}
