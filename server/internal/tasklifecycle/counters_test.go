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

func TestEnsureCountersCreatesFromExistingRefs(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"Work/acme/tasks/a.md":      "---\nid: a\nref: CORE-41\n---\n",
		"Work/acme/tasks/b.md":      "---\nid: b\nref: \"CORE-7\"\n---\n",
		"Work/_life/ideas/c.md":     "---\nref: LIFE-3\n---\n",
		"Inbox/items/2026-a.md":     "---\nref: INBOX-9\n---\n",
		"Work/acme/tasks/README.md": "no frontmatter ref: CORE-999\n",
		"Work/acme/notes/n.md":      "---\ntitle: mentions CORE-500 in body\n---\nref: CORE-800\n",
	})
	changed, err := EnsureCounters(root)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	ref, err := AllocateNextWorkRef(root)
	if err != nil || ref != "CORE-42" {
		t.Fatalf("work ref = %q, %v", ref, err)
	}
	inbox, err := AllocateNextInboxRef(root)
	if err != nil || inbox != "INBOX-10" {
		t.Fatalf("inbox ref = %q, %v", inbox, err)
	}
	if changed, _ := EnsureCounters(root); changed {
		t.Fatal("a second pass rewrote a healthy file")
	}
}

func TestEnsureCountersFillsMissingFieldsAndRaisesLowOnes(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"_registry/counters.json": `{"work_ref_prefix":"CORE","next_work_ref":3}`,
		"Work/acme/tasks/a.md":    "---\nref: CORE-10\n---\n",
	})
	if changed, err := EnsureCounters(root); err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "_registry", "counters.json"))
	for _, want := range []string{`"next_work_ref": 11`, `"next_inbox_ref": 1`, `"inbox_ref_prefix": "INBOX"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("counters lack %s:\n%s", want, raw)
		}
	}
}

func TestEnsureCountersLeavesInvalidJSONAlone(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"_registry/counters.json": `{"next_work_ref": `})
	if _, err := EnsureCounters(root); err == nil {
		t.Fatal("invalid JSON was accepted")
	}
	raw, _ := os.ReadFile(filepath.Join(root, "_registry", "counters.json"))
	if string(raw) != `{"next_work_ref": ` {
		t.Fatalf("invalid file was rewritten: %s", raw)
	}
}
