package managerskills

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:skills
var skillFS embed.FS

// Names is the Manager skill set, in teaching order.
var Names = []string{
	"intake",
	"shape-task",
	"resolve-project",
	"route",
	"triage-attention",
	"review",
	"briefing",
}

// Install writes the Manager skills into a data root. The one copy of each
// skill is .agents/skills/<name>/SKILL.md. Claude, Codex and Gemini get a stub
// in their own directory that names the canon, and Cursor gets a rule that
// includes it. The files belong to the App: one that differs is replaced.
func Install(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("data root is required")
	}
	for _, name := range Names {
		body, err := skillFS.ReadFile(embedPath(name))
		if err != nil {
			return err
		}
		desc := descriptionOf(body)
		targets := map[string][]byte{
			filepath.Join(root, ".agents", "skills", name, "SKILL.md"):      body,
			filepath.Join(root, ".claude", "skills", name, "SKILL.md"):      []byte(stub(name, desc)),
			filepath.Join(root, ".codex", "skills", name, "SKILL.md"):       []byte(stub(name, desc)),
			filepath.Join(root, ".gemini", "skills", name, "SKILL.md"):      []byte(stub(name, desc)),
			filepath.Join(root, ".cursor", "rules", "manager-"+name+".mdc"): []byte(cursorRule(name, desc)),
		}
		for path, data := range targets {
			if err := writeManaged(path, data); err != nil {
				return fmt.Errorf("install manager skill %s: %w", name, err)
			}
		}
	}
	return nil
}

// Description is the one-line frontmatter description of an embedded skill.
func Description(name string) (string, error) {
	body, err := skillFS.ReadFile(embedPath(name))
	if err != nil {
		return "", err
	}
	return descriptionOf(body), nil
}

// Read returns the canonical skill body from a data root.
func Read(root, name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(CanonPath(name))))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CanonPath is the data-root relative path of a skill.
func CanonPath(name string) string {
	return ".agents/skills/" + name + "/SKILL.md"
}

// EmbeddedNames lists the skill directories in the embed.
func EmbeddedNames() ([]string, error) {
	entries, err := fs.ReadDir(skillFS, "skills")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func embedPath(name string) string {
	return "skills/" + name + "/SKILL.md"
}

func stub(name, desc string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\nThe skill is `%s`. Read it and follow it. This file only points there.\n", name, desc, CanonPath(name))
}

func cursorRule(name, desc string) string {
	return fmt.Sprintf("---\ndescription: %s\nalwaysApply: false\n---\n\n@%s\n", desc, CanonPath(name))
}

func descriptionOf(body []byte) string {
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "description:"); ok {
			return strings.TrimSpace(rest)
		}
	}
	return "Manager skill"
}

// writeManaged skips a file that is already current. Otherwise it writes a temp
// file and renames it, so a reader never sees half a skill.
func writeManaged(path string, data []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return err
	}
	return os.Rename(name, path)
}
