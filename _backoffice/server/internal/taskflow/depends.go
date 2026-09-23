package taskflow

import "strings"

// NormalizeDependsOn trims blanks and drops empty/duplicate entries while
// preserving order. Mirrors tasklifecycle.NormalizeDependsOn so Patch/Create
// stay consistent without importing the lifecycle package.
func NormalizeDependsOn(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, 0, len(refs))
	seen := map[string]bool{}
	for _, raw := range refs {
		ref := strings.TrimSpace(raw)
		if ref == "" {
			continue
		}
		key := strings.ToUpper(ref)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	return out
}
