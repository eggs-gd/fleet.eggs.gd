package mdfile

import (
	"strconv"
	"strings"
	"unicode"
)

// ListRecords reads key as a block of records, each starting with a "  - "
// line and continuing over more-indented sibling-field lines, e.g.
//
//	milestones:
//	  - id: v1-launch
//	    title: v1 launch
//	    state: active
//	  - id: smart-project-setup
//	    title: Smart project setup
//	    state: parked
//
// Field values come back unquoted. Returns nil, nil when key is absent.
func ListRecords(text, key string) ([]map[string]string, error) {
	lines, _, err := splitFrontmatter(text)
	if err != nil {
		return nil, err
	}
	start, end := keyRange(lines, key)
	if start < 0 {
		return nil, nil
	}
	var records []map[string]string
	var current map[string]string
	for _, l := range lines[start+1 : end+1] {
		if item, ok := strings.CutPrefix(l, "  - "); ok {
			current = map[string]string{}
			records = append(records, current)
			setRecordField(current, item)
			continue
		}
		if current == nil {
			continue
		}
		setRecordField(current, strings.TrimSpace(l))
	}
	return records, nil
}

func setRecordField(record map[string]string, line string) {
	field, value, ok := strings.Cut(line, ":")
	if !ok {
		return
	}
	record[strings.TrimSpace(field)] = unquoteRecordValue(value)
}

// unquoteRecordValue reverses quoteRecordValue: a double-quoted value is
// unescaped with strconv so quotes, colons and newlines round-trip; anything
// else falls back to unquote.
func unquoteRecordValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	return unquote(value)
}

// quoteRecordValue double-quotes a value that a plain YAML scalar could not
// carry safely — a colon, '#', quotes, flow/indicator characters, edge
// whitespace, or any control character (a newline would otherwise end the
// frontmatter early).
func quoteRecordValue(value string) string {
	needs := strings.ContainsAny(value, `:#[]{}"'\`) ||
		strings.ContainsFunc(value, unicode.IsControl) ||
		value != strings.TrimSpace(value)
	if !needs && value != "" {
		needs = strings.ContainsRune("-?&*!|>%@`", rune(value[0]))
	}
	if !needs {
		return value
	}
	return strconv.Quote(value)
}

// SetRecords replaces key with a block list of records, each written in the
// given field order — map iteration order is not stable in Go, so callers
// pass fields explicitly, the same way SetList/AddListItem already take
// explicit items rather than reflecting over a struct. A field absent from a
// record is omitted from its block. Values are quoted as needed, so any string
// round-trips through ListRecords.
func SetRecords(text, key string, fields []string, records []map[string]string) (string, error) {
	lines, rest, err := splitFrontmatter(text)
	if err != nil {
		return "", err
	}
	block := []string{key + ":"}
	for _, record := range records {
		prefix := "  - "
		for _, field := range fields {
			value, ok := record[field]
			if !ok {
				continue
			}
			block = append(block, prefix+field+": "+quoteRecordValue(value))
			prefix = "    "
		}
	}
	if start, end := keyRange(lines, key); start >= 0 {
		lines = append(lines[:start], append(block, lines[end+1:]...)...)
	} else {
		lines = append(lines, block...)
	}
	return joinFrontmatter(lines, rest), nil
}
