package mdfile

import (
	"strings"
	"sync"
)

// EditMu serializes read-modify-write cycles on Markdown files that more than
// one part of the server edits, such as a project card (the scanner adds its
// tag while a Manager tool adds an alias). Hold it from the read to the write.
var EditMu sync.Mutex

// The functions below are one-shot forms of Doc for callers that edit a
// single key of a text they already hold.

// SetScalar replaces key with "key: value", or adds it before the closing marker.
func SetScalar(text, key, value string) (string, error) {
	d, err := ParseDoc(text)
	if err != nil {
		return "", err
	}
	d.SetRaw(key, value)
	return d.String(), nil
}

// SetList replaces key with a block list of items, written as given.
func SetList(text, key string, items []string) (string, error) {
	d, err := ParseDoc(text)
	if err != nil {
		return "", err
	}
	d.SetItems(key, items)
	return d.String(), nil
}

// SetFirstParagraph replaces the first non-heading, non-empty paragraph after
// the frontmatter block; see Doc.SetFirstParagraph.
func SetFirstParagraph(text, paragraph string) (string, error) {
	d, err := ParseDoc(text)
	if err != nil {
		return "", err
	}
	if err := d.SetFirstParagraph(paragraph); err != nil {
		return "", err
	}
	return d.String(), nil
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
	d, err := ParseDoc(text)
	if err != nil {
		return nil, err
	}
	return d.Items(key), nil
}

// AddListItem appends item to key's list, turning an inline list into a block
// list first. It reports false when item is already there (case-insensitive).
func AddListItem(text, key, item string) (string, bool, error) {
	d, err := ParseDoc(text)
	if err != nil {
		return "", false, err
	}
	items := d.Items(key)
	for _, have := range items {
		if strings.EqualFold(have, item) {
			return text, false, nil
		}
	}
	d.SetItems(key, append(items, item))
	return d.String(), true, nil
}
