// Package projecttag owns the short tag that prefixes a project's task refs
// (FLET-12). A tag lives in the project card's frontmatter as `tag`. The scanner
// generates one for a project that has none, and the operator may change it by
// editing the card.
package projecttag

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
)

// Inbox is the tag of the global Inbox ref space. No project may use it.
const Inbox = "INBOX"

// Length is the length of a generated tag. A hand-written tag may be 2 to 6
// characters.
const Length = 4

var pattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,5}$`)

// Valid reports whether tag can prefix task refs.
func Valid(tag string) bool {
	return pattern.MatchString(tag) && tag != Inbox
}

// FromCard reads the tag out of a project card's text. It returns "" when the
// card has none.
func FromCard(text string) string {
	fm, _ := mdfile.ParseFrontmatter(text)
	return strings.TrimSpace(mdfile.Scalar(fm, "tag", ""))
}

// ForProject reads the tag of Work/<project>/PROJECT.md. A project without a
// valid tag is an error: the scanner assigns tags, and one is added by hand
// with `tag:` in the card.
func ForProject(root, project string) (string, error) {
	path := filepath.Join(root, "Work", project, "PROJECT.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read project card for %s: %w", project, err)
	}
	tag := FromCard(string(raw))
	if tag == "" {
		return "", fmt.Errorf("project %s has no tag yet; the scanner assigns one on its next pass, or add `tag:` to %s", project, path)
	}
	if !Valid(tag) {
		return "", fmt.Errorf("project %s has tag %q in %s, which is not 2-6 capital letters or digits starting with a letter", project, tag, path)
	}
	return tag, nil
}

// Generate picks a Length-letter tag for name that is not in taken. It tries
// readable forms of the name first (its first letters, its initials, its
// consonants, and so on). When they are all taken it falls back to a sequence of
// letters seeded by the name, so the same name always gets the same tag for the
// same set of taken tags. It returns "" only if every 4-letter tag is taken.
func Generate(name string, taken map[string]bool) string {
	words := splitWords(name)
	all := strings.Join(words, "")
	var candidates []string
	add := func(s string) {
		s = strings.ToUpper(s)
		if len(s) == Length {
			candidates = append(candidates, s)
		}
	}
	rng := seeded(name)
	if n := len(all); n >= Length {
		add(all[:Length])
		add(initials(words, all))
		add(consonants(all))
		add(words[0][:min(2, len(words[0]))] + words[len(words)-1][:min(2, len(words[len(words)-1]))])
		add(all[:3] + all[n-1:])
	} else if n >= 2 {
		add(all + rng.letters(Length-n))
	}
	for _, c := range candidates {
		if !taken[c] && Valid(c) {
			return c
		}
	}
	for i := 0; i < 26*26*26*26; i++ {
		if c := rng.letters(Length); !taken[c] && Valid(c) {
			return c
		}
	}
	return ""
}

// splitWords lower-cases name and returns its runs of ASCII letters.
func splitWords(name string) []string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return words
}

// initials is the first letter of each word, filled up to Length with the
// remaining letters in order.
func initials(words []string, all string) string {
	var b strings.Builder
	for _, w := range words {
		b.WriteByte(w[0])
	}
	out := b.String()
	if len(out) >= Length {
		return out[:Length]
	}
	for _, w := range words {
		for i := 1; i < len(w) && len(out) < Length; i++ {
			out += string(w[i])
		}
	}
	return out
}

// consonants is the first letter followed by the next consonants, filled with
// vowels when the name has fewer than Length-1 of them.
func consonants(all string) string {
	out := all[:1]
	for _, r := range all[1:] {
		if len(out) < Length && !strings.ContainsRune("aeiou", r) {
			out += string(r)
		}
	}
	for _, r := range all[1:] {
		if len(out) < Length && strings.ContainsRune("aeiou", r) {
			out += string(r)
		}
	}
	return out
}

type letterSource struct{ state uint64 }

func seeded(name string) *letterSource {
	h := fnv.New64a()
	h.Write([]byte(name))
	s := h.Sum64()
	if s == 0 {
		s = 1
	}
	return &letterSource{state: s}
}

func (l *letterSource) next() uint64 {
	l.state ^= l.state << 13
	l.state ^= l.state >> 7
	l.state ^= l.state << 17
	return l.state
}

func (l *letterSource) letters(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(byte('A' + l.next()%26))
	}
	return b.String()
}
