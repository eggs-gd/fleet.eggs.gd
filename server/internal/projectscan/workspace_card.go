package projectscan

import (
	"encoding/json"
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
		// The scanner's own first guess. manager_describe flips this to
		// "confirmed" once a person or the Manager has approved or written
		// the paragraph below; MaintainCards never touches it after that.
		"summary_source: \"generated\"",
		"repositories:",
	}
	for _, repo := range project.Repositories {
		lines = append(lines, "  - "+jsonString(repo.RelativePath))
	}
	lines = append(lines, "---", "", "# "+project.Title, "", project.Summary)
	return joinLines(lines)
}

func jsonString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}
