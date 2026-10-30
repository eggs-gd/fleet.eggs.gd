package projectscan

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseCardFrontmatter(t *testing.T) {
	id, repos := parseCardFrontmatter("id: \"alpha\"\ntitle: A\nrepositories:\n  - \"a/one\"\n  - a/two\nstatus: ok\n  - ignored")
	if id != "alpha" {
		t.Fatalf("id = %q", id)
	}
	if want := []string{"a/one", "a/two"}; !reflect.DeepEqual(repos, want) {
		t.Fatalf("repositories = %v, want %v", repos, want)
	}
}

func TestReadCardOverride(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "alpha", "PROJECT.md")
	writeTestFile(t, full, "---\nid: alpha\nrepositories:\n  - a/one\n---\n\n## Technology Overrides\nadd_languages: [go, rust]\nremove_tooling:\n  - docker\n\n## Other\n")
	id, repos, override, ok := readCardOverride(full)
	if !ok || id != "alpha" || !reflect.DeepEqual(repos, []string{"a/one"}) {
		t.Fatalf("got %q %v %v", id, repos, ok)
	}
	if !reflect.DeepEqual(override.Add, []string{"go", "rust"}) || !reflect.DeepEqual(override.Remove, []string{"docker"}) {
		t.Fatalf("override = %+v", override)
	}

	cases := map[string]string{
		"no frontmatter": "# title\n",
		"unterminated":   "---\nid: a\n",
		"no section":     "---\nid: a\n---\nbody\n",
		"empty section":  "---\nid: a\n---\n\n## Technology Overrides\nnothing: here\n",
	}
	for name, text := range cases {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "-"), "PROJECT.md")
		writeTestFile(t, path, text)
		if _, _, _, ok := readCardOverride(path); ok {
			t.Errorf("%s: want no override", name)
		}
	}
}

func TestReadCardOverrideFallsBackToFolderName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folder-id", "PROJECT.md")
	writeTestFile(t, path, "---\ntitle: x\n---\n\n## Technology Overrides\nadd_languages: go\n")
	id, _, _, ok := readCardOverride(path)
	if !ok || id != "folder-id" {
		t.Fatalf("id = %q ok = %v", id, ok)
	}
}

func TestParseProjectCardOverrides(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a", "PROJECT.md"), "---\nid: a\nrepositories:\n  - r/one\n  - r/two\n---\n\n## Technology Overrides\nadd_languages: go\n")
	writeTestFile(t, filepath.Join(dir, "b", "PROJECT.md"), "---\nid: b\n---\nnothing\n")
	got := parseProjectCardOverrides(dir)
	if len(got) != 2 || got["r/one"].ProjectID != "a" || got["r/two"].ProjectID != "a" {
		t.Fatalf("overrides = %+v", got)
	}
	if len(parseProjectCardOverrides("")) != 0 {
		t.Fatal("blank work dir must give no overrides")
	}
}

func TestReadmeSummaryPieces(t *testing.T) {
	lines := []string{"", "[![badge](x)](y)", "# Fleet tool", "", "```", "not prose", "```", "- list item", "| table |", "<div>", "Real prose here.", "More."}
	if got := readmeTitle(lines, "fallback"); got != "Fleet tool" {
		t.Fatalf("title = %q", got)
	}
	if got := readmeTitle([]string{"no heading"}, "fallback"); got != "fallback" {
		t.Fatalf("title = %q", got)
	}
	if got := firstProseLine(lines); got != "Real prose here." {
		t.Fatalf("prose = %q", got)
	}
	if got := firstProseLine([]string{"# only a heading", "- list"}); got != "" {
		t.Fatalf("prose = %q, want empty", got)
	}
	if !hasAnyPrefix("[<x", "a", "[<") || hasAnyPrefix("plain", "#", "-") {
		t.Fatal("hasAnyPrefix")
	}
}

