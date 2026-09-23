package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

func TestAppendEventWritesNDJSON(t *testing.T) {
	root := t.TempDir()
	if err := AppendEvent(root, Event{Type: "first"}); err != nil { t.Fatal(err) }
	if err := AppendEvent(root, Event{Type: "second"}); err != nil { t.Fatal(err) }
	file, err := os.Open(EventLogPath(root)); if err != nil { t.Fatal(err) }
	defer file.Close()
	var events []Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() { var event Event; if err := json.Unmarshal(scanner.Bytes(), &event); err != nil { t.Fatal(err) }; events = append(events, event) }
	if err := scanner.Err(); err != nil { t.Fatal(err) }
	if len(events) != 2 || events[0].Time == "" { t.Fatalf("events = %#v", events) }
}
