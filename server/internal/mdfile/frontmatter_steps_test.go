package mdfile

import (
	"reflect"
	"testing"
)

func TestParseFrontmatterLine(t *testing.T) {
	fm, body := ParseFrontmatter("---\nid: a\ntitle: \"Quoted: Title\"\nempty:\ndepends_on:\n  - X-1\n  - \"X-2\"\nmilestones:\n  next: \"v2\"\n  blank:\ngarbage line\n---\n\nBody text\n")
	if body != "Body text\n" {
		t.Fatalf("body = %q", body)
	}
	if fm["id"] != "a" || fm["title"] != "Quoted: Title" {
		t.Fatalf("scalars = %v", fm)
	}
	if v, ok := fm["empty"].([]string); !ok || len(v) != 0 {
		t.Fatalf("empty key must be an empty list, got %#v", fm["empty"])
	}
	if !reflect.DeepEqual(fm["depends_on"], []string{"X-1", "X-2"}) {
		t.Fatalf("list = %#v", fm["depends_on"])
	}
	if fm["milestones.next"] != "v2" || fm["milestones.blank"] != "" {
		t.Fatalf("nested = %v", fm)
	}
	if _, ok := fm["garbage line"]; ok {
		t.Fatal("a line without a colon is not a key")
	}
}

func TestParseFrontmatterWithoutBlock(t *testing.T) {
	fm, body := ParseFrontmatter("just text")
	if len(fm) != 0 || body != "just text" {
		t.Fatalf("got %v %q", fm, body)
	}
}
