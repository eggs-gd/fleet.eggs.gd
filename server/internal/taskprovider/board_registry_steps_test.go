package taskprovider

import "testing"

func TestMergeEffectiveRepositoryViews(t *testing.T) {
	raw := []registryRepositoryJSON{
		{ID: "1", Name: "", RelativePath: "a/one", Remote: ""},
		{ID: "", Name: "two", RelativePath: "a/two", Branch: "main"},
		{ID: "3", Name: "three"},
	}
	effective := []registryRepositoryJSON{
		{ID: "1", Name: "one", RelativePath: "a/one", Remote: "r1", Branch: "dev", ParentRepositoryID: "p", NestedRepositoryIDs: []string{"n"}},
		{ID: "2", RelativePath: "a/two", Remote: "r2", Branch: "other"},
	}
	got := mergeEffectiveRepositoryViews(raw, effective)
	if len(got) != 3 {
		t.Fatalf("got %d rows", len(got))
	}
	if got[0].Name != "one" || got[0].Remote != "r1" || got[0].Branch != "dev" || got[0].ParentRepositoryID != "p" || len(got[0].NestedRepositoryIDs) != 1 {
		t.Fatalf("matched by id, empty fields filled: %+v", got[0])
	}
	if got[1].ID != "2" || got[1].Branch != "main" || got[1].Remote != "r2" {
		t.Fatalf("matched by path, set fields kept: %+v", got[1])
	}
	if got[2].Name != "three" || got[2].Remote != "" {
		t.Fatalf("no match, unchanged: %+v", got[2])
	}
	if out := mergeEffectiveRepositoryViews(nil, effective); len(out) != 2 {
		t.Fatal("no raw rows returns the effective ones")
	}
	if out := mergeEffectiveRepositoryViews(raw, nil); len(out) != 3 {
		t.Fatal("no effective rows returns the raw ones")
	}
}
