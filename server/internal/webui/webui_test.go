package webui

import "testing"

func TestEmbeddedReportsAMissingDashboard(t *testing.T) {
	// A source checkout holds only the placeholder. A build made by `make build`
	// holds index.html. Either way the answer must match what is embedded.
	fsys, ok := Embedded()
	if ok && fsys == nil {
		t.Fatal("Embedded reported a dashboard but returned no filesystem")
	}
	if !ok && fsys != nil {
		t.Fatal("Embedded returned a filesystem for a build without a dashboard")
	}
}
