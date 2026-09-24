// Package mdfile parses the small YAML-ish frontmatter + Markdown body shape
// used by every Core file (task cards, workspace cards). It intentionally
// knows nothing about tasks, workspaces, or lifecycle rules — those live in
// their own packages (tasklifecycle, corechain) and call into mdfile only for
// generic parsing.
package mdfile

import (
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Whitespace matches one or more whitespace characters, for collapsing
// multi-line Markdown text into a single line.
var Whitespace = regexp.MustCompile(`\s+`)

// Frontmatter is the parsed YAML-ish header of a Core Markdown file. Scalar
// values are strings; list values (`key:\n  - a\n  - b`) are []string.
type Frontmatter map[string]any

// ReadMarkdown reads a file and splits it into frontmatter and body.
func ReadMarkdown(path string) (Frontmatter, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	fm, body := ParseFrontmatter(string(data))
	return fm, body, nil
}

// ParseFrontmatter splits `---\n...\n---\n<body>` text into a Frontmatter map
// and the remaining body. Text without a frontmatter block is returned as a
// body with an empty Frontmatter.
func ParseFrontmatter(text string) (Frontmatter, string) {
	if !strings.HasPrefix(text, "---\n") {
		return Frontmatter{}, text
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return Frontmatter{}, text
	}
	end += 4
	raw := strings.Split(text[4:end], "\n")
	body := strings.TrimLeft(text[end+4:], "\n\r\t ")

	fm := Frontmatter{}
	currentKey := ""
	for _, line := range raw {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "  ") && currentKey != "" && !strings.HasPrefix(line, "  - ") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if value == "" {
				fm[currentKey+"."+key] = ""
				continue
			}
			fm[currentKey+"."+key] = ParseScalar(value)
			continue
		}
		if strings.HasPrefix(line, "  - ") && currentKey != "" {
			values, _ := fm[currentKey].([]string)
			fm[currentKey] = append(values, ParseScalar(strings.TrimPrefix(line, "  - ")))
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		currentKey = key
		if value == "" {
			fm[key] = []string{}
			continue
		}
		fm[key] = ParseScalar(value)
	}
	return fm, body
}

// ParseScalar unquotes a YAML-ish scalar value.
func ParseScalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			return unquoted
		}
		return value[1 : len(value)-1]
	}
	return value
}

// Scalar returns the string value for key, or fallback when absent/empty.
func Scalar(fm Frontmatter, key string, fallback string) string {
	if value, ok := fm[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

// List returns the []string value for key, or an empty slice when absent.
func List(fm Frontmatter, key string) []string {
	values, ok := fm[key].([]string)
	if !ok {
		return []string{}
	}
	return values
}

// Boolean returns true only when the raw scalar value is the literal "true".
func Boolean(fm Frontmatter, key string) bool {
	value, ok := fm[key].(string)
	return ok && value == "true"
}

// IntValue parses key as an integer, returning fallback when absent or
// unparsable.
func IntValue(fm Frontmatter, key string, fallback int) int {
	value, err := strconv.Atoi(Scalar(fm, key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

// FirstParagraph returns the first non-heading, non-empty paragraph of a
// Markdown body, collapsed onto a single line.
func FirstParagraph(body string) string {
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(lines) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		lines = append(lines, trimmed)
	}
	return strings.TrimSpace(Whitespace.ReplaceAllString(strings.Join(lines, " "), " "))
}
