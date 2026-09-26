package managerskills

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallWritesCanonAndPointerStubs(t *testing.T) {
	root := t.TempDir()
	if err := Install(root); err != nil {
		t.Fatalf("Install: %v", err)
	}
	for _, name := range Names {
		canonRel := filepath.Join(".agents", "skills", name, "SKILL.md")
		canon, err := os.ReadFile(filepath.Join(root, canonRel))
		if err != nil {
			t.Fatalf("canon %s: %v", name, err)
		}
		if bytes.Contains(canon, []byte("MANAGER.md")) || bytes.Contains(canon, []byte("INDEX.md")) || bytes.Contains(canon, []byte("_registry")) {
			t.Errorf("%s still points at a board document", name)
		}
		want := ".agents/skills/" + name + "/SKILL.md"
		for _, rel := range []string{
			filepath.Join(".claude", "skills", name, "SKILL.md"),
			filepath.Join(".codex", "skills", name, "SKILL.md"),
			filepath.Join(".gemini", "skills", name, "SKILL.md"),
		} {
			body, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Fatalf("stub %s: %v", rel, err)
			}
			if !bytes.Contains(body, []byte(want)) || bytes.Contains(body, canon) {
				t.Errorf("%s must point at %s and not repeat it:\n%s", rel, want, body)
			}
			if !bytes.Contains(body, []byte("name: "+name)) {
				t.Errorf("%s lacks the skill name", rel)
			}
		}
		rulePath := filepath.Join(root, ".cursor", "rules", "manager-"+name+".mdc")
		rule, err := os.ReadFile(rulePath)
		if err != nil {
			t.Fatalf("cursor rule %s: %v", name, err)
		}
		if !bytes.Contains(rule, []byte("alwaysApply: false")) || !bytes.Contains(rule, []byte("@"+want)) {
			t.Errorf("cursor rule for %s = %s", name, rule)
		}
	}
}

func TestInstallSkipsCurrentFilesAndReplacesEditedOnesWithoutBackups(t *testing.T) {
	root := t.TempDir()
	if err := Install(root); err != nil {
		t.Fatal(err)
	}
	canon := filepath.Join(root, ".agents", "skills", "intake", "SKILL.md")
	original, _ := os.ReadFile(canon)
	info, _ := os.Stat(canon)

	if err := Install(root); err != nil {
		t.Fatal(err)
	}
	again, _ := os.Stat(canon)
	if !again.ModTime().Equal(info.ModTime()) {
		t.Error("a current file was rewritten")
	}

	if err := os.WriteFile(canon, []byte("edited by hand"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install(root); err != nil {
		t.Fatal(err)
	}
	restored, _ := os.ReadFile(canon)
	if !bytes.Equal(restored, original) {
		t.Fatalf("canon was not restored: %q", restored)
	}
	backups, _ := filepath.Glob(filepath.Join(root, ".agents", "skills", "intake", "*"))
	if len(backups) != 1 {
		t.Fatalf("extra files next to the skill: %v", backups)
	}
}
