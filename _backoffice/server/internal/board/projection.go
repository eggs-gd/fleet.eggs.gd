// Package board owns dashboard-facing task and workspace projections.
package board

import (
	"path/filepath"

	"github.com/eggs-gd/core.eggs.gd/internal/mdfile"
)

type Workspace struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Kind         string            `json:"kind"`
	ReviewStatus string            `json:"review_status"`
	Status       string            `json:"status"`
	Source       string            `json:"source"`
	Repositories []string          `json:"repositories"`
	Summary      string            `json:"summary"`
	Path         string            `json:"path"`
	RelativePath string            `json:"relative_path"`
	Technology   TechnologySummary `json:"technology"`
}

type Project struct {
	ID           string            `json:"id"`
	WorkspaceID  string            `json:"workspace_id"`
	Title        string            `json:"title"`
	Kind         string            `json:"kind"`
	Source       string            `json:"source"`
	Repositories []string          `json:"repositories"`
	Summary      string            `json:"summary"`
	Path         string            `json:"path"`
	RelativePath string            `json:"relative_path"`
	Technology   TechnologySummary `json:"technology"`
}

type TechnologySummary struct {
	EffectiveTags []string               `json:"effective_tags"`
	DetectedTags  []string               `json:"detected_tags"`
	Repositories  []RepositoryTechnology `json:"repositories,omitempty"`
}

type RepositoryTechnology struct {
	ID                  string               `json:"id"`
	Name                string               `json:"name"`
	RelativePath        string               `json:"relative_path"`
	Remote              string               `json:"remote,omitempty"`
	Branch              string               `json:"branch,omitempty"`
	ParentRepositoryID  string               `json:"parent_repository_id,omitempty"`
	NestedRepositoryIDs []string             `json:"nested_repository_ids,omitempty"`
	Effective           TechnologyProfile    `json:"effective"`
	Detected            TechnologyProfile    `json:"detected"`
	EffectiveTags       []string             `json:"effective_tags"`
	DetectedTags        []string             `json:"detected_tags"`
	Evidence            []TechnologyEvidence `json:"evidence,omitempty"`
}

type TechnologyProfile struct {
	Languages    []string `json:"languages,omitempty"`
	Frameworks   []string `json:"frameworks,omitempty"`
	Runtimes     []string `json:"runtimes,omitempty"`
	Tooling      []string `json:"tooling,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type TechnologyEvidence struct {
	Kind         string   `json:"kind,omitempty"`
	Path         string   `json:"path"`
	Technologies []string `json:"technologies"`
	Detail       string   `json:"detail,omitempty"`
	Ignored      bool     `json:"ignored,omitempty"`
	IgnoreReason string   `json:"ignore_reason,omitempty"`
}

type LifeItem struct {
	ID           string `json:"id"`
	Ref          string `json:"ref"`
	Title        string `json:"title"`
	Kind         string `json:"kind"`
	Status       string `json:"status"`
	Source       string `json:"source"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Summary      string `json:"summary"`
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
}

type RegistryInfo struct {
	RepositoriesCount int                    `json:"repositories_count"`
	GroupsCount       int                    `json:"groups_count"`
	Repositories      []RepositoryTechnology `json:"repositories"`
}

func LoadWorkspaceFile(root, workspaceFile string) (Workspace, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Workspace{}, err
	}
	workspaceFile, err = filepath.Abs(workspaceFile)
	if err != nil {
		return Workspace{}, err
	}
	fm, body, err := mdfile.ReadMarkdown(workspaceFile)
	if err != nil {
		return Workspace{}, err
	}
	id := mdfile.Scalar(fm, "id", filepath.Base(filepath.Dir(workspaceFile)))
	rel, _ := filepath.Rel(root, workspaceFile)
	return Workspace{
		ID:           id,
		Title:        mdfile.Scalar(fm, "title", id),
		Kind:         mdfile.Scalar(fm, "kind", "workspace"),
		ReviewStatus: mdfile.Scalar(fm, "review_status", "draft"),
		Status:       mdfile.Scalar(fm, "status", "discovered"),
		Source:       mdfile.Scalar(fm, "source", ""),
		Repositories: mdfile.List(fm, "repositories"),
		Summary:      mdfile.FirstParagraph(body),
		Path:         workspaceFile,
		RelativePath: filepath.ToSlash(rel),
	}, nil
}
