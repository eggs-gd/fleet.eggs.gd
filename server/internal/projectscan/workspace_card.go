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
