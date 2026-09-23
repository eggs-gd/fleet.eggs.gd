package providerconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsToMarkdownWhenConfigFileAbsent(t *testing.T) {
	settings, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if settings.Type != "markdown" {
		t.Fatalf("Type = %q, want markdown", settings.Type)
	}
}

func TestLoadMarkdownExplicit(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "taskProvider:\n  type: markdown\n")
	settings, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Type != "markdown" {
		t.Fatalf("Type = %q, want markdown", settings.Type)
	}
}

func TestLoadPlaneWithStatusMapAndRepository(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `taskProvider:
  type: plane
  workspace: eggs_gd
  baseUrl: https://api.plane.so
  tokenEnv: PLANE_API_TOKEN
  project: 550e8400-e29b-41d4-a716-446655440000
  coreProject: core-eggs-gd
  repository: core.eggs.gd
statusMap:
  needs_review: In Review
  doing: In Progress
`)
	settings, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Type != "plane" {
		t.Fatalf("Type = %q, want plane", settings.Type)
	}
	if settings.Plane.Workspace != "eggs_gd" {
		t.Fatalf("Workspace = %q", settings.Plane.Workspace)
	}
	if settings.Plane.BaseURL != "https://api.plane.so" {
		t.Fatalf("BaseURL = %q", settings.Plane.BaseURL)
	}
	if settings.Plane.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("ProjectID = %q", settings.Plane.ProjectID)
	}
	if settings.Plane.CoreProject != "core-eggs-gd" {
		t.Fatalf("CoreProject = %q", settings.Plane.CoreProject)
	}
	if settings.Plane.Repository != "core.eggs.gd" {
		t.Fatalf("Repository = %q", settings.Plane.Repository)
	}
	if settings.Plane.StatusMap["needs_review"] != "In Review" || settings.Plane.StatusMap["doing"] != "In Progress" {
		t.Fatalf("StatusMap = %#v", settings.Plane.StatusMap)
	}
}

func TestLoadPlaneRequiresWorkspaceProjectAndCoreProject(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "taskProvider:\n  type: plane\n")
	if _, err := Load(root); err == nil {
		t.Fatal("expected an error when taskProvider.workspace/project/coreProject are missing")
	}
}

func TestLoadRejectsUnknownProviderType(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "taskProvider:\n  type: jira\n")
	if _, err := Load(root); err == nil {
		t.Fatal("expected an error for an unknown taskProvider.type")
	}
}

func writeConfig(t *testing.T, root string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
