package tasklifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var workRefCounterMu sync.Mutex

// LegacyWorkPrefix is the prefix every task ref had before projects got their
// own tags. Refs that carry it stay valid.
const LegacyWorkPrefix = "CORE"

// InboxPrefix is the prefix of the global Inbox ref space.
const InboxPrefix = "INBOX"

var refLine = regexp.MustCompile(`(?m)^ref:\s*["']?([A-Za-z][A-Za-z0-9]*)-(\d+)["']?\s*$`)

// counters is _registry/counters.json: the next number to hand out per ref
// prefix. The legacy fields are read so an older file migrates, and are never
// written back.
type counters struct {
	Next  map[string]int `json:"next"`
	Notes []string       `json:"notes,omitempty"`

	LegacyWorkPrefix  string `json:"work_ref_prefix,omitempty"`
	LegacyNextWork    int    `json:"next_work_ref,omitempty"`
	LegacyInboxPrefix string `json:"inbox_ref_prefix,omitempty"`
	LegacyNextInbox   int    `json:"next_inbox_ref,omitempty"`
}

func (c *counters) migrate() {
	if c.Next == nil {
		c.Next = map[string]int{}
	}
	if c.LegacyNextWork > 0 {
		prefix := strings.ToUpper(strings.TrimSpace(c.LegacyWorkPrefix))
		if prefix == "" {
			prefix = LegacyWorkPrefix
		}
		if c.Next[prefix] < c.LegacyNextWork {
			c.Next[prefix] = c.LegacyNextWork
		}
	}
	if c.LegacyNextInbox > 0 {
		prefix := strings.ToUpper(strings.TrimSpace(c.LegacyInboxPrefix))
		if prefix == "" {
			prefix = InboxPrefix
		}
		if c.Next[prefix] < c.LegacyNextInbox {
			c.Next[prefix] = c.LegacyNextInbox
		}
	}
	c.LegacyWorkPrefix, c.LegacyNextWork, c.LegacyInboxPrefix, c.LegacyNextInbox = "", 0, "", 0
}

func countersPath(root string) string {
	return filepath.Join(root, "_registry", "counters.json")
}

// readCounters loads the file. A missing file is reported as missing, and a
// file that is not valid JSON is an error that names it.
func readCounters(root string) (c counters, missing bool, err error) {
	data, err := os.ReadFile(countersPath(root))
	if errors.Is(err, fs.ErrNotExist) {
		c.migrate()
		return c, true, nil
	}
	if err != nil {
		return c, false, fmt.Errorf("read counters: %w", err)
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, false, fmt.Errorf("counters file %s is not valid JSON, fix it by hand: %w", countersPath(root), err)
	}
	c.migrate()
	return c, false, nil
}

func writeCounters(root string, c counters) error {
	encoded, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := writeCountersAtomic(countersPath(root), append(encoded, '\n')); err != nil {
		return fmt.Errorf("write counters: %w", err)
	}
	return nil
}

// AllocateNextRef hands out the next ref for prefix, such as FLET-12, and
// records the new counter. A prefix the file has not seen yet starts above the
// highest ref of that prefix already in the tree.
func AllocateNextRef(root, prefix string) (string, error) {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	if !refPrefix.MatchString(prefix) {
		return "", fmt.Errorf("ref prefix %q is not valid", prefix)
	}
	workRefCounterMu.Lock()
	defer workRefCounterMu.Unlock()

	c, _, err := readCounters(root)
	if err != nil {
		return "", err
	}
	next, known := c.Next[prefix]
	if !known || next < 1 {
		highest, err := highestRefs(root)
		if err != nil {
			return "", err
		}
		next = highest[prefix] + 1
	}
	c.Next[prefix] = next + 1
	if err := writeCounters(root, c); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", prefix, next), nil
}

var refPrefix = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,5}$`)

// AllocateNextInboxRef allocates the next INBOX-N ref.
func AllocateNextInboxRef(root string) (string, error) {
	return AllocateNextRef(root, InboxPrefix)
}

// EnsureCounters keeps _registry/counters.json usable. It creates the file when
// it is missing, migrates an older layout, and raises every counter that is not
// above the highest ref of its prefix already in the tree, so a ref is never
// handed out twice. A file that is not valid JSON is left alone and reported:
// that is data a person has to look at. It reports whether the file changed.
func EnsureCounters(root string) (bool, error) {
	workRefCounterMu.Lock()
	defer workRefCounterMu.Unlock()

	c, missing, err := readCounters(root)
	if err != nil {
		return false, err
	}
	before := fmt.Sprint(sortedCounters(c.Next))
	legacy := false
	if data, readErr := os.ReadFile(countersPath(root)); readErr == nil {
		legacy = strings.Contains(string(data), "next_work_ref") || strings.Contains(string(data), "next_inbox_ref") || strings.Contains(string(data), "next_life_ref")
	}
	if c.Next[InboxPrefix] < 1 {
		c.Next[InboxPrefix] = 1
	}
	highest, err := highestRefs(root)
	if err != nil {
		return false, err
	}
	for prefix, n := range highest {
		if refPrefix.MatchString(prefix) && n >= c.Next[prefix] {
			c.Next[prefix] = n + 1
		}
	}
	if len(c.Notes) == 0 || legacy {
		c.Notes = []string{
			"Task refs are TAG-N, where TAG is the tag in the project's PROJECT.md. INBOX-N is the global Inbox.",
			"Refs are monotonically increasing and must not be reused.",
		}
	}
	if !missing && !legacy && before == fmt.Sprint(sortedCounters(c.Next)) {
		return false, nil
	}
	if err := writeCounters(root, c); err != nil {
		return false, err
	}
	return true, nil
}

func sortedCounters(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+strconv.Itoa(v))
	}
	sort.Strings(out)
	return out
}

// highestRefs returns the largest number seen per upper-cased ref prefix in
// the frontmatter of Work/**.md and Inbox/items/*.md.
func highestRefs(root string) (map[string]int, error) {
	highest := map[string]int{}
	for _, dir := range []string{"Work", filepath.Join("Inbox", "items")} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
				return nil
			}
			head, err := frontmatterOf(path)
			if err != nil {
				return err
			}
			for _, m := range refLine.FindAllStringSubmatch(head, -1) {
				n, convErr := strconv.Atoi(m[2])
				if convErr != nil {
					continue
				}
				if prefix := strings.ToUpper(m[1]); n > highest[prefix] {
					highest[prefix] = n
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return highest, nil
}

func frontmatterOf(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return "", nil
	}
	text := string(buf[:n])
	if !strings.HasPrefix(text, "---\n") {
		return "", nil
	}
	head, _, _ := strings.Cut(text[4:], "\n---")
	return head, nil
}
