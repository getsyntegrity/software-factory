package state

import (
	"reflect"
	"strings"
	"testing"
)

// oddPairs is the ODD table: the v2 transitions, the three returns to spec
// and X → merged for every non-terminal X.
var oddPairs = map[string][]string{
	Intake:     {NeedsInfo, Rejected, Spec, Merged},
	NeedsInfo:  {Intake, Rejected, Merged},
	Spec:       {Build, Merged},
	Build:      {Gate, NeedsHuman, Spec, Merged},
	Gate:       {Review, Build, NeedsHuman, Spec, Merged},
	Review:     {Approve, Build, NeedsHuman, Spec, Merged},
	Approve:    {Merging, NeedsHuman, Merged},
	Merging:    {NeedsHuman, Merged},
	NeedsHuman: {Build, Rejected, Merged},
	Merged:     {},
	Rejected:   {},
}

// oddFacts passes every ODD guard for candidate 2 at once.
func oddFacts() Facts {
	return Facts{
		SpecExists:        true,
		BuildStatus:       "READY",
		Head:              "abc123",
		Gate:              &GateResult{Verdict: "PASS", Commit: "abc123", Candidate: 2},
		ExpectedReviewers: []string{"code"},
		Reviews:           map[string]string{"code": "PASS"},
		ReviewCandidates:  map[string]int{"code": 2},
		Decision:          "APPROVE",
		DecisionCandidate: 2,
	}
}

func mustAdvance(t *testing.T, p Profile, pos Position, to string, f Facts) Result {
	t.Helper()
	res, err := p.Advance(pos, to, f)
	if err != nil {
		t.Fatalf("%s → %s from %+v: %v", pos.State, to, pos, err)
	}
	return res
}

func mustRefuse(t *testing.T, p Profile, pos Position, to string, f Facts, wants ...string) {
	t.Helper()
	_, err := p.Advance(pos, to, f)
	if err == nil {
		t.Fatalf("%s → %s from %+v: accepted, want an error", pos.State, to, pos)
	}
	for _, w := range wants {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("%s → %s: error %q does not contain %q", pos.State, to, err, w)
		}
	}
}

