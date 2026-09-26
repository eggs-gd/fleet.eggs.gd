package tasklifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var refLine = regexp.MustCompile(`(?m)^ref:\s*["']?([A-Za-z]+)-(\d+)["']?\s*$`)

// EnsureCounters keeps _registry/counters.json usable. It creates the file when
// it is missing, fills any counter that is absent or below 1, and raises a
// counter that is not above the highest ref already in the tree, so a ref is
// never handed out twice. A file that is not valid JSON is left alone and
// reported: that is data a person has to look at. It reports whether the file
// changed.
func EnsureCounters(root string) (bool, error) {
	workRefCounterMu.Lock()
	defer workRefCounterMu.Unlock()

	path := filepath.Join(root, "_registry", "counters.json")
	var counters workRefCounters
	data, err := os.ReadFile(path)
	missing := errors.Is(err, fs.ErrNotExist)
	switch {
	case missing:
		counters.Notes = []string{
			"Human-facing refs are monotonically increasing and must not be reused.",
			"Use CORE-N for Work items, INBOX-N for raw Inbox items, and LIFE-N for personal notes, ideas, reminders, and decisions.",
		}
	case err != nil:
		return false, fmt.Errorf("read counters: %w", err)
	default:
		if err := json.Unmarshal(data, &counters); err != nil {
			return false, fmt.Errorf("counters file %s is not valid JSON, fix it by hand: %w", path, err)
		}
	}
	before := counters

	fill := func(prefix *string, fallback string, next *int) {
		if strings.TrimSpace(*prefix) == "" {
			*prefix = fallback
		}
		if *next < 1 {
			*next = 1
		}
	}
	fill(&counters.WorkRefPrefix, "CORE", &counters.NextWorkRef)
	fill(&counters.InboxRefPrefix, "INBOX", &counters.NextInboxRef)
	fill(&counters.LifeRefPrefix, "LIFE", &counters.NextLifeRef)

	highest, err := highestRefs(root)
	if err != nil {
		return false, err
	}
	raise := func(prefix string, next *int) {
		if n := highest[strings.ToUpper(prefix)]; n >= *next {
			*next = n + 1
		}
	}
	raise(counters.WorkRefPrefix, &counters.NextWorkRef)
	raise(counters.InboxRefPrefix, &counters.NextInboxRef)
	raise(counters.LifeRefPrefix, &counters.NextLifeRef)

	if !missing && counters.NextWorkRef == before.NextWorkRef && counters.NextInboxRef == before.NextInboxRef &&
		counters.NextLifeRef == before.NextLifeRef && counters.WorkRefPrefix == before.WorkRefPrefix &&
		counters.InboxRefPrefix == before.InboxRefPrefix && counters.LifeRefPrefix == before.LifeRefPrefix {
		return false, nil
	}
	encoded, err := json.MarshalIndent(counters, "", "  ")
	if err != nil {
		return false, err
	}
	if err := writeCountersAtomic(path, append(encoded, '\n')); err != nil {
		return false, fmt.Errorf("write counters: %w", err)
	}
	return true, nil
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
