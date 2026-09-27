package mdfile

import (
	"errors"
	"strings"
	"sync"
)

// EditMu serializes read-modify-write cycles on Markdown files that more than
// one part of the server edits, such as a project card (the scanner adds its
// tag while a Manager tool adds an alias). Hold it from the read to the write.
var EditMu sync.Mutex

// The helpers below edit the leading frontmatter block of a Markdown text and
// leave everything else byte for byte as it was.

// ErrNoFrontmatter is returned when the text has no complete frontmatter block.
var ErrNoFrontmatter = errors.New("no complete frontmatter block")

func splitFrontmatter(text string) (lines []string, rest string, err error) {
	if !strings.HasPrefix(text, "---\n") {
		return nil, "", ErrNoFrontmatter
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return nil, "", ErrNoFrontmatter
	}
	end += 4
	return strings.Split(text[4:end], "\n"), text[end:], nil
}

func joinFrontmatter(lines []string, rest string) string {
	return "---\n" + strings.Join(lines, "\n") + rest
}

// keyRange finds the lines a key occupies: its own line plus any indented
// continuation lines. It returns -1 when the key is absent.
func keyRange(lines []string, key string) (start, end int) {
	for i, l := range lines {
		if !strings.HasPrefix(l, key+":") {
			continue
		}
		end = i
		for end+1 < len(lines) && strings.HasPrefix(lines[end+1], " ") {
			end++
		}
		return i, end
	}
	return -1, -1
}

// SetScalar replaces key with "key: value", or adds it before the closing marker.
func SetScalar(text, key, value string) (string, error) {
	lines, rest, err := splitFrontmatter(text)
	if err != nil {
		return "", err
	}
	line := key + ": " + value
	if start, end := keyRange(lines, key); start >= 0 {
		lines = append(lines[:start], append([]string{line}, lines[end+1:]...)...)
	} else {
		lines = append(lines, line)
	}
	return joinFrontmatter(lines, rest), nil
}

// SetList replaces key with a block list of items, written as given.
func SetList(text, key string, items []string) (string, error) {
	lines, rest, err := splitFrontmatter(text)
	if err != nil {
		return "", err
	}
	block := []string{key + ":"}
	for _, item := range items {
		block = append(block, "  - "+item)
	}
	if start, end := keyRange(lines, key); start >= 0 {
		lines = append(lines[:start], append(block, lines[end+1:]...)...)
	} else {
		lines = append(lines, block...)
	}
	return joinFrontmatter(lines, rest), nil
}

// SetFirstParagraph replaces the first non-heading, non-empty paragraph after
// the frontmatter block with paragraph, leaving any heading before it and
// anything after the following blank line untouched. It appends the
// paragraph after the frontmatter (and any leading headings) when the body
// has no such paragraph yet.
func SetFirstParagraph(text, paragraph string) (string, error) {
	lines, rest, err := splitFrontmatter(text)
	if err != nil {
		return "", err
	}
	_, body, ok := strings.Cut(rest, "\n---")
	if !ok {
		return "", ErrNoFrontmatter
	}
	return joinFrontmatter(lines, "\n---"+replaceFirstParagraph(body, paragraph)), nil
}

func replaceFirstParagraph(body, paragraph string) string {
	lines := strings.Split(body, "\n")
	start, end := -1, len(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if start >= 0 {
				end = i
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start < 0 {
		trimmed := strings.TrimRight(body, "\n")
		if trimmed == "" {
			return "\n\n" + paragraph
		}
		return trimmed + "\n\n" + paragraph
	}
	next := append([]string{}, lines[:start]...)
	next = append(next, paragraph)
	next = append(next, lines[end:]...)
	return strings.Join(next, "\n")
}

// ListItems reads key as a block list or as an inline [a, b] list. Items come
// back unquoted.
func ListItems(text, key string) ([]string, error) {
	lines, _, err := splitFrontmatter(text)
	if err != nil {
		return nil, err
	}
	start, end := keyRange(lines, key)
	if start < 0 {
		return nil, nil
	}
	var items []string
	if inline := strings.TrimSpace(strings.TrimPrefix(lines[start], key+":")); inline != "" {
		for _, part := range strings.Split(strings.Trim(inline, "[]"), ",") {
			if part = unquote(part); part != "" {
				items = append(items, part)
			}
		}
		return items, nil
	}
	for _, l := range lines[start+1 : end+1] {
		if item, ok := strings.CutPrefix(strings.TrimSpace(l), "-"); ok {
			if item = unquote(item); item != "" {
				items = append(items, item)
			}
		}
	}
	return items, nil
}

// AddListItem appends item to key's list, turning an inline list into a block
// list first. It reports false when item is already there (case-insensitive).
func AddListItem(text, key, item string) (string, bool, error) {
	items, err := ListItems(text, key)
	if err != nil {
		return "", false, err
	}
	for _, have := range items {
		if strings.EqualFold(have, item) {
			return text, false, nil
		}
	}
	next, err := SetList(text, key, append(items, item))
	return next, err == nil, err
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}
