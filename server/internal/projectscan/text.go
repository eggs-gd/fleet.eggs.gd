package projectscan

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var nonAlnumLower = regexp.MustCompile(`[^a-z0-9]+`)
var htmlTag = regexp.MustCompile(`<[^>]+>`)
var mdImage = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
var mdLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
var mdCode = regexp.MustCompile("`([^`]+)`")
var whitespace = regexp.MustCompile(`\s+`)

func utcNow() string {
	return time.Now().UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z07:00")
}

func slugify(value, fallback string) string {
	slug := strings.Trim(strings.ToLower(nonAlnum.ReplaceAllString(value, "-")), "-")
	if slug == "" {
		return fallback
	}
	return slug
}

func shortHash(basis string) string {
	sum := sha1.Sum([]byte(basis))
	return hex.EncodeToString(sum[:])[:12]
}

func normalizeRemote(remote string) string {
	value := strings.TrimSpace(remote)
	value = strings.TrimPrefix(value, "ssh://git@")
	if strings.HasPrefix(value, "https://github.com/") {
		value = "github.com:" + strings.TrimPrefix(value, "https://github.com/")
	}
	if strings.HasPrefix(value, "git@github.com:") {
		value = "github.com:" + strings.TrimPrefix(value, "git@github.com:")
	}
	value = strings.TrimSuffix(value, ".git")
	return strings.ToLower(value)
}

func lastSegment(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	if i := strings.LastIndex(value, "/"); i >= 0 {
		return value[i+1:]
	}
	return value
}

func readText(path string, maxChars int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := string(data)
	if len(text) > maxChars {
		text = text[:maxChars]
	}
	return text
}

func cleanMarkdown(value string) string {
	value = htmlTag.ReplaceAllString(value, " ")
	value = mdImage.ReplaceAllString(value, " ")
	value = mdLink.ReplaceAllString(value, "$1")
	value = mdCode.ReplaceAllString(value, "$1")
	value = whitespace.ReplaceAllString(value, " ")
	return strings.Trim(value, " #*-|")
}

func strPtr(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}

func sortedUnique(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
