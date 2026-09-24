package projectscan

import "fmt"

func renderBootstrapReport(root string, repositories []Repository, groups []Group, candidates []DeletionCandidate, generatedAt string) string {
	lines := []string{
		"# Bootstrap Report",
		"",
		fmt.Sprintf("Generated at: `%s`", generatedAt),
		fmt.Sprintf("Projects root: `%s`", root),
		"",
		fmt.Sprintf("Discovered repositories: **%d**", len(repositories)),
		fmt.Sprintf("Groups and candidates: **%d**", len(groups)),
		fmt.Sprintf("Deletion review candidates: **%d**", len(candidates)),
		"",
		"## Repositories",
		"",
		"| Name | Path | Branch | Stack | Summary |",
		"|---|---|---|---|---|",
	}
	for _, repo := range repositories {
		stack := "-"
		if len(repo.Stack) > 0 {
			stack = joinComma(repo.Stack)
		}
		branch := repo.Branch
		if branch == "" {
			branch = "-"
		}
		lines = append(lines, fmt.Sprintf("| %s | `%s` | `%s` | %s | %s |", repo.Title, repo.RelativePath, branch, stack, escapePipe(repo.Summary)))
	}
	lines = append(lines, "", "## Groups And Candidates", "")
	if len(groups) == 0 {
		lines = append(lines, "No grouping suggestions detected.")
	} else {
		labels := map[string]string{}
		for _, repo := range repositories {
			labels[repo.ID] = fmt.Sprintf("%s (%s)", repo.Name, repo.RelativePath)
		}
		for _, group := range groups {
			var names []string
			for _, id := range group.RepositoryIDs {
				label := labels[id]
				if label == "" {
					label = id
				}
				names = append(names, "`"+label+"`")
			}
			lines = append(lines,
				"### "+group.SuggestedProjectID,
				"",
				"- Kind: "+group.Kind,
				"- Confidence: "+group.Confidence,
				"- Decision: "+group.Decision,
				"- Reason: "+group.Reason,
				"- Repositories: "+joinComma(names),
				"",
			)
		}
	}
	return joinLines(lines)
}

func renderDeletionReport(candidates []DeletionCandidate, generatedAt string) string {
	lines := []string{
		"# Deletion Candidates",
		"",
		fmt.Sprintf("Generated at: `%s`", generatedAt),
		"",
		"These are review candidates only. Core never deletes repositories from this report.",
		"",
		fmt.Sprintf("Candidates: **%d**", len(candidates)),
		"",
	}
	if len(candidates) == 0 {
		lines = append(lines, "No deletion candidates detected.")
	} else {
		lines = append(lines, "| Confidence | Path | Git | Reasons | Details |", "|---|---|---|---|---|")
		for _, candidate := range candidates {
			details := escapePipe(joinBreak(candidate.Details))
			lines = append(lines, fmt.Sprintf("| %s | `%s` | %s | %s | %s |", candidate.Confidence, candidate.RelativePath, candidate.GitStatus, joinComma(candidate.Reasons), details))
		}
	}
	return joinLines(lines)
}

func joinLines(lines []string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n"
		}
		out += line
	}
	return stringsTrimRightNewlines(out) + "\n"
}

func stringsTrimRightNewlines(value string) string {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == ' ') {
		value = value[:len(value)-1]
	}
	return value
}

func joinComma(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ", "
		}
		out += value
	}
	return out
}

func joinBreak(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += "<br>"
		}
		out += value
	}
	return out
}

func escapePipe(value string) string {
	out := ""
	for _, r := range value {
		if r == '|' {
			out += `\|`
			continue
		}
		out += string(r)
	}
	return out
}
