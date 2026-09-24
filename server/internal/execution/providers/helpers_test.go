package providers

import (
	"strings"
	"testing"
)

func TestTruncateForSessionLog(t *testing.T) {
	if got := truncateForSessionLog("  hello  ", 10); got != "hello" {
		t.Fatalf("short = %q", got)
	}
	long := strings.Repeat("x", 50)
	got := truncateForSessionLog(long, 10)
	if got != "xxxxxxxxxx..." {
		t.Fatalf("long = %q", got)
	}
	if got := truncateForSessionLog("", 10); got != "" {
		t.Fatalf("empty = %q", got)
	}
}
