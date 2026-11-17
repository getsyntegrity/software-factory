package state

import (
	"reflect"
	"strings"
	"testing"

	"github.com/getsyntegrity/software-factory/internal/jobs"
)

var allStates = []string{
	Intake, NeedsInfo, Spec, Build, Gate, Review, Approve, Merging, NeedsHuman, Merged, Rejected,
}

// v2Pairs is the v2 TRANSITIONS table plus X → merged for every non-terminal X.
var v2Pairs = map[string][]string{
	Intake:     {NeedsInfo, Rejected, Spec, Merged},
	NeedsInfo:  {Intake, Rejected, Merged},
	Spec:       {Build, Merged},
	Build:      {Gate, NeedsHuman, Merged},
	Gate:       {Review, Build, NeedsHuman, Merged},
	Review:     {Approve, Build, NeedsHuman, Merged},
	Approve:    {Merging, NeedsHuman, Merged},
	Merging:    {NeedsHuman, Merged},
	NeedsHuman: {Build, Rejected, Merged},
	Merged:     {},
	Rejected:   {},
}

func legal(from, to string) bool {
	for _, s := range v2Pairs[from] {
		if s == to {
			return true
		}
	}
	return false
}

// passingFacts satisfies every guard that can be satisfied at once.
func passingFacts() Facts {
	return Facts{
		PRMerged:          true,
		IntakeDecision:    "ACCEPT",
		RejectionRecorded: true,
		SpecExists:        true,
		BuildStatus:       "READY",
		Head:              "abc123",
		Gate:              &GateResult{Verdict: "PASS", Commit: "abc123"},
		ExpectedReviewers: []string{"code"},
		Reviews:           map[string]string{"code": "PASS"},
		Decision:          "APPROVE",
	}
}

func TestStateValues(t *testing.T) {
	want := []string{"intake", "needs-info", "spec", "build", "gate", "review", "approve", "merging", "needs-human", "merged", "rejected"}
	if !reflect.DeepEqual(allStates, want) {
		t.Errorf("states = %v, want %v", allStates, want)
	}
	if Intake != jobs.StateIntake {
		t.Errorf("Intake = %q, jobs.StateIntake = %q", Intake, jobs.StateIntake)
	}
	for _, s := range allStates {
		terminal := s == Merged || s == Rejected
		waiting := s == NeedsInfo || s == NeedsHuman
		if Terminal(s) != terminal {
			t.Errorf("Terminal(%q) = %v, want %v", s, Terminal(s), terminal)
		}
		if Waiting(s) != waiting {
			t.Errorf("Waiting(%q) = %v, want %v", s, Waiting(s), waiting)
		}
	}
}

func TestAllowsTable(t *testing.T) {
	count := 0
	for _, from := range allStates {
		for _, to := range allStates {
			count++
			if got, want := V2.Allows(from, to), legal(from, to); got != want {
				t.Errorf("V2.Allows(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}
	if count != 121 {
		t.Fatalf("checked %d pairs, want 121", count)
	}
	for _, unknown := range []string{"", "bogus", "Build", "MERGED"} {
		for _, s := range allStates {
			if V2.Allows(unknown, s) {
				t.Errorf("V2.Allows(%q, %q) = true", unknown, s)
			}
			if V2.Allows(s, unknown) {
				t.Errorf("V2.Allows(%q, %q) = true", s, unknown)
			}
		}
	}
}

func TestAdvanceRefusesIllegalPairs(t *testing.T) {
	for _, from := range allStates {
		for _, to := range allStates {
			if legal(from, to) {
				continue
			}
			_, err := V2.Advance(Position{State: from, Round: 1, RoundLimit: 3}, to, passingFacts())
			if err == nil {
				t.Errorf("Advance %s → %s: no error", from, to)
				continue
			}
			if msg := err.Error(); !strings.Contains(msg, from) || !strings.Contains(msg, to) {
				t.Errorf("Advance %s → %s: error %q does not name both states", from, to, msg)
			}
		}
	}
}

func TestAdvanceRefusesUnknownStates(t *testing.T) {
	if _, err := V2.Advance(Position{State: "bogus"}, Merged, passingFacts()); err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Errorf("unknown source: err = %v", err)
	}
	if _, err := V2.Advance(Position{State: Intake}, "bogus", passingFacts()); err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Errorf("unknown target: err = %v", err)
	}
}

// guardCase is one Advance call and whether it must be accepted.
type guardCase struct {
	name   string
	pos    Position
	to     string
	facts  func(*Facts)
	accept bool
	want   Result // checked only when accept is true
	errHas string // checked only when accept is false
}

func runGuardCases(t *testing.T, cases []guardCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			facts := passingFacts()
			if c.facts != nil {
				c.facts(&facts)
			}
			got, err := V2.Advance(c.pos, c.to, facts)
			if c.accept {
				if err != nil {
					t.Fatalf("Advance: %v", err)
				}
				if got.State != c.want.State || got.Round != c.want.Round || got.NextOperation != NextOperation(c.want.State) {
					t.Errorf("Advance = %+v, want state %q round %d", got, c.want.State, c.want.Round)
				}
				if got.Cause == "" || (c.want.Cause != "" && !strings.Contains(got.Cause, c.want.Cause)) {
					t.Errorf("cause = %q, want it to contain %q", got.Cause, c.want.Cause)
				}
				return
			}
			if err == nil {
				t.Fatalf("Advance accepted: %+v", got)
			}
			if c.errHas != "" && !strings.Contains(err.Error(), c.errHas) {
				t.Errorf("error %q does not contain %q", err, c.errHas)
			}
		})
	}
}

