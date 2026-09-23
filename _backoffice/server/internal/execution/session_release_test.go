package execution

import "testing"

func TestSessionReleaseEligible(t *testing.T) {
	if !SessionReleaseEligible(RuntimeSession{ClaimID: "a", ExecutionStatus: "resumable"}) {
		t.Fatal("resumable should be eligible")
	}
	if !SessionReleaseEligible(RuntimeSession{ClaimID: "a", ExecutionStatus: "dead"}) {
		t.Fatal("dead should be eligible")
	}
	if SessionReleaseEligible(RuntimeSession{ClaimID: "a", ExecutionStatus: "succeeded"}) {
		t.Fatal("succeeded should not be eligible")
	}
	if SessionReleaseEligible(RuntimeSession{ExecutionStatus: "resumable"}) {
		t.Fatal("missing claim id should not be eligible")
	}
}
