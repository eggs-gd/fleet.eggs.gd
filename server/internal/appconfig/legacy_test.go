package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPathPrefersTheCurrentNameAndFallsBackToTheLegacyOne(t *testing.T) {
	touch := func(root, name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name  string
		files []string
		want  string
	}{
		{"neither exists: the current name, to be created", nil, "fleet.yaml"},
		{"only the legacy file", []string{"core.yaml"}, "core.yaml"},
		{"only the current file", []string{"fleet.yaml"}, "fleet.yaml"},
		{"both: the current one wins", []string{"core.yaml", "fleet.yaml"}, "fleet.yaml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, f := range tc.files {
				touch(root, f)
			}
			if got, want := ReadPath(root, "fleet.yaml", "core.yaml"), filepath.Join(root, tc.want); got != want {
				t.Fatalf("ReadPath = %q, want %q", got, want)
			}
		})
	}
}
