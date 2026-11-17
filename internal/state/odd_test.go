package state

import (
	"testing"

	"github.com/getsyntegrity/go-specs/specs"
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

func mustAdvance(ctx *specs.Context, p Profile, pos Position, to string, f Facts) Result {
	res, err := p.Advance(pos, to, f)
	ctx.Expect(err).To(specs.BeNil())
	return res
}

func mustRefuse(ctx *specs.Context, p Profile, pos Position, to string, f Facts, wants ...string) {
	_, err := p.Advance(pos, to, f)
	ctx.Expect(err).To(specs.Not(specs.BeNil()))
	for _, w := range wants {
		ctx.Expect(err.Error()).To(specs.Contain(w))
	}
}

func TestODDProfile(t *testing.T) {
	specs.Describe(t, "the ODD job state machine", func(s *specs.Spec) {
		s.Describe("deciding which moves the ODD table allows", func(s *specs.Spec) {
			s.It("allows exactly the pairs of the table among all 121", func(ctx *specs.Context) {
				count := 0
				for _, from := range allStates {
					for _, to := range allStates {
						count++
						want := false
						for _, st := range oddPairs[from] {
							want = want || st == to
						}
						ctx.Expect(ODD.Allows(from, to)).ToEqual(want)
					}
				}
				ctx.Expect(count).ToEqual(121)
			})
		})

		s.Describe("returning to spec under the v2 profile", func(s *specs.Spec) {
			s.It("refuses the return from build, gate and review as illegal", func(ctx *specs.Context) {
				f := oddFacts()
				f.SpecRevision = true
				for _, from := range []string{Build, Gate, Review} {
					mustRefuse(ctx, V2, Position{State: from, Round: 1, RoundLimit: 3}, Spec, f, "not a legal transition")
				}
			})
		})

		s.Describe("returning to spec under the ODD profile", func(s *specs.Spec) {
			s.It("returns from build, gate and review with a new candidate only for a spec revision", func(ctx *specs.Context) {
				for _, from := range []string{Build, Gate, Review} {
					f := oddFacts()
					f.SpecRevision = true
					pos := Position{State: from, Round: 1, RoundLimit: 3, Candidate: 2}
					res := mustAdvance(ctx, ODD, pos, Spec, f)
					ctx.Expect(res.State).ToEqual(Spec)
					ctx.Expect(res.Round).ToEqual(1)
					ctx.Expect(res.Candidate).ToEqual(3)
					ctx.Expect(res.NextOperation).ToEqual("run_spec")
					ctx.Expect(res.Cause).To(specs.Contain("spec"))
					f.SpecRevision = false
					mustRefuse(ctx, ODD, pos, Spec, f, "SpecRevision")
				}
			})
		})

		s.Describe("counting build rounds across returns to spec", func(s *specs.Spec) {
			s.It("consumes a round per build and does not reset the limit on a return", func(ctx *specs.Context) {
				f := oddFacts()
				f.SpecRevision = true
				pos := Position{State: Spec, Round: 0, RoundLimit: 3}
				steps := []struct {
					to    string
					round int
				}{{Build, 1}, {Spec, 1}, {Build, 2}, {Spec, 2}, {Build, 3}}
				for _, st := range steps {
					res := mustAdvance(ctx, ODD, pos, st.to, f)
					ctx.Expect(res.State).ToEqual(st.to)
					ctx.Expect(res.Round).ToEqual(st.round)
					pos = Position{State: res.State, Round: res.Round, RoundLimit: pos.RoundLimit, Candidate: res.Candidate}
				}
				// The limit counts builds and the spec returns did not reset it.
				mustRefuse(ctx, ODD, Position{State: Spec, Round: 3, RoundLimit: 3}, Build, f, "needs-human", "3 of 3")
			})
		})

		s.Describe("starting a new build round after the gate or review", func(s *specs.Spec) {
			s.It("moves to the next round and candidate, and refuses a feedback round from needs-human", func(ctx *specs.Context) {
				f := oddFacts()
				f.Reviews = map[string]string{"code": "CHANGES"}
				res := mustAdvance(ctx, ODD, Position{State: Review, Round: 1, RoundLimit: 3, Candidate: 2}, Build, f)
				ctx.Expect(res.Round).ToEqual(2)
				ctx.Expect(res.Candidate).ToEqual(3)
				f.Gate = &GateResult{Verdict: "FAIL", Commit: "abc123"}
				res = mustAdvance(ctx, ODD, Position{State: Gate, Round: 2, RoundLimit: 3, Candidate: 3}, Build, f)
				ctx.Expect(res.Round).ToEqual(3)
				ctx.Expect(res.Candidate).ToEqual(4)
				mustRefuse(ctx, ODD, Position{State: NeedsHuman, Round: 1, RoundLimit: 3}, Build, f, "feedback rounds")
			})
		})

		s.Describe("reaching the round limit", func(s *specs.Spec) {
			s.It("sends build, gate and review to needs-human instead of back to spec", func(ctx *specs.Context) {
				f := oddFacts()
				f.SpecRevision = true
				for _, from := range []string{Build, Gate, Review} {
					pos := Position{State: from, Round: 3, RoundLimit: 3}
					mustRefuse(ctx, ODD, pos, Spec, f, "needs-human", "3 of 3")
					res := mustAdvance(ctx, ODD, pos, NeedsHuman, f)
					ctx.Expect(res.Round).ToEqual(3)
					ctx.Expect(res.Cause).To(specs.Contain("round 3 of 3"))
				}
				mustRefuse(ctx, ODD, Position{State: Spec, Round: 3, RoundLimit: 3}, Build, f, "needs-human")
			})

			s.It("keeps the v2 needs-human moves in the last round", func(ctx *specs.Context) {
				f := oddFacts()
				f.Gate = &GateResult{Verdict: "FAIL", Commit: "abc123"}
				pos := Position{State: Gate, Round: 3, RoundLimit: 3}
				mustRefuse(ctx, ODD, pos, Build, f, "needs-human", "3 of 3")
				res := mustAdvance(ctx, ODD, pos, NeedsHuman, f)
				ctx.Expect(res.Cause).ToEqual("gate FAIL in round 3 of 3")
				f = oddFacts()
				f.BuildStatus = "BLOCKED"
				mustAdvance(ctx, ODD, Position{State: Build, Round: 3, RoundLimit: 3}, NeedsHuman, f)
				f = oddFacts()
				mustRefuse(ctx, ODD, Position{State: Build, Round: 3, RoundLimit: 3}, NeedsHuman, f, "BLOCKED")
			})
		})

		s.Describe("numbering candidates as the job moves", func(s *specs.Spec) {
			s.It("starts a new candidate on each build or spec return under ODD and never under v2", func(ctx *specs.Context) {
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
					res := mustAdvance(ctx, ODD, Position{State: c.from, Round: 1, RoundLimit: 3, Candidate: 2}, c.to, f)
					ctx.Expect(res.Candidate).ToEqual(c.want)
				}
				res := mustAdvance(ctx, V2, Position{State: Spec, RoundLimit: 3, Candidate: 4}, Build, f)
				ctx.Expect(res.Candidate).ToEqual(4)
			})
		})

		s.Describe("refusing evidence from an earlier candidate", func(s *specs.Spec) {
			s.It("refuses a stale gate, review or decision and keeps the v2 checks", func(ctx *specs.Context) {
				review := Position{State: Review, Round: 1, RoundLimit: 3, Candidate: 2}
				approve := Position{State: Approve, Round: 1, RoundLimit: 3, Candidate: 2}
				for _, pos := range []Position{review, approve} {
					to := map[string]string{Review: Approve, Approve: Merging}[pos.State]

					f := oddFacts()
					f.Gate.Candidate = 1
					mustRefuse(ctx, ODD, pos, to, f, "gate", "candidate 1", "current candidate 2")

					f = oddFacts()
					f.ReviewCandidates = map[string]int{"code": 1}
					mustRefuse(ctx, ODD, pos, to, f, "review code", "candidate 1", "current candidate 2")

					f = oddFacts()
					f.ReviewCandidates = map[string]int{}
					mustRefuse(ctx, ODD, pos, to, f, "review code")

					mustAdvance(ctx, ODD, pos, to, oddFacts())
				}
				f := oddFacts()
				f.DecisionCandidate = 1
				mustRefuse(ctx, ODD, approve, Merging, f, "decision", "candidate 1", "current candidate 2")

				// The v2 checks still apply.
				f = oddFacts()
				f.Reviews = map[string]string{"code": "CHANGES"}
				mustRefuse(ctx, ODD, review, Approve, f, "CHANGES")
				f = oddFacts()
				f.Gate.Commit = "old"
				mustRefuse(ctx, ODD, approve, Merging, f, "HEAD")
			})
		})

		s.Describe("invalidating evidence when a new candidate is built", func(s *specs.Spec) {
			s.It("refuses the gate of the old candidate after a spec return and a rebuild", func(ctx *specs.Context) {
				f := oddFacts()
				f.SpecRevision = true
				res := mustAdvance(ctx, ODD, Position{State: Review, Round: 1, RoundLimit: 3, Candidate: 2}, Spec, f)
				res = mustAdvance(ctx, ODD, Position{State: res.State, Round: res.Round, RoundLimit: 3, Candidate: res.Candidate}, Build, f)
				// Evidence produced for candidate 2 is stale for the new build.
				mustRefuse(ctx, ODD, Position{State: Review, Round: res.Round, RoundLimit: 3, Candidate: res.Candidate}, Approve, f,
					"the gate is from candidate 2, not the current candidate 4")
			})
		})

		s.Describe("keeping the caller's facts and position unchanged", func(s *specs.Spec) {
			s.It("leaves the facts and the position as they were after moves", func(ctx *specs.Context) {
				f := oddFacts()
				f.SpecRevision = true
				before := oddFacts()
				before.SpecRevision = true
				pos := Position{State: Approve, Round: 1, RoundLimit: 3, Candidate: 2}
				posBefore := pos
				mustAdvance(ctx, ODD, pos, Merging, f)
				mustAdvance(ctx, ODD, Position{State: Review, Round: 1, RoundLimit: 3, Candidate: 2}, Spec, f)
				ctx.Expect(f).ToEqual(before)
				ctx.Expect(pos).ToEqual(posBefore)
			})
		})
	})
}
