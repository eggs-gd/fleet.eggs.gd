package projectscan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func renderWorkspace(project Workspace) string {
	lines := []string{
		"---",
		"id: " + jsonString(project.ID),
		"title: " + jsonString(project.Title),
		"kind: " + jsonString(project.Kind),
		"review_status: " + jsonString(project.ReviewStatus),
		"status: " + jsonString(project.Status),
		"source: " + jsonString(project.Source),
		"repositories:",
	}
	for _, repo := range project.Repositories {
		lines = append(lines, "  - "+jsonString(repo.RelativePath))
	}
	lines = append(lines, "---", "", "# "+project.Title, "", project.Summary, "", "## Registry", "")
	stack := "-"
	if len(project.Stack) > 0 {
		stack = joinComma(project.Stack)
	}
	lines = append(lines,
		"- Project id: `"+project.ID+"`",
		"- Kind: `"+project.Kind+"`",
		"- Review status: `"+project.ReviewStatus+"`",
		"- Status: `"+project.Status+"`",
		"- Source: `"+project.Source+"`",
		"- Stack: "+stack,
		"",
		"## Repositories",
		"",
		"| Repository | Branch | Effective Profile | Remote | Summary |",
		"|---|---|---|---|---|",
	)
	repos := append([]workspaceRepo{}, project.Repositories...)
	sort.Slice(repos, func(i, j int) bool {
		return strings.ToLower(repos[i].RelativePath) < strings.ToLower(repos[j].RelativePath)
	})
	for _, repo := range repos {
		branch := repo.Branch
		if branch == "" {
			branch = "-"
		}
		remote := repo.Remote
		if remote == "" {
			remote = "-"
		}
		lines = append(lines, fmt.Sprintf("| `%s` | `%s` | %s | `%s` | %s |", repo.RelativePath, branch, formatProfile(repo), remote, escapePipe(repo.Summary)))
	}
	lines = append(lines, "", "## Relationships", "")
	if len(project.Relations) == 0 {
		lines = append(lines, "No related _registry candidates recorded.")
	} else {
		for _, item := range project.Relations {
			var paths []string
			for _, path := range item.RepositoryPaths {
				paths = append(paths, "`"+path+"`")
			}
			lines = append(lines, fmt.Sprintf("- %s / %s / %s: %s", item.Kind, item.Reason, item.Confidence, joinComma(paths)))
		}
	}
	lines = append(lines, "", "## Protection", "")
	if len(project.Protection) == 0 {
		lines = append(lines, "No protection rules recorded.")
	} else {
		for _, note := range project.Protection {
			lines = append(lines, "- "+note)
		}
	}
	lines = append(lines, "", "## Technology Evidence", "")
	evidenceLines := technologyEvidence(repos)
	lines = append(lines, evidenceLines...)
	lines = append(lines, "", "## Notes", "", "- Generated from _registry. Review and edit before treating this as canonical workspace metadata.")
	return joinLines(lines)
}

func formatProfile(repo workspaceRepo) string {
	if repo.Effective == nil {
		if len(repo.Stack) == 0 {
			return "-"
		}
		return joinComma(repo.Stack)
	}
	var fields []string
	add := func(label string, values []string) {
		if len(values) > 0 {
			fields = append(fields, label+": "+joinComma(values))
		}
	}
	add("languages", repo.Effective.Languages)
	add("frameworks", repo.Effective.Frameworks)
	add("runtimes", repo.Effective.Runtimes)
	add("tooling", repo.Effective.Tooling)
	add("capabilities", repo.Effective.Capabilities)
	if len(fields) == 0 {
		return "-"
	}
	return joinBreak(fields)
}

func technologyEvidence(repos []workspaceRepo) []string {
	var lines []string
	for _, repo := range repos {
		if repo.Detected == nil || len(repo.Detected.Evidence) == 0 {
			continue
		}
		lines = append(lines, "### `"+repo.RelativePath+"`", "", "| Path | Technologies | Detail | Status |", "|---|---|---|---|")
		for _, item := range repo.Detected.Evidence {
			techs := "-"
			if len(item.Technologies) > 0 {
				techs = joinComma(item.Technologies)
			}
			status := "active"
			if item.Ignored && item.IgnoreReason != nil {
				status = *item.IgnoreReason
			}
			lines = append(lines, fmt.Sprintf("| `%s` | %s | %s | %s |", item.Path, techs, escapePipe(item.Detail), status))
		}
		lines = append(lines, "")
	}
	if len(lines) == 0 {
		return []string{"No technology evidence recorded."}
	}
	return lines
}

func jsonString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}
