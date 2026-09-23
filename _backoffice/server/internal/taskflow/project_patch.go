package taskflow

import (
	"context"
	"strings"
)

type projectPatchCtxKey struct{}

type projectPatch struct {
	Project    string
	Repository string
}

// WithProjectPatch marks a provider Update as an intentional project move so
// adapters can rewrite storage identity without treating every field update
// as a project change.
func WithProjectPatch(ctx context.Context, project string, repository string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	project = strings.TrimSpace(project)
	if project == "" {
		return ctx
	}
	return context.WithValue(ctx, projectPatchCtxKey{}, projectPatch{
		Project:    project,
		Repository: strings.TrimSpace(repository),
	})
}

// ProjectPatchFrom returns the project/repository attached by WithProjectPatch.
func ProjectPatchFrom(ctx context.Context) (project string, repository string, ok bool) {
	if ctx == nil {
		return "", "", false
	}
	value, ok := ctx.Value(projectPatchCtxKey{}).(projectPatch)
	if !ok || strings.TrimSpace(value.Project) == "" {
		return "", "", false
	}
	return value.Project, value.Repository, true
}
