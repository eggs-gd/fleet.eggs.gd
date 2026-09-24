package projectscan

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func discoverGitRepos(root string) ([]string, error) {
	var repos []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || !entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if path != root {
			if _, pruned := pruneDirs[name]; pruned || name == ".git" {
				return fs.SkipDir
			}
		}
		gitPath := filepath.Join(path, ".git")
		info, err := os.Lstat(gitPath)
		if err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			repos = append(repos, path)
		}
		return nil
	})
	sort.Slice(repos, func(i, j int) bool {
		return strings.ToLower(repos[i]) < strings.ToLower(repos[j])
	})
	return repos, err
}

// RepoPaths lists git checkouts under root. The live server polls this and
// runs a full registry scan only when the set changes.
func RepoPaths(root string) ([]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	abs = resolvePath(abs)
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan root %s is not a directory", abs)
	}
	return discoverGitRepos(abs)
}

func runGit(repo string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func repoID(path, remote string) string {
	remotePart := "no-remote"
	slugBasis := filepath.Base(path)
	if remote != "" {
		remotePart = normalizeRemote(remote)
		slugBasis = lastSegment(remotePart)
	}
	basis := "repo:" + remotePart + "|path:" + resolvePath(path)
	return slugify(slugBasis, "repo") + "-" + shortHash(basis)
}

func remoteIdentity(remote string) string {
	if remote == "" {
		return ""
	}
	normalized := normalizeRemote(remote)
	basis := "remote:" + normalized
	return slugify(lastSegment(normalized), "repo") + "-" + shortHash(basis)
}

func previousRepoID(path, remote string) string {
	var basis, slugBasis string
	if remote != "" {
		normalized := normalizeRemote(remote)
		basis = "remote:" + normalized
		slugBasis = lastSegment(normalized)
	} else {
		basis = "path:" + resolvePath(path)
		slugBasis = filepath.Base(path)
	}
	return slugify(slugBasis, "repo") + "-" + shortHash(basis)
}

func resolvePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}

func nestedRepoRoots(repo string, all []string) map[string]struct{} {
	roots := map[string]struct{}{}
	prefix := repo + string(filepath.Separator)
	for _, candidate := range all {
		if candidate == repo {
			continue
		}
		if strings.HasPrefix(candidate, prefix) {
			roots[candidate] = struct{}{}
		}
	}
	return roots
}

func parentMap(repos []string) map[string]string {
	ordered := append([]string{}, repos...)
	sort.Slice(ordered, func(i, j int) bool {
		return strings.Count(ordered[i], string(filepath.Separator)) < strings.Count(ordered[j], string(filepath.Separator))
	})
	parents := map[string]string{}
	for _, repo := range ordered {
		parent := ""
		for _, candidate := range ordered {
			if candidate == repo {
				continue
			}
			if strings.HasPrefix(repo, candidate+string(filepath.Separator)) && len(candidate) > len(parent) {
				parent = candidate
			}
		}
		parents[repo] = parent
	}
	return parents
}

func summarizeReadme(repo string) (title, summary, source, readmePath string) {
	name := filepath.Base(repo)
	readme := findReadme(repo)
	if readme == "" {
		return name, "No README found. Repository at " + name + ".", "fallback", ""
	}
	lines := make([]string, 0)
	for _, line := range strings.Split(readText(readme, 12000), "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	title = name
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			if cleaned := cleanMarkdown(strings.TrimSpace(strings.TrimLeft(line, "#"))); cleaned != "" {
				title = cleaned
			}
			break
		}
	}
	summary = ""
	inCode := false
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCode = !inCode
			continue
		}
		if inCode || line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[!") {
			continue
		}
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "|") || strings.HasPrefix(line, "<") || strings.HasPrefix(line, "[<") {
			continue
		}
		summary = cleanMarkdown(line)
		break
	}
	source = "readme"
	if summary == "" {
		summary = "README present for " + title + ", but no short prose summary was detected."
		source = "readme-title"
	}
	return title, summary, source, readme
}

func findReadme(repo string) string {
	for _, name := range readmeNames {
		candidate := filepath.Join(repo, name)
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() {
			return candidate
		}
	}
	return ""
}
