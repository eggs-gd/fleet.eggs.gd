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
	block, rest, ok := splitEnvelope(text)
	if !ok {
		return Frontmatter{}, text
	}
	raw := strings.Split(block, "\n")
	body := strings.TrimLeft(strings.TrimPrefix(rest, "\n---"), "\n\r\t ")

	fm := Frontmatter{}
	currentKey := ""
	for _, line := range raw {
		if strings.TrimSpace(line) == "" {
			continue
		}
		currentKey = parseFrontmatterLine(fm, currentKey, line)
	}
	return fm, body
}

// parseFrontmatterLine reads one non-blank frontmatter line into fm and returns
// the top-level key that following indented lines belong to.
func parseFrontmatterLine(fm Frontmatter, currentKey string, line string) string {
	switch {
	case currentKey != "" && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "  - "):
		if key, value, ok := strings.Cut(strings.TrimSpace(line), ":"); ok {
			value = strings.TrimSpace(value)
			if value != "" {
				value = ParseScalar(value)
			}
			fm[currentKey+"."+strings.TrimSpace(key)] = value
		}
		return currentKey
	case currentKey != "" && strings.HasPrefix(line, "  - "):
		values, _ := fm[currentKey].([]string)
		fm[currentKey] = append(values, ParseScalar(strings.TrimPrefix(line, "  - ")))
		return currentKey
	}
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return currentKey
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if value == "" {
		fm[key] = []string{}
	} else {
		fm[key] = ParseScalar(value)
	}
	return key
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
