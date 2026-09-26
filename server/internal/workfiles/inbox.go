// Package workfiles owns the Markdown files the Manager tools write outside
// the task board: Inbox items and the project card.
package workfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

var inboxRefPattern = regexp.MustCompile(`(?i)^INBOX-(\d+)$`)

// ErrNotFound is returned when an Inbox item does not exist.
var ErrNotFound = errors.New("not found")

// InboxItem is one captured note.
type InboxItem struct {
	Ref        string
	Path       string
	Body       string
	PromotedTo string
}

// IsInboxRef reports whether ref looks like INBOX-<n>.
func IsInboxRef(ref string) bool {
	return inboxRefPattern.MatchString(strings.TrimSpace(ref))
}

func inboxDir(root string) string {
	return filepath.Join(root, "Inbox", "items")
}

// ListInbox returns the refs of every captured item, sorted by number.
func ListInbox(root string) ([]string, error) {
	items, err := scanInbox(root)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].num < items[j].num })
	refs := make([]string, 0, len(items))
	for _, item := range items {
		refs = append(refs, item.ref)
	}
	return refs, nil
}

// CaptureInbox stores text as a new Inbox item with the next INBOX ref from
// _registry/counters.json. The file is created exclusively, so two captures
// never share a path.
func CaptureInbox(root, text string, now time.Time) (ref, path string, err error) {
	text = strings.TrimSpace(text)
	dir := inboxDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	ref, err = tasklifecycle.AllocateNextInboxRef(root)
	if err != nil {
		return "", "", err
	}
	slug := tasklifecycle.SlugifyTitle(firstLine(text))
	if slug == "" {
		slug = "note"
	}
	day := now.Format("2006-01-02")
	stamp := now.Format(time.RFC3339)
	for i := 1; ; i++ {
		base := day + "-" + slug
		if i > 1 {
			base = fmt.Sprintf("%s-%d", base, i)
		}
		body := fmt.Sprintf("---\nid: inbox-%s\nref: %s\nstatus: untriaged\nsource: manager\ncreated_at: %s\nupdated_at: %s\npromoted_to:\n---\n\n## Raw Input\n\n%s\n\n## Activity Log\n\n- %s captured\n",
			base, ref, stamp, stamp, text, stamp)
		path = filepath.Join(dir, base+".md")
		f, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(openErr, os.ErrExist) {
			continue
		}
		if openErr != nil {
			return "", "", openErr
		}
		_, writeErr := f.WriteString(body)
		closeErr := f.Close()
		if writeErr != nil {
			return "", "", writeErr
		}
		return ref, path, closeErr
	}
}

// ReadInbox loads one item by ref.
func ReadInbox(root, ref string) (InboxItem, error) {
	ref = strings.ToUpper(strings.TrimSpace(ref))
	if !IsInboxRef(ref) {
		return InboxItem{}, fmt.Errorf("%q is not an INBOX ref", ref)
	}
	items, err := scanInbox(root)
	if err != nil {
		return InboxItem{}, err
	}
	for _, item := range items {
		if item.ref != ref {
			continue
		}
		fm, body, err := mdfile.ReadMarkdown(item.path)
		if err != nil {
			return InboxItem{}, err
		}
		return InboxItem{
			Ref:        ref,
			Path:       item.path,
			Body:       rawInput(body),
			PromotedTo: mdfile.Scalar(fm, "promoted_to", ""),
		}, nil
	}
	return InboxItem{}, ErrNotFound
}

// MarkPromoted records the task an item became.
func MarkPromoted(root, ref, taskRef string, now time.Time) error {
	item, err := ReadInbox(root, ref)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(item.Path)
	if err != nil {
		return err
	}
	stamp := now.Format(time.RFC3339)
	text := string(raw)
	for key, value := range map[string]string{"status": "promoted", "promoted_to": taskRef, "updated_at": stamp} {
		if text, err = mdfile.SetScalar(text, key, value); err != nil {
			return fmt.Errorf("%s: %w", item.Path, err)
		}
	}
	if !strings.Contains(text, "## Activity Log") {
		text = strings.TrimRight(text, "\n") + "\n\n## Activity Log\n"
	}
	text = strings.TrimRight(text, "\n") + fmt.Sprintf("\n- %s promoted to %s\n", stamp, taskRef)
	return writeAtomic(item.Path, []byte(text))
}

type inboxFile struct {
	ref  string
	num  int
	path string
}

func scanInbox(root string) ([]inboxFile, error) {
	dir := inboxDir(root)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var items []inboxFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		fm, _, err := mdfile.ReadMarkdown(path)
		if err != nil {
			return nil, err
		}
		ref := strings.ToUpper(mdfile.Scalar(fm, "ref", ""))
		match := inboxRefPattern.FindStringSubmatch(ref)
		if len(match) != 2 {
			continue
		}
		num, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, fmt.Errorf("%s: bad ref %q", path, ref)
		}
		items = append(items, inboxFile{ref: ref, num: num, path: path})
	}
	return items, nil
}

// rawInput returns the "## Raw Input" section, or the whole body when the
// item has no such section.
func rawInput(body string) string {
	_, after, ok := strings.Cut(body, "## Raw Input")
	if !ok {
		return strings.TrimSpace(body)
	}
	if before, _, found := strings.Cut(after, "\n## "); found {
		after = before
	}
	return strings.TrimSpace(after)
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return strings.TrimSpace(line)
}
