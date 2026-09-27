package workfiles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var testNow = time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

func newRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "_registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	counters := `{"work_ref_prefix":"CORE","next_work_ref":1,"inbox_ref_prefix":"INBOX","next_inbox_ref":10}`
	if err := os.WriteFile(filepath.Join(root, "_registry", "counters.json"), []byte(counters), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCaptureNumbersAndPromote(t *testing.T) {
	root := newRoot(t)
	ref, path, err := CaptureInbox(root, "look at billing\nsecond line", testNow)
	if err != nil || ref != "INBOX-10" || filepath.Base(path) != "2026-08-01-look-at-billing.md" {
		t.Fatalf("first capture = %q %q, %v", ref, path, err)
	}
	if ref, _, _ := CaptureInbox(root, "look at billing", testNow); ref != "INBOX-11" {
		t.Fatalf("second capture = %q", ref)
	}
	item, err := ReadInbox(root, "inbox-10")
	if err != nil || item.Body != "look at billing\nsecond line" || item.Status != "untriaged" || len(item.PromotedTo) != 0 {
		t.Fatalf("read = %+v, %v", item, err)
	}
	if err := LinkInboxToTask(root, "INBOX-10", "CORE-7", testNow); err != nil {
		t.Fatal(err)
	}
	// One capture can spawn several tasks: a second, different link appends
	// instead of replacing.
	if err := LinkInboxToTask(root, "INBOX-10", "CORE-8", testNow); err != nil {
		t.Fatal(err)
	}
	// Linking the same task again is a no-op: no duplicate entry, no extra log line.
	if err := LinkInboxToTask(root, "INBOX-10", "CORE-7", testNow); err != nil {
		t.Fatal(err)
	}
	item, err = ReadInbox(root, "INBOX-10")
	if err != nil || strings.Join(item.PromotedTo, ",") != "CORE-7,CORE-8" || item.Body != "look at billing\nsecond line" || item.Status != "promoted" {
		t.Fatalf("after linking = %+v, %v", item, err)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{"status: promoted", "promoted_to:\n  - CORE-7\n  - CORE-8", "promoted to CORE-7", "promoted to CORE-8"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("linked file lacks %q:\n%s", want, raw)
		}
	}
	if strings.Count(string(raw), "promoted to CORE-7") != 1 {
		t.Errorf("relinking the same task duplicated its log line:\n%s", raw)
	}
	if _, err := ReadInbox(root, "INBOX-99"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing = %v", err)
	}
	items, _ := ListInbox(root)
	var refs []string
	for _, i := range items {
		refs = append(refs, i.Ref)
	}
	if strings.Join(refs, ",") != "INBOX-10,INBOX-11" {
		t.Fatalf("list = %v", refs)
	}
	if items[0].Preview != "look at billing" || items[0].Status != "promoted" {
		t.Fatalf("list item summary = %+v", items[0])
	}
	if items[1].Status != "untriaged" || len(items[1].PromotedTo) != 0 {
		t.Fatalf("unpromoted list item = %+v", items[1])
	}
	counters, _ := os.ReadFile(filepath.Join(root, "_registry", "counters.json"))
	if !strings.Contains(string(counters), `"INBOX": 12`) {
		t.Fatalf("counter not advanced: %s", counters)
	}
}

func TestCaptureCreatesTheCountersFileFromTheTree(t *testing.T) {
	root := t.TempDir()
	ref, _, err := CaptureInbox(root, "first note", testNow)
	if err != nil || ref != "INBOX-1" {
		t.Fatalf("capture = %q, %v", ref, err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "_registry", "counters.json"))
	if err != nil || !strings.Contains(string(raw), `"INBOX": 2`) {
		t.Fatalf("counters = %s, %v", raw, err)
	}
}

func TestCaptureWithNonASCIITitleStillGetsAFile(t *testing.T) {
	root := newRoot(t)
	_, path, err := CaptureInbox(root, "глянути білінг", testNow)
	if err != nil || filepath.Base(path) != "2026-08-01-note.md" {
		t.Fatalf("path = %q, %v", path, err)
	}
}

func TestConcurrentCaptureNeverSharesANumber(t *testing.T) {
	root := newRoot(t)
	var wg sync.WaitGroup
	refs := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ref, _, err := CaptureInbox(root, "note", testNow)
			if err != nil {
				t.Error(err)
				return
			}
			refs <- ref
		}()
	}
	wg.Wait()
	close(refs)
	seen := map[string]bool{}
	for ref := range refs {
		if seen[ref] {
			t.Fatalf("%s handed out twice", ref)
		}
		seen[ref] = true
	}
	if len(seen) != 20 {
		t.Fatalf("captured %d items", len(seen))
	}
}

func TestProjectAliases(t *testing.T) {
	root := t.TempDir()
	if _, _, err := AddProjectAlias(root, "acme", "x"); err == nil {
		t.Fatal("wrote an alias for a project without a card")
	}
	dir := filepath.Join(root, "Work", "acme")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(dir, "PROJECT.md")
	if err := os.WriteFile(card, []byte("---\nid: acme\naliases:\n  - existing\nstatus: active\n---\n\n# Acme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		_, wrote, err := AddProjectAlias(root, "acme", "Billing")
		if err != nil || wrote != (i == 0) {
			t.Fatalf("alias write %d = %v, %v", i, wrote, err)
		}
	}
	got, _ := os.ReadFile(card)
	if !strings.Contains(string(got), "aliases:\n  - existing\n  - Billing\nstatus: active\n---") {
		t.Fatalf("aliases not in frontmatter:\n%s", got)
	}
	if strings.Contains(string(got), "Activity Log") {
		t.Fatalf("an alias must not add a log entry:\n%s", got)
	}
	aliases, err := ProjectAliases(root, "acme")
	if err != nil || strings.Join(aliases, ",") != "existing,Billing" {
		t.Fatalf("aliases = %v, %v", aliases, err)
	}
	if aliases, err := ProjectAliases(root, "nope"); err != nil || aliases != nil {
		t.Fatalf("missing card aliases = %v, %v", aliases, err)
	}
}

func TestDefaultAssignee(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Work", "acme")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "PROJECT.md"), []byte("---\nid: acme\nassignment:\n  default_assignee: \"Codex\"\n---\n\ndefault_assignee: cursor\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DefaultAssignee(root, "acme"); got != "codex" {
		t.Fatalf("assignee = %q", got)
	}
	if got := DefaultAssignee(root, "../acme"); got != "" {
		t.Fatalf("traversal = %q", got)
	}
}
