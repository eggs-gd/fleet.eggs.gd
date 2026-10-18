package markdown

import "strings"

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
