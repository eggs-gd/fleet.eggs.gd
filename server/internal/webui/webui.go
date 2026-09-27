// Package webui holds the dashboard that ships inside the binary. `make ui`
// copies view/dist into dist/ before the Go build. A checkout that has not built
// the dashboard holds only the placeholder, so `go build` and `go test` work
// without Node, and Embedded reports that there is no dashboard.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Embedded returns the dashboard built into this binary. ok is false when the
// binary was built without one.
func Embedded() (fsys fs.FS, ok bool) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