func TestGuardMerged(t *testing.T) {
	var cases []guardCase
	for _, from := range allStates {
		if Terminal(from) {
			continue
		}
		pos := Position{State: from, Round: 2, RoundLimit: 3}
		cases = append(cases,
			guardCase{name: from + " merged", pos: pos, to: Merged, accept: true, want: Result{State: Merged, Round: 2}},
			guardCase{name: from + " not merged", pos: pos, to: Merged, facts: func(f *Facts) { f.PRMerged = false }},
		)
	}
	runGuardCases(t, cases)
}

func TestGuardIntake(t *testing.T) {
	pos := Position{State: Intake}
	decide := func(d string) func(*Facts) { return func(f *Facts) { f.IntakeDecision = d } }
	runGuardCases(t, []guardCase{
		{name: "accept", pos: pos, to: Spec, facts: decide("ACCEPT"), accept: true, want: Result{State: Spec}},
		{name: "needs info", pos: pos, to: NeedsInfo, facts: decide("NEEDS-INFO"), accept: true, want: Result{State: NeedsInfo}},
		{name: "reject", pos: pos, to: Rejected, facts: decide("REJECT"), accept: true, want: Result{State: Rejected}},
		{name: "spec on reject", pos: pos, to: Spec, facts: decide("REJECT"), errHas: "ACCEPT"},
		{name: "needs info on accept", pos: pos, to: NeedsInfo, facts: decide("ACCEPT")},
		{name: "rejected on needs info", pos: pos, to: Rejected, facts: decide("NEEDS-INFO")},
		{name: "empty decision", pos: pos, to: Spec, facts: decide("")},
		{name: "lower case", pos: pos, to: Spec, facts: decide("accept")},
		{name: "unknown decision", pos: pos, to: Rejected, facts: decide("NO")},
	})
}

func TestGuardNeedsInfoToIntake(t *testing.T) {
	runGuardCases(t, []guardCase{
		{name: "no guard", pos: Position{State: NeedsInfo}, to: Intake, facts: func(f *Facts) { *f = Facts{} }, accept: true, want: Result{State: Intake}},
	})
}

func TestGuardRejected(t *testing.T) {
	var cases []guardCase
	for _, from := range []string{NeedsInfo, NeedsHuman} {
		pos := Position{State: from, Round: 2, RoundLimit: 3}
		cases = append(cases,
			guardCase{name: from + " recorded", pos: pos, to: Rejected, accept: true, want: Result{State: Rejected, Round: 2}},
			guardCase{name: from + " not recorded", pos: pos, to: Rejected, facts: func(f *Facts) { f.RejectionRecorded = false }},
		)
	}
	runGuardCases(t, cases)
}

func TestGuardSpecToBuild(t *testing.T) {
	pos := Position{State: Spec, Round: 0, RoundLimit: 3}
	runGuardCases(t, []guardCase{
		{name: "spec exists", pos: pos, to: Build, accept: true, want: Result{State: Build, Round: 1}},
		{name: "no spec", pos: pos, to: Build, facts: func(f *Facts) { f.SpecExists = false }},
	})
}

