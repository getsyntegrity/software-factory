package state

import (
	"fmt"
	"strings"

	"github.com/getsyntegrity/software-factory/internal/jobs"
)

// The job states.
const (
	Intake     = jobs.StateIntake
	NeedsInfo  = "needs-info"
	Spec       = "spec"
	Build      = "build"
	Gate       = "gate"
	Review     = "review"
	Approve    = "approve"
	Merging    = "merging"
	NeedsHuman = "needs-human"
	Merged     = "merged"
	Rejected   = "rejected"
)

// nextOperations maps every known state to the operation that comes next.
var nextOperations = map[string]string{
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

// Known reports whether s is one of the job states.
func Known(s string) bool {
	_, ok := nextOperations[s]
	return ok
}

// Terminal reports whether s is a state no job leaves.
func Terminal(s string) bool {
	return s == Merged || s == Rejected
}

// Waiting reports whether s is a state that waits for a person.
func Waiting(s string) bool {
	return s == NeedsInfo || s == NeedsHuman
}

// NextOperation returns the operation that follows state s, or "" for an
// unknown state.
func NextOperation(s string) string {
	return nextOperations[s]
}

// Position is where a job is: its state, its round, its round limit and the
// candidate it is on. Candidate counts the spec and build runs under ODD.
type Position struct {
	State      string
	Round      int
	RoundLimit int
	Candidate  int
}

// GateResult is the gate verdict, the commit it was run on and, under ODD,
// the candidate it was run for.
type GateResult struct {
	Verdict   string
	Commit    string
	Candidate int
}

// Facts are what the caller has already read about a job. Advance compares
// verdicts, statuses and decisions exactly; anything else fails closed.
type Facts struct {
	PRMerged          bool              // the job's PR is merged
	IntakeDecision    string            // ACCEPT, NEEDS-INFO or REJECT
	RejectionRecorded bool              // a person recorded a rejection
	SpecExists        bool              // the spec has been written
	BuildStatus       string            // READY or BLOCKED
	Head              string            // the commit at the branch HEAD
	Gate              *GateResult       // nil when the gate has not run
	ExpectedReviewers []string          // the reviewers the job waits for
	Reviews           map[string]string // reviewer to PASS or CHANGES
	Decision          string            // APPROVE or ESCALATE
	SpecRevision      bool              // ODD: the spec needs revising
	ReviewCandidates  map[string]int    // ODD: reviewer to the candidate reviewed
	DecisionCandidate int               // ODD: the candidate the decision is for
}

// Result is an accepted move: the new state, round and candidate, why the job
// moved and what runs next.
type Result struct {
	State         string
	Round         int
	Candidate     int
	Cause         string
	NextOperation string
}

// Profile is a set of legal transitions and the guards that go with them.
type Profile struct {
	transitions map[string][]string
	odd         bool // run oddGuard and count candidates
}

// V2 is the compatibility profile: the transitions and guards of factory v2.
var V2 = Profile{
	transitions: map[string][]string{
		Intake:     {NeedsInfo, Rejected, Spec},
		NeedsInfo:  {Intake, Rejected},
		Spec:       {Build},
		Build:      {Gate, NeedsHuman},
		Gate:       {Review, Build, NeedsHuman},
		Review:     {Approve, Build, NeedsHuman},
		Approve:    {Merging, NeedsHuman},
		Merging:    {NeedsHuman},
		NeedsHuman: {Build, Rejected},
	},
}

// ODD is the v2 profile plus returns to spec from build, gate and review, a
// round consumed by every build, and approve and merging bound to evidence
// from the current candidate.
var ODD = Profile{
	transitions: map[string][]string{
		Intake:     {NeedsInfo, Rejected, Spec},
		NeedsInfo:  {Intake, Rejected},
		Spec:       {Build},
		Build:      {Gate, NeedsHuman, Spec},
		Gate:       {Review, Build, NeedsHuman, Spec},
		Review:     {Approve, Build, NeedsHuman, Spec},
		Approve:    {Merging, NeedsHuman},
		Merging:    {NeedsHuman},
		NeedsHuman: {Build, Rejected},
	},
	odd: true,
}

// Allows reports whether from → to is a legal pair in p. Any known
// non-terminal state may move to merged.
func (p Profile) Allows(from, to string) bool {
	if !Known(from) || !Known(to) || Terminal(from) {
		return false
	}
	if to == Merged {
		return true
	}
	for _, s := range p.transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// Advance moves a job at pos to the state to, or returns an error saying why
// it can't. It reads only pos and facts and changes neither.
func (p Profile) Advance(pos Position, to string, facts Facts) (Result, error) {
	from := pos.State
	for _, s := range []string{from, to} {
		if !Known(s) {
			return Result{}, fmt.Errorf("state: unknown state %q", s)
		}
	}
	if !p.Allows(from, to) {
		return Result{}, fmt.Errorf("state: %s → %s is not a legal transition", from, to)
	}
	check, candidate := guard, pos.Candidate
	if p.odd {
		check = oddGuard
		if to == Spec || to == Build {
			candidate++
		}
	}
	round, cause, err := check(pos, to, facts)
	if err != nil {
		return Result{}, fmt.Errorf("state: %s → %s: %w", from, to, err)
	}
	return Result{State: to, Round: round, Candidate: candidate, Cause: cause, NextOperation: NextOperation(to)}, nil
}

// oddGuard runs the ODD guard for pos.State → to: the round limit and the
// returns to spec, then the v2 guard, then the candidate checks on evidence.
func oddGuard(pos Position, to string, facts Facts) (int, string, error) {
	from := pos.State
	reworkable := from == Build || from == Gate || from == Review
	exhausted := pos.Round >= pos.RoundLimit
	at := fmt.Sprintf("round %d of %d", pos.Round, pos.RoundLimit)
	switch {
	case to == Spec && reworkable:
		if !facts.SpecRevision {
			return 0, "", fmt.Errorf("the spec-revision fact (SpecRevision) is not set")
		}
		if exhausted {
			return 0, "", fmt.Errorf("%s: max_rounds exhausted; go to needs-human", at)
		}
		return pos.Round, "the spec is being revised in " + at, nil

	case to == Build && (from == Spec || reworkable) && exhausted:
		return 0, "", fmt.Errorf("%s: max_rounds exhausted; go to needs-human", at)

	case to == NeedsHuman && reworkable && exhausted && facts.SpecRevision:
		return pos.Round, fmt.Sprintf("spec revision needed in %s; max_rounds exhausted", at), nil
	}

	round, cause, err := guard(pos, to, facts)
	if err != nil || !(from == Review && to == Approve || from == Approve && to == Merging) {
		return round, cause, err
	}
	if err := currentEvidence(pos.Candidate, facts, to == Merging); err != nil {
		return 0, "", err
	}
	return round, cause + " for candidate " + fmt.Sprint(pos.Candidate), nil
}

// currentEvidence checks that the gate, every expected review and, when
// decision is set, the decision were produced for candidate.
func currentEvidence(candidate int, facts Facts, decision bool) error {
	stale := func(what string, c int) error {
		return fmt.Errorf("%s is from candidate %d, not the current candidate %d", what, c, candidate)
	}
	if facts.Gate == nil {
		return fmt.Errorf("the gate has not run")
	}
	if facts.Gate.Candidate != candidate {
		return stale("the gate", facts.Gate.Candidate)
	}
	for _, r := range facts.ExpectedReviewers {
		c, ok := facts.ReviewCandidates[r]
		if !ok {
			return fmt.Errorf("review %s has no candidate; the current candidate is %d", r, candidate)
		}
		if c != candidate {
			return stale("review "+r, c)
		}
	}
	if decision && facts.DecisionCandidate != candidate {
		return stale("the decision", facts.DecisionCandidate)
	}
	return nil
}

// guard runs the v2 guard for pos.State → to and returns the new round and
// the cause of the move.
func guard(pos Position, to string, facts Facts) (int, string, error) {
	from, round := pos.State, pos.Round
	switch {
	case to == Merged:
		if !facts.PRMerged {
			return 0, "", fmt.Errorf("the PR isn't merged")
		}
		return round, "the PR is merged", nil

	case from == Intake:
		want := map[string]string{NeedsInfo: "NEEDS-INFO", Rejected: "REJECT", Spec: "ACCEPT"}[to]
		if facts.IntakeDecision != want {
			return 0, "", fmt.Errorf("the intake decision is %q, not %s", facts.IntakeDecision, want)
		}
		return round, "intake decided " + want, nil

	case to == Rejected: // from needs-info or needs-human
		if !facts.RejectionRecorded {
			return 0, "", fmt.Errorf("no rejection was recorded")
		}
		return round, "a rejection was recorded", nil

	case from == NeedsInfo: // → intake
		return round, "the author answered; intake runs again", nil

	case from == Spec:
		if !facts.SpecExists {
			return 0, "", fmt.Errorf("the spec doesn't exist")
		}
		return round + 1, fmt.Sprintf("the spec exists; round %d starts", round+1), nil

	case from == Build && to == Gate:
		if facts.BuildStatus != "READY" {
			return 0, "", fmt.Errorf("the build status is %q, not READY", facts.BuildStatus)
		}
		return round, "the build is READY", nil

	case from == Build: // → needs-human
		if facts.BuildStatus != "BLOCKED" {
			return 0, "", fmt.Errorf("the build status is %q, not BLOCKED", facts.BuildStatus)
		}
		return round, "the build is blocked", nil

	case from == Gate:
		g := facts.Gate
		if g == nil {
			return 0, "", fmt.Errorf("the gate has not run")
		}
		if facts.Head == "" {
			return 0, "", fmt.Errorf("the HEAD commit is unknown")
		}
		if g.Commit != facts.Head {
			return 0, "", fmt.Errorf("the gate ran on %q, not HEAD %q; run the gate again", g.Commit, facts.Head)
		}
		if to == Review {
			if g.Verdict != "PASS" {
				return 0, "", fmt.Errorf("the gate verdict is %q, not PASS", g.Verdict)
			}
			return round, "gate PASS on " + g.Commit, nil
		}
		if g.Verdict != "FAIL" {
			return 0, "", fmt.Errorf("the gate verdict is %q, not FAIL", g.Verdict)
		}
		return nextRound(pos, to, "gate FAIL")

	case from == Review:
		changes, err := reviewChanges(facts)
		if err != nil {
			return 0, "", err
		}
		if to == Approve {
			if len(changes) > 0 {
				return 0, "", fmt.Errorf("reviews want CHANGES: %s", strings.Join(changes, ", "))
			}
			return round, "every review passed", nil
		}
		if len(changes) == 0 {
			return 0, "", fmt.Errorf("every review passed")
		}
		return nextRound(pos, to, "review CHANGES from "+strings.Join(changes, ", "))

	case from == Approve && to == Merging:
		if facts.Decision != "APPROVE" {
			return 0, "", fmt.Errorf("the decision is %q, not APPROVE", facts.Decision)
		}
		if g := facts.Gate; g == nil || g.Verdict != "PASS" || facts.Head == "" || g.Commit != facts.Head {
			return 0, "", fmt.Errorf("the gate didn't pass on HEAD %q", facts.Head)
		}
		changes, err := reviewChanges(facts)
		if err != nil {
			return 0, "", err
		}
		if len(changes) > 0 {
			return 0, "", fmt.Errorf("reviews want CHANGES: %s", strings.Join(changes, ", "))
		}
		return round, "decision APPROVE on a passing candidate", nil

	case from == Approve: // → needs-human
		if facts.Decision != "ESCALATE" {
			return 0, "", fmt.Errorf("the decision is %q, not ESCALATE", facts.Decision)
		}
		return round, "decision ESCALATE", nil

	case from == Merging: // → needs-human
		return round, "the merge needs a human", nil

	case from == NeedsHuman: // → build
		return 0, "", fmt.Errorf("feedback rounds are not supported yet")
	}
	return 0, "", fmt.Errorf("no guard for this move")
}

// nextRound settles a failed round: build with round + 1 before the last
// round, needs-human in it. A wrong target is refused naming the right one.
func nextRound(pos Position, to, what string) (int, string, error) {
	last := pos.Round >= pos.RoundLimit
	want := Build
	if last {
		want = NeedsHuman
	}
	if to != want {
		return 0, "", fmt.Errorf("round %d of %d: go to %s", pos.Round, pos.RoundLimit, want)
	}
	cause := fmt.Sprintf("%s in round %d of %d", what, pos.Round, pos.RoundLimit)
	if last {
		return pos.Round, cause, nil
	}
	return pos.Round + 1, fmt.Sprintf("%s; round %d starts", cause, pos.Round+1), nil
}

// reviewChanges returns the expected reviewers that want CHANGES, or an error
// when the list is empty, a verdict is missing or a verdict is neither PASS
// nor CHANGES.
func reviewChanges(facts Facts) ([]string, error) {
	if len(facts.ExpectedReviewers) == 0 {
		return nil, fmt.Errorf("no reviewers are expected")
	}
	var changes []string
	for _, r := range facts.ExpectedReviewers {
		v, ok := facts.Reviews[r]
		switch {
		case !ok:
			return nil, fmt.Errorf("reviewer %s has no verdict", r)
		case v == "CHANGES":
			changes = append(changes, r)
		case v != "PASS":
			return nil, fmt.Errorf("reviewer %s's verdict is %q, not PASS or CHANGES", r, v)
		}
	}
	return changes, nil
}
