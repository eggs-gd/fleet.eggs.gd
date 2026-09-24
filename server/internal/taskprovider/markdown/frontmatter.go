package markdown

import (
	"fmt"
	"strconv"
	"strings"
)

func splitMarkdown(text string) (string, string, error) {
	if !strings.HasPrefix(text, "---\n") {
		return "", "", fmt.Errorf("task markdown frontmatter is missing")
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return "", "", fmt.Errorf("task markdown frontmatter is not closed")
	}
	end += 4
	body := strings.TrimLeft(text[end+4:], "\n\r\t ")
	return text[4:end] + "\n", body, nil
}

func setFrontmatterValue(frontmatter string, key string, value string) (string, error) {
	if strings.Contains(key, ".") {
		return "", fmt.Errorf("nested frontmatter edits are not supported for %q", key)
	}
	lines := strings.Split(strings.TrimRight(frontmatter, "\n"), "\n")
	for i, line := range lines {
		currentKey, _, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(currentKey) == key && !strings.HasPrefix(line, "  ") {
			lines[i] = key + ": " + quoteFrontmatter(value)
			return strings.Join(lines, "\n") + "\n", nil
		}
	}
	return strings.Join(append(lines, key+": "+quoteFrontmatter(value)), "\n") + "\n", nil
}

func setFrontmatterRepositories(frontmatter string, repositories []string) (string, error) {
	return setFrontmatterStringList(frontmatter, "repositories", repositories)
}

func setFrontmatterDependsOn(frontmatter string, dependsOn []string) (string, error) {
	return setFrontmatterStringList(frontmatter, "depends_on", dependsOn)
}

func setFrontmatterStringList(frontmatter string, key string, values []string) (string, error) {
	lines := strings.Split(strings.TrimRight(frontmatter, "\n"), "\n")
	start := -1
	end := -1
	for i, line := range lines {
		currentKey, _, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, "  ") {
			continue
		}
		if strings.TrimSpace(currentKey) != key {
			continue
		}
		start = i
		end = i + 1
		for end < len(lines) {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				end++
				continue
			}
			if strings.HasPrefix(lines[end], "  ") || strings.HasPrefix(trimmed, "- ") {
				end++
				continue
			}
			break
		}
		break
	}

	replacement := []string{key + ": []"}
	if len(values) > 0 {
		replacement = make([]string, 0, 1+len(values))
		replacement = append(replacement, key+":")
		for _, value := range values {
			replacement = append(replacement, "  - "+quoteFrontmatter(value))
		}
	}

	if start < 0 {
		return strings.Join(append(lines, replacement...), "\n") + "\n", nil
	}
	updated := append([]string{}, lines[:start]...)
	updated = append(updated, replacement...)
	updated = append(updated, lines[end:]...)
	return strings.Join(updated, "\n") + "\n", nil
}

func quoteFrontmatter(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.ContainsAny(value, ":#[]{}\"'") {
		return strconv.Quote(value)
	}
	return value
}

func appendReviewComment(body string, author string, text string) string {
	body = strings.TrimRight(body, "\n")
	comment := formatComment(author, text)
	if !strings.Contains(body, "\n## Review Comments") && !strings.HasPrefix(body, "## Review Comments") {
		return body + "\n\n## Review Comments\n\n" + comment + "\n"
	}

	lines := strings.Split(body, "\n")
	insertAt := len(lines)
	for i, line := range lines {
		if strings.TrimSpace(line) != "## Review Comments" {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "## ") {
				insertAt = j
				break
			}
		}
		break
	}

	updated := append([]string{}, lines[:insertAt]...)
	if len(updated) > 0 && strings.TrimSpace(updated[len(updated)-1]) != "" {
		updated = append(updated, "")
	}
	updated = append(updated, comment)
	updated = append(updated, lines[insertAt:]...)
	return strings.TrimRight(strings.Join(updated, "\n"), "\n") + "\n"
}
