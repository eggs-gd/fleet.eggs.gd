package board

import "testing"

func TestRepositoryProjectsShareTheWorkspaceMilestones(t *testing.T) {
	workspace := Workspace{
		ID:           "acme",
		Kind:         "workspace_group",
		Repositories: []string{"acme/web", "acme/api"},
		Milestones:   []Milestone{{ID: "v1-launch", Title: "v1 launch", State: "active"}},
	}
	projects := ProjectsFromWorkspace(workspace)
	if len(projects) != 3 {
		t.Fatalf("projects = %d, want workspace + 2 repository projects", len(projects))
	}
	for _, project := range projects {
		if len(project.Milestones) != 1 || project.Milestones[0].ID != "v1-launch" {
			t.Fatalf("project %s (%s) milestones = %#v", project.ID, project.Kind, project.Milestones)
		}
	}
}