func TestGuardBuild(t *testing.T) {
	pos := Position{State: Build, Round: 1, RoundLimit: 3}
	status := func(s string) func(*Facts) { return func(f *Facts) { f.BuildStatus = s } }
	runGuardCases(t, []guardCase{
		{name: "ready", pos: pos, to: Gate, facts: status("READY"), accept: true, want: Result{State: Gate, Round: 1}},
		{name: "blocked", pos: pos, to: NeedsHuman, facts: status("BLOCKED"), accept: true, want: Result{State: NeedsHuman, Round: 1, Cause: "blocked"}},
		{name: "gate on blocked", pos: pos, to: Gate, facts: status("BLOCKED")},
		{name: "needs-human on ready", pos: pos, to: NeedsHuman, facts: status("READY")},
		{name: "empty status gate", pos: pos, to: Gate, facts: status("")},
		{name: "empty status needs-human", pos: pos, to: NeedsHuman, facts: status("")},
		{name: "lower case", pos: pos, to: Gate, facts: status("ready")},
	})
}

func TestGuardGate(t *testing.T) {
	gate := func(verdict, commit string) func(*Facts) {
		return func(f *Facts) { f.Gate = &GateResult{Verdict: verdict, Commit: commit} }
	}
	r1 := Position{State: Gate, Round: 1, RoundLimit: 3}
	r3 := Position{State: Gate, Round: 3, RoundLimit: 3}
	runGuardCases(t, []guardCase{
		{name: "pass", pos: r1, to: Review, facts: gate("PASS", "abc123"), accept: true, want: Result{State: Review, Round: 1}},
		{name: "review on fail", pos: r1, to: Review, facts: gate("FAIL", "abc123")},
		{name: "fail round 1", pos: r1, to: Build, facts: gate("FAIL", "abc123"), accept: true, want: Result{State: Build, Round: 2}},
		{name: "fail round 1 needs-human", pos: r1, to: NeedsHuman, facts: gate("FAIL", "abc123"), errHas: "build"},
		{name: "fail round 3", pos: r3, to: NeedsHuman, facts: gate("FAIL", "abc123"), accept: true, want: Result{State: NeedsHuman, Round: 3, Cause: "gate FAIL in round 3 of 3"}},
		{name: "fail round 3 build", pos: r3, to: Build, facts: gate("FAIL", "abc123"), errHas: "needs-human"},
		{name: "build on pass", pos: r1, to: Build, facts: gate("PASS", "abc123")},
		{name: "needs-human on pass", pos: r3, to: NeedsHuman, facts: gate("PASS", "abc123")},
		{name: "no gate result", pos: r1, to: Review, facts: func(f *Facts) { f.Gate = nil }},
		{name: "empty verdict", pos: r1, to: Review, facts: gate("", "abc123")},
		{name: "empty verdict build", pos: r1, to: Build, facts: gate("", "abc123")},
		{name: "lower case", pos: r1, to: Review, facts: gate("pass", "abc123")},
		{name: "other commit", pos: r1, to: Review, facts: gate("PASS", "def456")},
		{name: "other commit fail", pos: r1, to: Build, facts: gate("FAIL", "def456")},
		{name: "empty head", pos: r1, to: Review, facts: func(f *Facts) { f.Head = ""; f.Gate = &GateResult{Verdict: "PASS"} }},
	})
}

