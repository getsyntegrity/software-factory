package store

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/getsyntegrity/go-specs/mock"
	"github.com/getsyntegrity/go-specs/specs"
	"github.com/getsyntegrity/software-factory/internal/jobs"
	"github.com/getsyntegrity/software-factory/internal/state"
)

// spySource is an artifactSource over a mock.Controller.
type spySource struct {
	ctrl *mock.Controller
}

func (s spySource) ReadArtifact(id string, a Artifact) ([]byte, error) {
	r := s.ctrl.Method("ReadArtifact").Call(id, a)
	return mock.Value[[]byte](r, 0), r.Err(1)
}

func (s spySource) Reviewers(id string, round int) ([]string, error) {
	r := s.ctrl.Method("Reviewers").Call(id, round)
	return mock.Value[[]string](r, 0), r.Err(1)
}

// newSpySource returns a spySource serving files, keyed by their path in the
// job folder, for any number of calls.
func newSpySource(ctx *specs.Context, files map[string]string) spySource {
	ctrl := mock.NewController(ctx)
	ctrl.Method("ReadArtifact").Expect(mock.Any(), mock.Any()).AnyTimes().Do(func(args []any) []any {
		rel, err := args[1].(Artifact).Path()
		if err != nil {
			return []any{nil, err}
		}
		content, ok := files[rel]
		if !ok {
			return []any{nil, fmt.Errorf("spy: %s: %w", rel, ErrArtifactNotFound)}
		}
		return []any{[]byte(content), nil}
	})
	ctrl.Method("Reviewers").Expect(mock.Any(), mock.Any()).AnyTimes().Do(func(args []any) []any {
		prefix := fmt.Sprintf("round-%d/review-", args[1].(int))
		reviewers := []string{}
		for rel := range files {
			if strings.HasPrefix(rel, prefix) {
				reviewers = append(reviewers, strings.TrimSuffix(strings.TrimPrefix(rel, prefix), ".md"))
			}
		}
		sort.Strings(reviewers)
		return []any{reviewers, nil}
	})
	return spySource{ctrl: ctrl}
}

func syncFixture() map[string]string {
	return map[string]string{
		"intake.md":              "DECISION: ACCEPT\nRISK: low\nSURFACES: none\nRESCOPED: no\n\n## Summary\n",
		"round-1/checks.md":      "GATE: PASS\nCOMMIT: 2934f5dc28593f1f995d5e319d2b2cbd5f678718\n\n## Summary\n",
		"round-1/review-code.md": "VERDICT: PASS\n\n## Summary\n",
	}
}

func reviewJob() jobs.Job {
	return jobs.Job{ID: "job-0123456789abcdef", RequestID: "r", State: state.Review, NextOperation: "run_review", Round: 1}
}

// sameIdentity reports whether a and b agree on every field sync never changes.
func sameIdentity(a, b jobs.Job) bool {
	return a.ID == b.ID && a.RequestID == b.RequestID && a.State == b.State &&
		a.NextOperation == b.NextOperation && a.Round == b.Round
}

