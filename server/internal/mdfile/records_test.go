package mdfile

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestListRecordsMissingKeyReturnsNil(t *testing.T) {
	records, err := ListRecords("---\nid: a\n---\n", "milestones")
	if err != nil || records != nil {
		t.Fatalf("ListRecords = %#v, %v", records, err)
	}
}

func TestListRecordsEmptyBlock(t *testing.T) {
	records, err := ListRecords("---\nid: a\nmilestones:\n---\n", "milestones")
	if err != nil || records != nil {
		t.Fatalf("ListRecords = %#v, %v", records, err)
	}
}

func TestListRecordsOneAndMultiple(t *testing.T) {
	text := "---\nid: a\nmilestones:\n  - id: v1-launch\n    title: v1 launch\n    state: active\n  - id: smart-setup\n    title: \"Smart setup\"\n    state: parked\nstatus: old\n---\n\nbody\n"
	records, err := ListRecords(text, "milestones")
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{
		{"id": "v1-launch", "title": "v1 launch", "state": "active"},
		{"id": "smart-setup", "title": "Smart setup", "state": "parked"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("ListRecords = %#v, want %#v", records, want)
	}
}

func TestListRecordsRejectsTextWithoutABlock(t *testing.T) {
	if _, err := ListRecords("no frontmatter", "milestones"); !errors.Is(err, ErrNoFrontmatter) {
		t.Fatalf("err = %v", err)
	}
}

func TestSetRecordsRoundTripsInFieldOrder(t *testing.T) {
	text := "---\nid: a\nstatus: old\n---\n\nbody\n"
	fields := []string{"id", "title", "version", "state"}
	records := []map[string]string{
		{"id": "v1-launch", "title": "v1 launch", "state": "active"},
		{"id": "smart-setup", "title": "Smart setup", "version": "v0.2.0", "state": "parked"},
	}
	got, err := SetRecords(text, "milestones", fields, records)
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nid: a\nstatus: old\nmilestones:\n  - id: v1-launch\n    title: v1 launch\n    state: active\n  - id: smart-setup\n    title: Smart setup\n    version: v0.2.0\n    state: parked\n---\n\nbody\n"
	if got != want {
		t.Fatalf("SetRecords =\n%q\nwant\n%q", got, want)
	}

	roundTripped, err := ListRecords(got, "milestones")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundTripped, records) {
		t.Fatalf("round trip = %#v, want %#v", roundTripped, records)
	}
}

func TestSetRecordsReplacesExistingBlock(t *testing.T) {
	text := "---\nid: a\nmilestones:\n  - id: old\n    state: parked\nstatus: old\n---\n\nbody\n"
	got, err := SetRecords(text, "milestones", []string{"id", "state"}, []map[string]string{
		{"id": "new", "state": "active"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nid: a\nmilestones:\n  - id: new\n    state: active\nstatus: old\n---\n\nbody\n"
	if got != want {
		t.Fatalf("SetRecords = %q, want %q", got, want)
	}
}

func TestSetRecordsEmptyList(t *testing.T) {
	got, err := SetRecords("---\nid: a\n---\n", "milestones", []string{"id"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "---\nid: a\nmilestones:\n---\n" {
		t.Fatalf("SetRecords = %q", got)
	}
}

func TestSetRecordsRejectsTextWithoutABlock(t *testing.T) {
	if _, err := SetRecords("no frontmatter", "milestones", []string{"id"}, nil); !errors.Is(err, ErrNoFrontmatter) {
		t.Fatalf("err = %v", err)
	}
}

func TestSetRecordsRoundTripsValuesThatNeedQuoting(t *testing.T) {
	values := []string{
		"Fix: login",
		`say "hi"`,
		"# not a comment",
		"a\n---\nfoo: bar",
		"  padded  ",
		"- leading dash",
		`back\slash`,
	}
	for _, value := range values {
		text := "---\nid: a\n---\n\nbody\n"
		got, err := SetRecords(text, "milestones", []string{"id", "title"}, []map[string]string{
			{"id": "m", "title": value},
		})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(got, "\n---") != 1 {
			t.Fatalf("value %q broke the frontmatter envelope:\n%s", value, got)
		}
		records, err := ListRecords(got, "milestones")
		if err != nil || len(records) != 1 || records[0]["title"] != value {
			t.Fatalf("value %q round-tripped to %#v, %v\n%s", value, records, err, got)
		}
		if !strings.HasSuffix(got, "---\n\nbody\n") {
			t.Fatalf("body was disturbed for %q:\n%s", value, got)
		}
	}
}
