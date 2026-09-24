package audit

import "testing"

func TestAppendEventWritesRows(t *testing.T) {
	root := t.TempDir()
	if err := AppendEvent(root, Event{Type: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := AppendEvent(root, Event{Type: "second", Details: map[string]any{"ok": true}}); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEvents(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Time == "" || events[0].Type != "first" || events[1].Type != "second" {
		t.Fatalf("events = %#v", events)
	}
	if events[1].Details["ok"] != true {
		t.Fatalf("details = %#v", events[1].Details)
	}
	text, err := EventLogText(root)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(text, `"type":"first"`) || !contains(text, `"ok":true`) {
		t.Fatalf("event log text = %s", text)
	}
}

func contains(text, part string) bool {
	for i := 0; i+len(part) <= len(text); i++ {
		if text[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