func TestGuardReview(t *testing.T) {
	reviews := func(expected []string, got map[string]string) func(*Facts) {
		return func(f *Facts) { f.ExpectedReviewers = expected; f.Reviews = got }
	}
	two := []string{"code", "ux"}
	allPass := map[string]string{"code": "PASS", "ux": "PASS"}
	changes := map[string]string{"code": "PASS", "ux": "CHANGES"}
	r1 := Position{State: Review, Round: 1, RoundLimit: 3}
	r3 := Position{State: Review, Round: 3, RoundLimit: 3}
	runGuardCases(t, []guardCase{
		{name: "all pass", pos: r1, to: Approve, facts: reviews(two, allPass), accept: true, want: Result{State: Approve, Round: 1}},
		{name: "approve on changes", pos: r1, to: Approve, facts: reviews(two, changes), errHas: "ux"},
		{name: "changes round 1", pos: r1, to: Build, facts: reviews(two, changes), accept: true, want: Result{State: Build, Round: 2}},
		{name: "changes round 1 needs-human", pos: r1, to: NeedsHuman, facts: reviews(two, changes), errHas: "build"},
		{name: "changes round 3", pos: r3, to: NeedsHuman, facts: reviews(two, changes), accept: true, want: Result{State: NeedsHuman, Round: 3, Cause: "round 3 of 3"}},
		{name: "changes round 3 build", pos: r3, to: Build, facts: reviews(two, changes), errHas: "needs-human"},
		{name: "build on all pass", pos: r1, to: Build, facts: reviews(two, allPass)},
		{name: "missing reviewer", pos: r1, to: Approve, facts: reviews(two, map[string]string{"code": "PASS"}), errHas: "ux"},
		{name: "missing reviewer build", pos: r1, to: Build, facts: reviews(two, map[string]string{"code": "CHANGES"})},
		{name: "lower case", pos: r1, to: Approve, facts: reviews(two, map[string]string{"code": "PASS", "ux": "pass"})},
		{name: "LGTM", pos: r1, to: Approve, facts: reviews(two, map[string]string{"code": "PASS", "ux": "LGTM"})},
		{name: "empty verdict", pos: r1, to: Approve, facts: reviews(two, map[string]string{"code": "PASS", "ux": ""})},
		{name: "no expected reviewers", pos: r1, to: Approve, facts: reviews(nil, allPass)},
	})
}

func TestGuardApprove(t *testing.T) {
	pos := Position{State: Approve, Round: 2, RoundLimit: 3}
	runGuardCases(t, []guardCase{
		{name: "approve", pos: pos, to: Merging, accept: true, want: Result{State: Merging, Round: 2}},
		{name: "escalate", pos: pos, to: NeedsHuman, facts: func(f *Facts) { f.Decision = "ESCALATE" }, accept: true, want: Result{State: NeedsHuman, Round: 2}},
		{name: "merging on escalate", pos: pos, to: Merging, facts: func(f *Facts) { f.Decision = "ESCALATE" }},
		{name: "needs-human on approve", pos: pos, to: NeedsHuman},
		{name: "empty decision", pos: pos, to: Merging, facts: func(f *Facts) { f.Decision = "" }},
		{name: "empty decision needs-human", pos: pos, to: NeedsHuman, facts: func(f *Facts) { f.Decision = "" }},
		{name: "lower case", pos: pos, to: Merging, facts: func(f *Facts) { f.Decision = "approve" }},
		{name: "gate fail", pos: pos, to: Merging, facts: func(f *Facts) { f.Gate = &GateResult{Verdict: "FAIL", Commit: "abc123"} }},
		{name: "gate other commit", pos: pos, to: Merging, facts: func(f *Facts) { f.Gate = &GateResult{Verdict: "PASS", Commit: "def456"} }},
		{name: "no gate", pos: pos, to: Merging, facts: func(f *Facts) { f.Gate = nil }},
		{name: "empty head", pos: pos, to: Merging, facts: func(f *Facts) { f.Head = ""; f.Gate = &GateResult{Verdict: "PASS"} }},
		{name: "review changes", pos: pos, to: Merging, facts: func(f *Facts) { f.Reviews = map[string]string{"code": "CHANGES"} }},
		{name: "review missing", pos: pos, to: Merging, facts: func(f *Facts) { f.ExpectedReviewers = []string{"code", "ux"} }},
		{name: "no expected reviewers", pos: pos, to: Merging, facts: func(f *Facts) { f.ExpectedReviewers = nil }},
	})
}

func TestGuardMergingToNeedsHuman(t *testing.T) {
	runGuardCases(t, []guardCase{
		{name: "no guard", pos: Position{State: Merging, Round: 2, RoundLimit: 3}, to: NeedsHuman, facts: func(f *Facts) { *f = Facts{} }, accept: true, want: Result{State: NeedsHuman, Round: 2}},
	})
}

