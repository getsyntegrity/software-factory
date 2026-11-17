package state

import (
	"testing"

	"github.com/getsyntegrity/go-specs/specs"
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

// runGuardCases registers one spec per case, named after the case.
func runGuardCases(s *specs.Spec, cases []guardCase) {
	for _, c := range cases {
		s.It(c.name, func(ctx *specs.Context) {
			facts := passingFacts()
			if c.facts != nil {
				c.facts(&facts)
			}
			got, err := V2.Advance(c.pos, c.to, facts)
			if c.accept {
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(got.State).ToEqual(c.want.State)
				ctx.Expect(got.Round).ToEqual(c.want.Round)
				ctx.Expect(got.NextOperation).ToEqual(NextOperation(c.want.State))
				ctx.Expect(got.Cause).To(specs.Not(specs.BeEmpty()))
				if c.want.Cause != "" {
					ctx.Expect(got.Cause).To(specs.Contain(c.want.Cause))
				}
				return
			}
			ctx.Expect(err).To(specs.Not(specs.BeNil()))
			if c.errHas != "" {
				ctx.Expect(err.Error()).To(specs.Contain(c.errHas))
			}
		})
	}
}

func TestStateMachine(t *testing.T) {
	specs.Describe(t, "the v2 job state machine", func(s *specs.Spec) {
		s.Describe("naming the job states", func(s *specs.Spec) {
			s.It("lists the eleven states in order", func(ctx *specs.Context) {
				want := []string{"intake", "needs-info", "spec", "build", "gate", "review", "approve", "merging", "needs-human", "merged", "rejected"}
				ctx.Expect(allStates).ToEqual(want)
			})

			s.It("shares the intake state with the job record", func(ctx *specs.Context) {
				ctx.Expect(Intake).ToEqual(jobs.StateIntake)
			})

			s.It("marks merged and rejected terminal and needs-info and needs-human waiting", func(ctx *specs.Context) {
				for _, st := range allStates {
					ctx.Expect(Terminal(st)).ToEqual(st == Merged || st == Rejected)
					ctx.Expect(Waiting(st)).ToEqual(st == NeedsInfo || st == NeedsHuman)
				}
			})
		})

		s.Describe("deciding which moves the v2 table allows", func(s *specs.Spec) {
			s.It("allows exactly the pairs of the table among all 121", func(ctx *specs.Context) {
				count := 0
				for _, from := range allStates {
					for _, to := range allStates {
						count++
						ctx.Expect(V2.Allows(from, to)).ToEqual(legal(from, to))
					}
				}
				ctx.Expect(count).ToEqual(121)
			})

			s.It("allows no move from or to an unknown state", func(ctx *specs.Context) {
				for _, unknown := range []string{"", "bogus", "Build", "MERGED"} {
					for _, st := range allStates {
						ctx.Expect(V2.Allows(unknown, st)).To(specs.BeFalse())
						ctx.Expect(V2.Allows(st, unknown)).To(specs.BeFalse())
					}
				}
			})
		})

		s.Describe("advancing a job to a state the table does not allow", func(s *specs.Spec) {
			for _, from := range allStates {
				for _, to := range allStates {
					if legal(from, to) {
						continue
					}
					s.It("refuses "+from+" to "+to+" naming both states", func(ctx *specs.Context) {
						_, err := V2.Advance(Position{State: from, Round: 1, RoundLimit: 3}, to, passingFacts())
						ctx.Expect(err).To(specs.Not(specs.BeNil()))
						ctx.Expect(err.Error()).To(specs.Contain(from))
						ctx.Expect(err.Error()).To(specs.Contain(to))
					})
				}
			}
		})

		s.Describe("advancing from or to an unknown state", func(s *specs.Spec) {
			s.It("refuses an unknown source and names it", func(ctx *specs.Context) {
				_, err := V2.Advance(Position{State: "bogus"}, Merged, passingFacts())
				ctx.Expect(err).To(specs.Not(specs.BeNil()))
				ctx.Expect(err.Error()).To(specs.Contain("bogus"))
			})

			s.It("refuses an unknown target and names it", func(ctx *specs.Context) {
				_, err := V2.Advance(Position{State: Intake}, "bogus", passingFacts())
				ctx.Expect(err).To(specs.Not(specs.BeNil()))
				ctx.Expect(err.Error()).To(specs.Contain("bogus"))
			})
		})

		s.Describe("moving any open job to merged once its PR is merged", func(s *specs.Spec) {
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
			runGuardCases(s, cases)
		})

		s.Describe("leaving intake according to the intake decision", func(s *specs.Spec) {
			pos := Position{State: Intake}
			decide := func(d string) func(*Facts) { return func(f *Facts) { f.IntakeDecision = d } }
			runGuardCases(s, []guardCase{
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
		})

		s.Describe("returning a job from needs-info to intake", func(s *specs.Spec) {
			runGuardCases(s, []guardCase{
				{name: "no guard", pos: Position{State: NeedsInfo}, to: Intake, facts: func(f *Facts) { *f = Facts{} }, accept: true, want: Result{State: Intake}},
			})
		})

		s.Describe("rejecting a job that waits on a person", func(s *specs.Spec) {
			var cases []guardCase
			for _, from := range []string{NeedsInfo, NeedsHuman} {
				pos := Position{State: from, Round: 2, RoundLimit: 3}
				cases = append(cases,
					guardCase{name: from + " recorded", pos: pos, to: Rejected, accept: true, want: Result{State: Rejected, Round: 2}},
					guardCase{name: from + " not recorded", pos: pos, to: Rejected, facts: func(f *Facts) { f.RejectionRecorded = false }},
				)
			}
			runGuardCases(s, cases)
		})

		s.Describe("starting the first build once the spec exists", func(s *specs.Spec) {
			pos := Position{State: Spec, Round: 0, RoundLimit: 3}
			runGuardCases(s, []guardCase{
				{name: "spec exists", pos: pos, to: Build, accept: true, want: Result{State: Build, Round: 1}},
				{name: "no spec", pos: pos, to: Build, facts: func(f *Facts) { f.SpecExists = false }},
			})
		})

		s.Describe("leaving build according to the build status", func(s *specs.Spec) {
			pos := Position{State: Build, Round: 1, RoundLimit: 3}
			status := func(st string) func(*Facts) { return func(f *Facts) { f.BuildStatus = st } }
			runGuardCases(s, []guardCase{
				{name: "ready", pos: pos, to: Gate, facts: status("READY"), accept: true, want: Result{State: Gate, Round: 1}},
				{name: "blocked", pos: pos, to: NeedsHuman, facts: status("BLOCKED"), accept: true, want: Result{State: NeedsHuman, Round: 1, Cause: "blocked"}},
				{name: "gate on blocked", pos: pos, to: Gate, facts: status("BLOCKED")},
				{name: "needs-human on ready", pos: pos, to: NeedsHuman, facts: status("READY")},
				{name: "empty status gate", pos: pos, to: Gate, facts: status("")},
				{name: "empty status needs-human", pos: pos, to: NeedsHuman, facts: status("")},
				{name: "lower case", pos: pos, to: Gate, facts: status("ready")},
			})
		})

		s.Describe("leaving the gate according to its verdict and the round", func(s *specs.Spec) {
			gate := func(verdict, commit string) func(*Facts) {
				return func(f *Facts) { f.Gate = &GateResult{Verdict: verdict, Commit: commit} }
			}
			r1 := Position{State: Gate, Round: 1, RoundLimit: 3}
			r3 := Position{State: Gate, Round: 3, RoundLimit: 3}
			runGuardCases(s, []guardCase{
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
		})

		s.Describe("leaving review according to the reviewers' verdicts and the round", func(s *specs.Spec) {
			reviews := func(expected []string, got map[string]string) func(*Facts) {
				return func(f *Facts) { f.ExpectedReviewers = expected; f.Reviews = got }
			}
			two := []string{"code", "ux"}
			allPass := map[string]string{"code": "PASS", "ux": "PASS"}
			changes := map[string]string{"code": "PASS", "ux": "CHANGES"}
			r1 := Position{State: Review, Round: 1, RoundLimit: 3}
			r3 := Position{State: Review, Round: 3, RoundLimit: 3}
			runGuardCases(s, []guardCase{
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
		})

		s.Describe("leaving approve according to the decision and the evidence", func(s *specs.Spec) {
			pos := Position{State: Approve, Round: 2, RoundLimit: 3}
			runGuardCases(s, []guardCase{
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
		})

		s.Describe("escalating a merge to a person", func(s *specs.Spec) {
			runGuardCases(s, []guardCase{
				{name: "no guard", pos: Position{State: Merging, Round: 2, RoundLimit: 3}, to: NeedsHuman, facts: func(f *Facts) { *f = Facts{} }, accept: true, want: Result{State: NeedsHuman, Round: 2}},
			})
		})

		s.Describe("resuming a build from needs-human", func(s *specs.Spec) {
			s.It("is in the table but refused with a feedback error", func(ctx *specs.Context) {
				ctx.Expect(V2.Allows(NeedsHuman, Build)).To(specs.BeTrue())
				_, err := V2.Advance(Position{State: NeedsHuman, Round: 3, RoundLimit: 3}, Build, passingFacts())
				ctx.Expect(err).To(specs.Not(specs.BeNil()))
				ctx.Expect(err.Error()).To(specs.Contain("feedback"))
			})
		})

		s.Describe("naming the next operation of a state", func(s *specs.Spec) {
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

			s.It("names the operation of every state", func(ctx *specs.Context) {
				for _, st := range allStates {
					ctx.Expect(NextOperation(st)).ToEqual(want[st])
				}
			})

			s.It("names no operation for an unknown state", func(ctx *specs.Context) {
				ctx.Expect(NextOperation("bogus")).ToEqual("")
			})

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
				s.It("gives "+c.pos.State+" to "+c.to+" a cause and the next operation", func(ctx *specs.Context) {
					facts := passingFacts()
					if c.facts != nil {
						c.facts(&facts)
					}
					got, err := V2.Advance(c.pos, c.to, facts)
					ctx.Expect(err).To(specs.BeNil())
					ctx.Expect(got.State).ToEqual(c.to)
					ctx.Expect(got.Cause).To(specs.Not(specs.BeEmpty()))
					ctx.Expect(got.NextOperation).ToEqual(want[c.to])
				})
			}
		})

		s.Describe("keeping the caller's facts unchanged", func(s *specs.Spec) {
			s.It("leaves the facts as they were after a move", func(ctx *specs.Context) {
				facts := passingFacts()
				facts.Reviews = map[string]string{"code": "CHANGES"}
				before := passingFacts()
				before.Reviews = map[string]string{"code": "CHANGES"}
				_, err := V2.Advance(Position{State: Review, Round: 1, RoundLimit: 3}, Build, facts)
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(facts).ToEqual(before)
			})
		})
	})
}
