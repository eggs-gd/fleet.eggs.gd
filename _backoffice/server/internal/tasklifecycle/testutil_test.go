package tasklifecycle

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testTaskMarkdown(ref string, title string, status string) string {
	return testTaskForProjectMarkdown(ref, title, status, "core-eggs-gd", "core.eggs.gd")
}

func testTaskForProjectMarkdown(ref string, title string, status string, project string, repository string) string {
	return `---
schema_version: 1
id: work-test
ref: ` + ref + `
title: "` + title + `"
type: feature
status: ` + status + `
project: ` + project + `
repositories:
  - ` + repository + `
assignee: codex
created_at: 2026-07-31T10:00:00+03:00
updated_at: 2026-07-31T10:00:00+03:00
launch:
---

# ` + title + `
`
}
