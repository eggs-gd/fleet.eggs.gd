package mdfile

import "testing"

func TestSetScalarAndList(t *testing.T) {
	text := "---\nid: a\nrepositories:\n  - x\n  - y\nstatus: old\n---\n\n# Body\n"
	got, err := SetScalar(text, "status", "new")
	if err != nil || got != "---\nid: a\nrepositories:\n  - x\n  - y\nstatus: new\n---\n\n# Body\n" {
		t.Fatalf("SetScalar = %q, %v", got, err)
	}
	got, err = SetList(text, "repositories", []string{`"z"`})
	if err != nil || got != "---\nid: a\nrepositories:\n  - \"z\"\nstatus: old\n---\n\n# Body\n" {
		t.Fatalf("SetList = %q, %v", got, err)
	}
	got, _ = SetScalar(text, "promoted_to", "CORE-1")
	if got != "---\nid: a\nrepositories:\n  - x\n  - y\nstatus: old\npromoted_to: CORE-1\n---\n\n# Body\n" {
		t.Fatalf("SetScalar add = %q", got)
	}
}

func TestAddListItemNormalizesInlineLists(t *testing.T) {
	text := "---\nid: a\naliases: [one, \"two\"]\n---\n\nbody\n"
	got, added, err := AddListItem(text, "aliases", "Three")
	if err != nil || !added || got != "---\nid: a\naliases:\n  - one\n  - two\n  - Three\n---\n\nbody\n" {
		t.Fatalf("AddListItem = %q %v %v", got, added, err)
	}
	if _, added, _ := AddListItem(got, "aliases", "TWO"); added {
		t.Fatal("duplicate alias added")
	}
	empty, _, _ := AddListItem("---\nid: a\naliases: []\n---\n", "aliases", "x")
	if empty != "---\nid: a\naliases:\n  - x\n---\n" {
		t.Fatalf("empty list = %q", empty)
	}
	missing, _, _ := AddListItem("---\nid: a\n---\n", "aliases", "x")
	if missing != "---\nid: a\naliases:\n  - x\n---\n" {
		t.Fatalf("missing key = %q", missing)
	}
}

func TestFrontmatterHelpersRejectTextWithoutABlock(t *testing.T) {
	for _, text := range []string{"# no frontmatter\n", "---\nid: a\nno end\n"} {
		if _, err := SetScalar(text, "a", "b"); err != ErrNoFrontmatter {
			t.Fatalf("%q: err = %v", text, err)
		}
	}
}