func TestSummarizeReadmeFiles(t *testing.T) {
	repo := t.TempDir()
	title, summary, source, path := summarizeReadme(repo)
	if source != "fallback" || path != "" || title != filepath.Base(repo) || !strings.Contains(summary, "No README found") {
		t.Fatalf("no readme: %q %q %q %q", title, summary, source, path)
	}
	writeTestFile(t, filepath.Join(repo, "README.md"), "# Title Only\n\n- a\n- b\n")
	title, summary, source, _ = summarizeReadme(repo)
	if title != "Title Only" || source != "readme-title" || !strings.Contains(summary, "no short prose summary") {
		t.Fatalf("title only: %q %q %q", title, summary, source)
	}
	writeTestFile(t, filepath.Join(repo, "README.md"), "# Title\n\nA short description.\n")
	_, summary, source, _ = summarizeReadme(repo)
	if source != "readme" || summary != "A short description." {
		t.Fatalf("prose: %q %q", summary, source)
	}
}

func repos(items ...Repository) []Repository { return items }

func TestRepositoriesByDropsBlankKeys(t *testing.T) {
	got := repositoriesBy(repos(Repository{ID: "1", Remote: "a"}, Repository{ID: "2"}, Repository{ID: "3", Remote: "a"}), func(r Repository) string { return r.Remote })
	if len(got) != 1 || len(got["a"]) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestSuggestGroups(t *testing.T) {
	list := repos(
		Repository{ID: "a1", Name: "alpha", RelativePath: "mono/alpha", ScanRoot: "/r", RemoteIdentity: "gh/x/alpha"},
		Repository{ID: "a2", Name: "alpha-copy", RelativePath: "mono/alpha-copy", ScanRoot: "/r", RemoteIdentity: "gh/x/alpha"},
		Repository{ID: "b1", Name: "beta-old", RelativePath: "solo/beta-old", ScanRoot: "/r"},
		Repository{ID: "b2", Name: "beta", RelativePath: "other/beta", ScanRoot: "/r", NestedRepositoryIDs: []string{"b1"}},
	)
	groups := suggestGroups(list)
	byKind := map[string][]Group{}
	for _, g := range groups {
		byKind[g.Kind] = append(byKind[g.Kind], g)
	}
	if g := byKind["workspace_group"]; len(g) != 1 || g[0].SuggestedProjectID != "mono" || !reflect.DeepEqual(g[0].RepositoryIDs, []string{"a1", "a2"}) {
		t.Fatalf("workspace groups = %+v", g)
	}
	if g := byKind["duplicate_candidate"]; len(g) != 1 || g[0].ID != "remote-gh/x/alpha" {
		t.Fatalf("duplicate groups = %+v", g)
	}
	if g := byKind["related_candidate"]; len(g) != 2 || g[0].Reason != "same base repository name" {
		t.Fatalf("related groups = %+v", g)
	}
	if g := byKind["nested_group"]; len(g) != 1 || !reflect.DeepEqual(g[0].RepositoryIDs, []string{"b1", "b2"}) {
		t.Fatalf("nested groups = %+v", g)
	}
	for i := 1; i < len(groups); i++ {
		if kindPriority(groups[i-1].Kind) > kindPriority(groups[i].Kind) {
			t.Fatalf("groups are not ordered by kind: %+v", groups)
		}
	}
}

func TestTopFolderGroupsNamesProjectsByRootWhenSeveralRoots(t *testing.T) {
	groups := topFolderGroups(repos(
		Repository{ID: "1", RelativePath: "mono/a", ScanRoot: "/one"},
		Repository{ID: "2", RelativePath: "mono/b", ScanRoot: "/one"},
		Repository{ID: "3", RelativePath: "other/c", ScanRoot: "/two"},
	))
	if len(groups) != 1 || groups[0].SuggestedProjectID != "one-mono" {
		t.Fatalf("groups = %+v", groups)
	}
}

func TestRankedUniqueGroupsDropsSmallAndRepeated(t *testing.T) {
	g := Group{Kind: "related_candidate", Reason: "r", SuggestedProjectID: "p", RepositoryIDs: []string{"a", "b"}}
	out := rankedUniqueGroups([]Group{g, g, {Kind: "nested_group", Reason: "r", RepositoryIDs: []string{"a"}}})
	if len(out) != 1 {
		t.Fatalf("got %+v", out)
	}
}

func TestNameHints(t *testing.T) {
	if got := nameHints("My-Old_Backup.tmp"); !reflect.DeepEqual(got, []string{"backup", "old", "tmp"}) {
		t.Fatalf("hints = %v", got)
	}
	if got := nameHints("product"); len(got) != 0 {
		t.Fatalf("hints = %v", got)
	}
}

func TestSuggestDeletionCandidates(t *testing.T) {
	list := repos(
		Repository{ID: "k", Name: "app", RelativePath: "app", Path: "/nonexistent/app", Remote: "git@x:y/app.git", RemoteIdentity: "x/y/app", ReadmePath: "README.md"},
		Repository{ID: "d", Name: "app-backup", RelativePath: "deep/dir/app-backup", Path: "/nonexistent/d", Remote: "git@x:y/app.git", RemoteIdentity: "x/y/app", ReadmePath: "README.md"},
		Repository{ID: "n", Name: "inner", RelativePath: "app/inner", Path: "/nonexistent/n", ParentRepositoryID: "k", Remote: "r", ReadmePath: "README.md"},
		Repository{ID: "p", Name: "keep", RelativePath: "protected/keep", Path: "/nonexistent/p"},
	)
	rules := protectionRules{ProtectedPrefixes: []prefixRule{{Prefix: "protected/", Reason: "kept"}}}
	got := suggestDeletionCandidates(list, rules)
	byID := map[string]DeletionCandidate{}
	for _, c := range got {
		byID[c.RepositoryID] = c
	}
	if _, ok := byID["p"]; ok {
		t.Fatal("a protected repository must not be a candidate")
	}
	if _, ok := byID["k"]; ok {
		t.Fatal("the shallowest checkout is the one to keep")
	}
	d := byID["d"]
	if d.Confidence != "high" || !contains(d.Reasons, "duplicate local checkout") || !contains(d.Reasons, "temporary name hint") {
		t.Fatalf("duplicate = %+v", d)
	}
	n := byID["n"]
	if n.Confidence != "high" || !contains(n.Details, "Repository is nested inside app; review whether it is vendored/submodule/intentional.") {
		t.Fatalf("nested = %+v", n)
	}
	if got[0].Confidence != "high" {
		t.Fatalf("candidates are not ranked: %+v", got)
	}
}

func TestDeletionConfidence(t *testing.T) {
	cases := []struct {
		reasons []string
		want    string
	}{
		{[]string{"nested repository"}, "high"},
		{[]string{"no git remote", "no README"}, "medium"},
		{[]string{"no git remote"}, "low"},
		{[]string{"no README"}, "low"},
	}
	for _, c := range cases {
		if got := deletionConfidence(c.reasons); got != c.want {
			t.Errorf("%v: %s, want %s", c.reasons, got, c.want)
		}
	}
}

func testWorkspaceRepos() map[string]workspaceRepo {
	return map[string]workspaceRepo{
		"1": {ID: "1", Name: "one", RelativePath: "mono/one"},
		"2": {ID: "2", Name: "two", RelativePath: "mono/two"},
		"3": {ID: "3", Name: "three", RelativePath: "solo/three"},
	}
}

func TestGroupProjectsAndMembers(t *testing.T) {
	byID := testWorkspaceRepos()
	groups := []Group{
		{Kind: "workspace_group", SuggestedProjectID: "mono", Reason: "same top-level folder", RepositoryIDs: []string{"1", "2", "missing"}},
		{Kind: "related_candidate", Reason: "r", RepositoryIDs: []string{"2", "3"}},
	}
	projects, covered := groupProjects(groups, byID)
	if len(projects) != 1 || projects[0].ID != "mono" || len(projects[0].Repositories) != 2 || projects[0].Title != "mono" {
		t.Fatalf("projects = %+v", projects)
	}
	if _, ok := covered["missing"]; !ok || len(covered) != 3 {
		t.Fatalf("covered = %v", covered)
	}
	paths, idx := groupMembers(groups[1], byID, map[string]int{"2": 0})
	if !reflect.DeepEqual(paths, []string{"mono/two", "solo/three"}) || len(idx) != 1 {
		t.Fatalf("paths %v idx %v", paths, idx)
	}
}

func TestAttachRelations(t *testing.T) {
	byID := testWorkspaceRepos()
	projects := []Workspace{{ID: "mono", Repositories: []workspaceRepo{byID["1"], byID["2"]}}, {ID: "solo", Repositories: []workspaceRepo{byID["3"]}}}
	groups := []Group{
		{Kind: "workspace_group", RepositoryIDs: []string{"1", "2"}},
		{Kind: "related_candidate", Reason: "same name", Confidence: "medium", RepositoryIDs: []string{"2", "3"}},
		{Kind: "duplicate_candidate", Reason: "hidden", RepositoryIDs: []string{"3"}},
	}
	rules := protectionRules{ProtectedPrefixes: []prefixRule{{Prefix: "solo", Reason: "p"}}}
	attachRelations(projects, groups, byID, rules)
	if len(projects[0].Relations) != 1 || projects[0].Relations[0].Reason != "same name" {
		t.Fatalf("mono relations = %+v", projects[0].Relations)
	}
	if len(projects[1].Relations) != 1 {
		t.Fatalf("solo relations = %+v", projects[1].Relations)
	}
	if allProtected(nil, rules) || !allProtected([]string{"solo/x"}, rules) || allProtected([]string{"solo/x", "mono/y"}, rules) {
		t.Fatal("allProtected")
	}
}

func TestLoadWorkspaceReposMergesEffective(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "repositories.json"), `{
  "repositories": [{"id":"1","name":"one","relative_path":"a/one","summary":"raw","branch":"main","remote":"r0"}],
  "effective_repositories": [{"id":"1","relative_path":"a/one","summary":"eff","branch":"","remote":"r1","effective":{"languages":["go"]}}]
}`)
	repos, err := loadWorkspaceRepos(dir)
	if err != nil || len(repos) != 1 {
		t.Fatalf("repos %v err %v", repos, err)
	}
	r := repos[0]
	if r.Summary != "eff" || r.Branch != "main" || r.Remote != "r1" || r.Effective == nil {
		t.Fatalf("merged = %+v", r)
	}
	if _, err := loadWorkspaceRepos(t.TempDir()); err == nil {
		t.Fatal("a missing registry must be an error")
	}
}

