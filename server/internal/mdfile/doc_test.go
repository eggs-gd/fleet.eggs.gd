package mdfile

import "testing"

// The expectations below were captured from the task provider's former private
// frontmatter editor, so folding it into Doc cannot change how task cards are
// rewritten.
const taskCard = "---\nid: a\nstatus: todo\nproject: p\nrepositories:\n  - old/repo\ndepends_on: []\nlaunch:\n  agent: x\nassignee: codex\n---\n\n# Body\n"

func edit(t *testing.T, change func(*Doc)) string {
	t.Helper()
	d, err := ParseDoc(taskCard)
	if err != nil {
		t.Fatal(err)
	}
	change(d)
	return d.String()
}

func TestDocScalarEdits(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Doc)
		want   string
	}{
		{"replace", func(d *Doc) { d.SetString("status", "doing") },
			"---\nid: a\nstatus: doing\nproject: p\nrepositories:\n  - old/repo\ndepends_on: []\nlaunch:\n  agent: x\nassignee: codex\n---\n\n# Body\n"},
		{"append", func(d *Doc) { d.SetString("milestone", "v1") },
			"---\nid: a\nstatus: todo\nproject: p\nrepositories:\n  - old/repo\ndepends_on: []\nlaunch:\n  agent: x\nassignee: codex\nmilestone: v1\n---\n\n# Body\n"},
		{"quotes a colon", func(d *Doc) { d.SetString("title", "a: b") },
			"---\nid: a\nstatus: todo\nproject: p\nrepositories:\n  - old/repo\ndepends_on: []\nlaunch:\n  agent: x\nassignee: codex\ntitle: \"a: b\"\n---\n\n# Body\n"},
		{"trims", func(d *Doc) { d.SetString("status", "  x  ") },
			"---\nid: a\nstatus: x\nproject: p\nrepositories:\n  - old/repo\ndepends_on: []\nlaunch:\n  agent: x\nassignee: codex\n---\n\n# Body\n"},
		{"remove", func(d *Doc) { d.Remove("status") },
			"---\nid: a\nproject: p\nrepositories:\n  - old/repo\ndepends_on: []\nlaunch:\n  agent: x\nassignee: codex\n---\n\n# Body\n"},
		{"remove missing is a no-op", func(d *Doc) { d.Remove("zzz") }, taskCard},
	}
	for _, c := range cases {
		if got := edit(t, c.change); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestDocListEdits(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Doc)
		want   string
	}{
		{"replace with quoting", func(d *Doc) { d.SetStrings("repositories", []string{"a/b", "c: d"}) },
			"---\nid: a\nstatus: todo\nproject: p\nrepositories:\n  - a/b\n  - \"c: d\"\ndepends_on: []\nlaunch:\n  agent: x\nassignee: codex\n---\n\n# Body\n"},
		{"clear to []", func(d *Doc) { d.SetStrings("repositories", nil) },
			"---\nid: a\nstatus: todo\nproject: p\nrepositories: []\ndepends_on: []\nlaunch:\n  agent: x\nassignee: codex\n---\n\n# Body\n"},
		{"fill an empty []", func(d *Doc) { d.SetStrings("depends_on", []string{"CORE-1"}) },
			"---\nid: a\nstatus: todo\nproject: p\nrepositories:\n  - old/repo\ndepends_on:\n  - CORE-1\nlaunch:\n  agent: x\nassignee: codex\n---\n\n# Body\n"},
		{"keep [] when still empty", func(d *Doc) { d.SetStrings("depends_on", nil) }, taskCard},
	}
	for _, c := range cases {
		if got := edit(t, c.change); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestDocBlockListWithFlushLeftDashes(t *testing.T) {
	d, err := ParseDoc("---\na: 1\nrepositories:\n- x\nz: 2\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	d.SetStrings("repositories", []string{"y"})
	if got, want := d.String(), "---\na: 1\nrepositories:\n  - y\nz: 2\n---\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDocRenderNormalizesTheBodySeparator(t *testing.T) {
	d, err := ParseDoc("---\nid: a\n---\n\n\n\n# Body\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Render(d.Body()); got != "---\nid: a\n---\n\n# Body\n" {
		t.Fatalf("Render = %q", got)
	}
}

func TestDocScalarReadsUnquoted(t *testing.T) {
	d, _ := ParseDoc("---\ntype: \"todo\"\nplain: x\n---\n")
	if v, ok := d.Scalar("type"); !ok || v != "todo" {
		t.Fatalf("type = %q %v", v, ok)
	}
	if _, ok := d.Scalar("missing"); ok {
		t.Fatal("found a missing key")
	}
}

func TestDocListRangeIncludesBlankLinesBetweenItems(t *testing.T) {
	d, err := ParseDoc("---\na: 1\nrepositories:\n  - x\n\n  - y\n\nz: 2\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	d.SetStrings("repositories", []string{"n"})
	// Both old items go; the blank line that only precedes the next key stays.
	if got, want := d.String(), "---\na: 1\nrepositories:\n  - n\n\nz: 2\n---\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if items := (func() []string { d2, _ := ParseDoc("---\nr:\n  - x\n\n  - y\n---\n"); return d2.Items("r") })(); len(items) != 2 {
		t.Fatalf("items across a blank line = %v", items)
	}
}