func TestNeedsHumanToBuildRefused(t *testing.T) {
	if !V2.Allows(NeedsHuman, Build) {
		t.Errorf("V2.Allows(needs-human, build) = false")
	}
	_, err := V2.Advance(Position{State: NeedsHuman, Round: 3, RoundLimit: 3}, Build, passingFacts())
	if err == nil || !strings.Contains(err.Error(), "feedback") {
		t.Errorf("Advance needs-human → build: err = %v, want a feedback error", err)
	}
}

func TestNextOperation(t *testing.T) {
	want := map[string]string{
		Intake:     jobs.NextRunIntake,
		NeedsInfo:  "await_author",
		Spec:       "run_spec",
		Build:      "run_build",
		Gate:       "run_gate",
		Review:     "run_review",
		Approve:    "run_approve",
		Merging:    "run_merge",
		NeedsHuman: "await_human",
		Merged:     "none",
		Rejected:   "none",
	}
	for _, s := range allStates {
		if got := NextOperation(s); got != want[s] {
			t.Errorf("NextOperation(%q) = %q, want %q", s, got, want[s])
		}
	}
	if got := NextOperation("bogus"); got != "" {
		t.Errorf("NextOperation(bogus) = %q, want empty", got)
	}
	// Every accepted move carries a cause and the next operation of its state.
	accepted := []struct {
		pos   Position
		to    string
		facts func(*Facts)
	}{
		{Position{State: Intake}, Spec, nil},
		{Position{State: Intake}, NeedsInfo, func(f *Facts) { f.IntakeDecision = "NEEDS-INFO" }},
		{Position{State: Intake}, Rejected, func(f *Facts) { f.IntakeDecision = "REJECT" }},
		{Position{State: NeedsInfo}, Intake, nil},
		{Position{State: NeedsHuman, Round: 3, RoundLimit: 3}, Rejected, nil},
		{Position{State: Spec, RoundLimit: 3}, Build, nil},
		{Position{State: Build, Round: 1, RoundLimit: 3}, Gate, nil},
		{Position{State: Build, Round: 1, RoundLimit: 3}, NeedsHuman, func(f *Facts) { f.BuildStatus = "BLOCKED" }},
		{Position{State: Gate, Round: 1, RoundLimit: 3}, Review, nil},
		{Position{State: Gate, Round: 1, RoundLimit: 3}, Build, func(f *Facts) { f.Gate.Verdict = "FAIL" }},
		{Position{State: Gate, Round: 3, RoundLimit: 3}, NeedsHuman, func(f *Facts) { f.Gate.Verdict = "FAIL" }},
		{Position{State: Review, Round: 1, RoundLimit: 3}, Approve, nil},
		{Position{State: Review, Round: 1, RoundLimit: 3}, Build, func(f *Facts) { f.Reviews = map[string]string{"code": "CHANGES"} }},
		{Position{State: Review, Round: 3, RoundLimit: 3}, NeedsHuman, func(f *Facts) { f.Reviews = map[string]string{"code": "CHANGES"} }},
		{Position{State: Approve, Round: 1, RoundLimit: 3}, Merging, nil},
		{Position{State: Approve, Round: 1, RoundLimit: 3}, NeedsHuman, func(f *Facts) { f.Decision = "ESCALATE" }},
		{Position{State: Merging, Round: 1, RoundLimit: 3}, NeedsHuman, nil},
		{Position{State: Merging, Round: 1, RoundLimit: 3}, Merged, nil},
	}
	for _, c := range accepted {
		facts := passingFacts()
		if c.facts != nil {
			c.facts(&facts)
		}
		got, err := V2.Advance(c.pos, c.to, facts)
		if err != nil {
			t.Errorf("Advance %s → %s: %v", c.pos.State, c.to, err)
			continue
		}
		if got.State != c.to || got.Cause == "" || got.NextOperation != want[c.to] {
			t.Errorf("Advance %s → %s = %+v, want state %q, a cause and next operation %q", c.pos.State, c.to, got, c.to, want[c.to])
		}
	}
}

func TestAdvanceDoesNotMutateFacts(t *testing.T) {
	facts := passingFacts()
	facts.Reviews = map[string]string{"code": "CHANGES"}
	before := passingFacts()
	before.Reviews = map[string]string{"code": "CHANGES"}
	if _, err := V2.Advance(Position{State: Review, Round: 1, RoundLimit: 3}, Build, facts); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(facts, before) {
		t.Errorf("facts changed: %+v, want %+v", facts, before)
	}
}