func TestWithEffectiveFields(t *testing.T) {
	got := withEffectiveFields(workspaceRepoJSON{Summary: "a", Branch: "b", Remote: "c"}, workspaceRepoJSON{Summary: "", Branch: "x"})
	if got.Summary != "a" || got.Branch != "x" || got.Remote != "c" {
		t.Fatalf("got %+v", got)
	}
}

func TestBuildWorkspaces(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "repositories.json"), `{"repositories":[
 {"id":"1","name":"one","relative_path":"mono/one"},
 {"id":"2","name":"two","relative_path":"mono/two"},
 {"id":"3","name":"three","relative_path":"solo/three"}]}`)
	writeTestFile(t, filepath.Join(dir, "project-groups.json"), `{"groups":[
 {"id":"g","kind":"workspace_group","reason":"same top-level folder","suggested_project_id":"mono","repository_ids":["1","2"]}]}`)
	ws, err := BuildWorkspaces(dir)
	if err != nil || len(ws) != 2 {
		t.Fatalf("ws %+v err %v", ws, err)
	}
	if ws[0].ID != "mono" || ws[0].Kind != "workspace_group" || ws[1].Kind != "standalone_repository" || ws[1].ID != "three" {
		t.Fatalf("ws = %+v", ws)
	}
}

func TestTagCardsAndProblems(t *testing.T) {
	work := t.TempDir()
	writeTestFile(t, filepath.Join(work, "alpha", "PROJECT.md"), "---\nid: alpha\ntag: ALP\n---\n")
	writeTestFile(t, filepath.Join(work, "beta", "PROJECT.md"), "---\nid: beta\ntag: ALP\n---\n")
	writeTestFile(t, filepath.Join(work, "gamma", "PROJECT.md"), "---\nid: gamma\ntag: bad tag!\n---\n")
	writeTestFile(t, filepath.Join(work, "delta", "PROJECT.md"), "no frontmatter\n")
	writeTestFile(t, filepath.Join(work, "epsilon", "PROJECT.md"), "---\nid: epsilon\n---\n")
	writeTestFile(t, filepath.Join(work, "stray.txt"), "x")

	cards, err := readTagCards(work)
	if err != nil || len(cards) != 5 || cards[0].id != "alpha" {
		t.Fatalf("cards %v err %v", cards, err)
	}
	cards, err = readTagCards(filepath.Join(work, "missing"))
	if err != nil || cards != nil {
		t.Fatalf("missing dir: %v %v", cards, err)
	}

	report, err := assignTags(work, CardReport{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(report.Problems, "\n")
	for _, want := range []string{"tag ALP is used by alpha, beta", "project gamma has tag", "project delta has no frontmatter"} {
		if !strings.Contains(joined, want) {
			t.Errorf("problems %q lack %q", joined, want)
		}
	}
	if !reflect.DeepEqual(report.Tagged, []string{"epsilon"}) {
		t.Fatalf("tagged = %v", report.Tagged)
	}
	text, _ := os.ReadFile(filepath.Join(work, "epsilon", "PROJECT.md"))
	if !strings.Contains(string(text), "tag:") {
		t.Fatalf("card was not tagged: %s", text)
	}
	if report, _ := assignTags(filepath.Join(work, "missing"), CardReport{}); len(report.Tagged) != 0 {
		t.Fatal("a missing work dir tags nothing")
	}
}

func TestDetectTechnologies(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "package.json"), `{"dependencies":{"svelte":"1","vite":"1"}}`)
	writeTestFile(t, filepath.Join(repo, "go.mod"), "module x\n")
	writeTestFile(t, filepath.Join(repo, "node_modules", "dep", "package.json"), "{}")
	writeTestFile(t, filepath.Join(repo, "a", "b", "c", "d", "package.json"), "{}")
	writeTestFile(t, filepath.Join(repo, "nested", "go.mod"), "module n\n")
	writeTestFile(t, filepath.Join(repo, "App.csproj"), "<Project/>")
	if err := os.MkdirAll(filepath.Join(repo, "Assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "ProjectSettings"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := detectTechnologies(repo, map[string]struct{}{filepath.Join(repo, "nested"): {}}, nil)
	for _, tech := range []string{"go", "svelte", "vite", "csharp", "unity"} {
		if !contains(stackFrom(got.TechLists), tech) {
			t.Errorf("missing %s in %v", tech, stackFrom(got.TechLists))
		}
	}
	for _, e := range got.Evidence {
		if strings.HasPrefix(e.Path, "node_modules") || strings.HasPrefix(e.Path, "nested") || strings.Count(e.Path, "/") >= 3 {
			t.Errorf("evidence must skip pruned, nested and deep paths: %s", e.Path)
		}
	}
	if !skipMarkerDir("/r/.git", ".git", ".git", nil) || skipMarkerDir("/r/src", "src", "src", nil) {
		t.Fatal("skipMarkerDir")
	}
	empty := detectTechnologies(t.TempDir(), nil, nil)
	if empty.Evidence == nil || len(empty.Evidence) != 0 {
		t.Fatalf("evidence = %v", empty.Evidence)
	}
}
