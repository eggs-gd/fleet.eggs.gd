package settings

import (
	"os"
	"path/filepath"
	"strings"
)

func fleetPreferredUse(root, agent string) (text, source string) {
	rel := filepath.Join("Fleet", agent+".md")
	path := filepath.Join(root, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", rel
	}
	return extractSection(string(data), "Best At"), rel
}

func fleetRouting(root string) (text, source string) {
	rel := filepath.Join("Fleet", "ROUTING.md")
	path := filepath.Join(root, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", rel
	}
	body := strings.TrimSpace(string(data))
	if len(body) > 2400 {
		body = strings.TrimSpace(body[:2400]) + "\n…"
	}
	return body, rel
}

func extractSection(markdown, heading string) string {
	lines := strings.Split(markdown, "\n")
	start := -1
	want := strings.ToLower(strings.TrimSpace(heading))
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if !strings.HasPrefix(trim, "## ") {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trim, "## ")))
		if start >= 0 {
			break
		}
		if title == want {
			start = i + 1
		}
	}
	if start < 0 {
		return ""
	}
	var body []string
	for _, line := range lines[start:] {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		body = append(body, line)
	}
	return strings.TrimSpace(strings.Join(body, "\n"))
}