func TestODDAllowsTable(t *testing.T) {
	count := 0
	for _, from := range allStates {
		for _, to := range allStates {
			count++
			want := false
			for _, s := range oddPairs[from] {
				want = want || s == to
			}
			if got := ODD.Allows(from, to); got != want {
				t.Errorf("ODD.Allows(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}
	if count != 121 {
		t.Errorf("walked %d pairs, want 121", count)
	}
}

func TestV2RefusesReturnsToSpec(t *testing.T) {
	f := oddFacts()
	f.SpecRevision = true
	for _, from := range []string{Build, Gate, Review} {
		mustRefuse(t, V2, Position{State: from, Round: 1, RoundLimit: 3}, Spec, f, "not a legal transition")
	}
}

func TestODDReturnToSpec(t *testing.T) {
	for _, from := range []string{Build, Gate, Review} {
		f := oddFacts()
		f.SpecRevision = true
		pos := Position{State: from, Round: 1, RoundLimit: 3, Candidate: 2}
		res := mustAdvance(t, ODD, pos, Spec, f)
		want := Result{State: Spec, Round: 1, Candidate: 3, NextOperation: "run_spec"}
		if res.State != want.State || res.Round != want.Round || res.Candidate != want.Candidate || res.NextOperation != want.NextOperation {
			t.Errorf("%s → spec = %+v, want %+v", from, res, want)
		}
		if !strings.Contains(res.Cause, "spec") {
			t.Errorf("%s → spec cause %q does not mention the spec", from, res.Cause)
		}
		f.SpecRevision = false
		mustRefuse(t, ODD, pos, Spec, f, "SpecRevision")
	}
}

func TestODDBuildsConsumeRounds(t *testing.T) {
	f := oddFacts()
	f.SpecRevision = true
	pos := Position{State: Spec, Round: 0, RoundLimit: 3}
	steps := []struct {
		to    string
		round int
	}{{Build, 1}, {Spec, 1}, {Build, 2}, {Spec, 2}, {Build, 3}}
	for i, s := range steps {
		res := mustAdvance(t, ODD, pos, s.to, f)
		if res.State != s.to || res.Round != s.round {
			t.Fatalf("step %d: %s → %s = %+v, want round %d", i, pos.State, s.to, res, s.round)
		}
		pos = Position{State: res.State, Round: res.Round, RoundLimit: pos.RoundLimit, Candidate: res.Candidate}
	}
	// The limit counts builds and the spec returns did not reset it.
	mustRefuse(t, ODD, Position{State: Spec, Round: 3, RoundLimit: 3}, Build, f, "needs-human", "3 of 3")
}

func TestODDBuildRoundFromGateAndReview(t *testing.T) {
	f := oddFacts()
	f.Reviews = map[string]string{"code": "CHANGES"}
	res := mustAdvance(t, ODD, Position{State: Review, Round: 1, RoundLimit: 3, Candidate: 2}, Build, f)
	if res.Round != 2 || res.Candidate != 3 {
		t.Errorf("review → build = %+v, want round 2, candidate 3", res)
	}
	f.Gate = &GateResult{Verdict: "FAIL", Commit: "abc123"}
	res = mustAdvance(t, ODD, Position{State: Gate, Round: 2, RoundLimit: 3, Candidate: 3}, Build, f)
	if res.Round != 3 || res.Candidate != 4 {
		t.Errorf("gate → build = %+v, want round 3, candidate 4", res)
	}
	mustRefuse(t, ODD, Position{State: NeedsHuman, Round: 1, RoundLimit: 3}, Build, f, "feedback rounds")
}

func TestODDRoundLimit(t *testing.T) {
	f := oddFacts()
	f.SpecRevision = true
	for _, from := range []string{Build, Gate, Review} {
		pos := Position{State: from, Round: 3, RoundLimit: 3}
		mustRefuse(t, ODD, pos, Spec, f, "needs-human", "3 of 3")
		res := mustAdvance(t, ODD, pos, NeedsHuman, f)
		if res.Round != 3 || !strings.Contains(res.Cause, "round 3 of 3") {
			t.Errorf("%s → needs-human = %+v, want round 3 and a cause naming round 3 of 3", from, res)
		}
	}
	mustRefuse(t, ODD, Position{State: Spec, Round: 3, RoundLimit: 3}, Build, f, "needs-human")

	// The v2 needs-human moves still work in the last round.
	f = oddFacts()
	f.Gate = &GateResult{Verdict: "FAIL", Commit: "abc123"}
	pos := Position{State: Gate, Round: 3, RoundLimit: 3}
	mustRefuse(t, ODD, pos, Build, f, "needs-human", "3 of 3")
	res := mustAdvance(t, ODD, pos, NeedsHuman, f)
	if res.Cause != "gate FAIL in round 3 of 3" {
		t.Errorf("gate → needs-human cause = %q", res.Cause)
	}
	f = oddFacts()
	f.BuildStatus = "BLOCKED"
	mustAdvance(t, ODD, Position{State: Build, Round: 3, RoundLimit: 3}, NeedsHuman, f)
	f = oddFacts()
	mustRefuse(t, ODD, Position{State: Build, Round: 3, RoundLimit: 3}, NeedsHuman, f, "BLOCKED")
}

func TestODDCandidates(t *testing.T) {
	f := oddFacts()
	f.SpecRevision = true
	for _, c := range []struct {
		from, to string
		want     int
	}{
		{Spec, Build, 3},
		{Gate, Spec, 3},
		{Build, Gate, 2},
		{Gate, Review, 2},
		{Review, Approve, 2},
		{Approve, Merging, 2},
	} {
		res := mustAdvance(t, ODD, Position{State: c.from, Round: 1, RoundLimit: 3, Candidate: 2}, c.to, f)
		if res.Candidate != c.want {
			t.Errorf("ODD %s → %s candidate = %d, want %d", c.from, c.to, res.Candidate, c.want)
		}
	}
	res := mustAdvance(t, V2, Position{State: Spec, RoundLimit: 3, Candidate: 4}, Build, f)
	if res.Candidate != 4 {
		t.Errorf("V2 spec → build candidate = %d, want 4", res.Candidate)
	}
}

func TestODDStaleEvidence(t *testing.T) {
	review := Position{State: Review, Round: 1, RoundLimit: 3, Candidate: 2}
	approve := Position{State: Approve, Round: 1, RoundLimit: 3, Candidate: 2}
	for _, pos := range []Position{review, approve} {
		to := map[string]string{Review: Approve, Approve: Merging}[pos.State]

		f := oddFacts()
		f.Gate.Candidate = 1
		mustRefuse(t, ODD, pos, to, f, "gate", "candidate 1", "current candidate 2")

		f = oddFacts()
		f.ReviewCandidates = map[string]int{"code": 1}
		mustRefuse(t, ODD, pos, to, f, "review code", "candidate 1", "current candidate 2")

		f = oddFacts()
		f.ReviewCandidates = map[string]int{}
		mustRefuse(t, ODD, pos, to, f, "review code")

		mustAdvance(t, ODD, pos, to, oddFacts())
	}
	f := oddFacts()
	f.DecisionCandidate = 1
	mustRefuse(t, ODD, approve, Merging, f, "decision", "candidate 1", "current candidate 2")

	// The v2 checks still apply.
	f = oddFacts()
	f.Reviews = map[string]string{"code": "CHANGES"}
	mustRefuse(t, ODD, review, Approve, f, "CHANGES")
	f = oddFacts()
	f.Gate.Commit = "old"
	mustRefuse(t, ODD, approve, Merging, f, "HEAD")
}

func TestODDEvidenceInvalidatedByNewCandidate(t *testing.T) {
	f := oddFacts()
	f.SpecRevision = true
	res := mustAdvance(t, ODD, Position{State: Review, Round: 1, RoundLimit: 3, Candidate: 2}, Spec, f)
	res = mustAdvance(t, ODD, Position{State: res.State, Round: res.Round, RoundLimit: 3, Candidate: res.Candidate}, Build, f)
	// Evidence produced for candidate 2 is stale for the new build.
	mustRefuse(t, ODD, Position{State: Review, Round: res.Round, RoundLimit: 3, Candidate: res.Candidate}, Approve, f,
		"the gate is from candidate 2, not the current candidate 4")
}

func TestODDAdvanceDoesNotMutateFacts(t *testing.T) {
	f := oddFacts()
	f.SpecRevision = true
	before := oddFacts()
	before.SpecRevision = true
	pos := Position{State: Approve, Round: 1, RoundLimit: 3, Candidate: 2}
	posBefore := pos
	mustAdvance(t, ODD, pos, Merging, f)
	mustAdvance(t, ODD, Position{State: Review, Round: 1, RoundLimit: 3, Candidate: 2}, Spec, f)
	if !reflect.DeepEqual(f, before) {
		t.Errorf("facts changed: %+v, want %+v", f, before)
	}
	if pos != posBefore {
		t.Errorf("pos changed: %+v, want %+v", pos, posBefore)
	}
}