func TestSync(t *testing.T) {
	specs.Describe(t, "syncing a job with its folder's artifacts", func(s *specs.Spec) {
		s.Describe("deriving fields from the round's files", func(s *specs.Spec) {
			s.It("derives intake, gate and reviews from the round's files", func(ctx *specs.Context) {
				job := reviewJob()
				got, err := syncJob(job, newSpySource(ctx, syncFixture()))
				ctx.Expect(err).To(specs.BeNil())
				want := job
				want.Intake = &jobs.Intake{Decision: "ACCEPT", Risk: "low", Surfaces: []string{}, Rescoped: false}
				want.Gate = "PASS"
				want.Reviews = map[string]string{"code": "PASS"}
				ctx.Expect(got).ToEqual(want)
			})

			s.It("returns an identical job when run again on its output", func(ctx *specs.Context) {
				src := newSpySource(ctx, syncFixture())
				first, err := syncJob(reviewJob(), src)
				ctx.Expect(err).To(specs.BeNil())
				again, err := syncJob(first, src)
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(again).ToEqual(first)
			})
		})

		s.Describe("leaving the fields whose files did not change alone", func(s *specs.Spec) {
			s.It("changes only reviews when only review-code.md changes", func(ctx *specs.Context) {
				before, _ := syncJob(reviewJob(), newSpySource(ctx, syncFixture()))
				files := syncFixture()
				files["round-1/review-code.md"] = "VERDICT: CHANGES\n\n## Findings\n"
				after, err := syncJob(before, newSpySource(ctx, files))
				ctx.Expect(err).To(specs.BeNil())
				want := before
				want.Reviews = map[string]string{"code": "CHANGES"}
				ctx.Expect(after).ToEqual(want)
				ctx.Expect(sameIdentity(after, reviewJob())).To(specs.BeTrue())
			})

			s.It("changes only gate when only checks.md changes", func(ctx *specs.Context) {
				before, _ := syncJob(reviewJob(), newSpySource(ctx, syncFixture()))
				files := syncFixture()
				files["round-1/checks.md"] = "GATE: FAIL\nCOMMIT: 2934f5dc28593f1f995d5e319d2b2cbd5f678718\n"
				after, err := syncJob(before, newSpySource(ctx, files))
				ctx.Expect(err).To(specs.BeNil())
				want := before
				want.Gate = "FAIL"
				ctx.Expect(after).ToEqual(want)
				ctx.Expect(sameIdentity(after, reviewJob())).To(specs.BeTrue())
			})

			s.It("keeps an existing intake when intake.md is missing", func(ctx *specs.Context) {
				job := reviewJob()
				job.Intake = &jobs.Intake{Decision: "ACCEPT", Risk: "high", Surfaces: []string{"ui"}, Rescoped: true}
				files := syncFixture()
				delete(files, "intake.md")
				got, err := syncJob(job, newSpySource(ctx, files))
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(got.Intake).ToEqual(&jobs.Intake{Decision: "ACCEPT", Risk: "high", Surfaces: []string{"ui"}, Rescoped: true})
			})
		})

		s.Describe("applying the round 0 and round 1 rules", func(s *specs.Spec) {
			s.It("keeps gate and reviews in round 0 and does not read checks.md", func(ctx *specs.Context) {
				job := reviewJob()
				job.Round = 0
				job.Gate = "FAIL"
				job.Reviews = map[string]string{"code": "CHANGES"}
				src := newSpySource(ctx, syncFixture())
				got, err := syncJob(job, src)
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(got.Gate).ToEqual("FAIL")
				ctx.Expect(got.Reviews).ToEqual(map[string]string{"code": "CHANGES"})
				ctx.Expect(sameIdentity(got, job)).To(specs.BeTrue())
				for _, call := range src.ctrl.Method("ReadArtifact").Calls() {
					ctx.Expect(call.Args[1].(Artifact).Kind).To(specs.Not(specs.Equal(KindChecks)))
				}
				ctx.Expect(src.ctrl.Method("Reviewers").Calls()).To(specs.BeEmpty())
			})

			s.It("empties gate in round 1 with no checks.md", func(ctx *specs.Context) {
				job := reviewJob()
				job.Gate = "PASS"
				files := syncFixture()
				delete(files, "round-1/checks.md")
				got, err := syncJob(job, newSpySource(ctx, files))
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(got.Gate).ToEqual("")
			})
		})

		s.Describe("applying the review-state rules", func(s *specs.Spec) {
			s.It("keeps reviews outside the review states", func(ctx *specs.Context) {
				job := reviewJob()
				job.State = state.Gate
				job.NextOperation = "run_gate"
				job.Reviews = map[string]string{"code": "CHANGES"}
				src := newSpySource(ctx, syncFixture())
				got, err := syncJob(job, src)
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(got.Reviews).ToEqual(map[string]string{"code": "CHANGES"})
				ctx.Expect(got.Gate).ToEqual("PASS")
				ctx.Expect(src.ctrl.Method("Reviewers").Calls()).To(specs.BeEmpty())
			})

			s.It("syncs reviews in every review state", func(ctx *specs.Context) {
				for _, st := range []string{state.Review, state.Approve, state.Merging, state.Merged, state.NeedsHuman} {
					job := reviewJob()
					job.State = st
					got, err := syncJob(job, newSpySource(ctx, syncFixture()))
					ctx.Expect(err).To(specs.BeNil())
					ctx.Expect(got.Reviews).ToEqual(map[string]string{"code": "PASS"})
				}
			})

			s.It("gives invalid for a review with an out-of-contract first line", func(ctx *specs.Context) {
				files := syncFixture()
				files["round-1/review-security.md"] = "verdict pass\n"
				got, err := syncJob(reviewJob(), newSpySource(ctx, files))
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(got.Reviews).ToEqual(map[string]string{"code": "PASS", "security": "invalid"})
			})
		})

		s.Describe("rejecting out-of-contract or unreadable files", func(s *specs.Spec) {
			s.It("fails on an out-of-contract intake header", func(ctx *specs.Context) {
				files := syncFixture()
				files["intake.md"] = "DECISION: MAYBE\nRISK: low\n"
				_, err := syncJob(reviewJob(), newSpySource(ctx, files))
				ctx.Expect(err).To(specs.Not(specs.BeNil()))
				ctx.Expect(err.Error()).To(specs.Contain("intake.md"))
			})

			s.It("fails on an out-of-contract checks header", func(ctx *specs.Context) {
				files := syncFixture()
				files["round-1/checks.md"] = "GATE: OK\n"
				_, err := syncJob(reviewJob(), newSpySource(ctx, files))
				ctx.Expect(err).To(specs.Not(specs.BeNil()))
				ctx.Expect(err.Error()).To(specs.Contain("checks.md"))
			})

			s.It("fails on a read error other than not found", func(ctx *specs.Context) {
				boom := errors.New("disk on fire")
				ctrl := mock.NewController(ctx)
				ctrl.Method("ReadArtifact").Expect(mock.Any(), Artifact{Kind: KindIntake}).Return(nil, boom)
				_, err := syncJob(reviewJob(), spySource{ctrl: ctrl})
				ctx.Expect(errors.Is(err, boom)).To(specs.BeTrue())
			})
		})
	})
}
