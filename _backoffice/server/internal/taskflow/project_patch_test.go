package taskflow

import (
	"context"
	"testing"
)

func TestProjectPatchContextRoundTrip(t *testing.T) {
	ctx := WithProjectPatch(context.Background(), " eggs-gd-prod/career-wizard ", " eGGs.gd.prod/career-wizard ")
	project, repository, ok := ProjectPatchFrom(ctx)
	if !ok {
		t.Fatal("expected project patch")
	}
	if project != "eggs-gd-prod/career-wizard" {
		t.Fatalf("project = %q", project)
	}
	if repository != "eGGs.gd.prod/career-wizard" {
		t.Fatalf("repository = %q", repository)
	}

	_, _, ok = ProjectPatchFrom(WithProjectPatch(context.Background(), "  ", "repo"))
	if ok {
		t.Fatal("empty project should not attach a patch")
	}
}
