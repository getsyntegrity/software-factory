package artifacts

import (
	"testing"

	"github.com/getsyntegrity/go-specs/specs"
)

// Fixture texts copied from the v2 job examples/jobs/001-add-a-subtract-function.
const (
	fixtureIntake = `DECISION: ACCEPT
RISK: low
SURFACES: none
RESCOPED: no

## Summary
- Accepted as a small, low-risk change with no UI surface
- Adds ` + "`subtract(a, b)`" + ` to ` + "`calc.py`" + ` after ` + "`add`" + `, returning ` + "`a - b`" + `
`
	fixtureBuild = `STATUS: READY

# Build: Add a subtract function

## Summary
- Round 1 committed ` + "`2934f5d`" + ` and opened draft PR #6
`
	fixtureChecks = `GATE: PASS
COMMIT: 2934f5dc28593f1f995d5e319d2b2cbd5f678718

## Summary
- checks.sh passed in 0s
- Protected paths: none changed
- Commit 2934f5d
`
	fixtureReview = `VERDICT: PASS

## Summary
- Passed with no findings
`
	fixtureDecision = `APPROVE

## Summary
- Approved: the change does exactly what issue #1 asks
`
)

func TestArtifacts(t *testing.T) {
	specs.Describe(t, "artifacts", func(s *specs.Spec) {
		s.Describe("fixture 001", func(s *specs.Spec) {
			s.It("parses intake.md", func(ctx *specs.Context) {
				intake, err := ParseIntake(fixtureIntake)
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(intake).ToEqual(Intake{Decision: "ACCEPT", Risk: "low", Surfaces: []string{}, Rescoped: false})
			})

			s.It("parses build.md", func(ctx *specs.Context) {
				status, err := ParseBuild(fixtureBuild)
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(status).ToEqual("READY")
			})

			s.It("parses round-1/checks.md", func(ctx *specs.Context) {
				checks, err := ParseChecks(fixtureChecks)
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(checks).ToEqual(Checks{Gate: "PASS", Commit: "2934f5dc28593f1f995d5e319d2b2cbd5f678718"})
			})

			s.It("parses round-1/review-code.md", func(ctx *specs.Context) {
				ctx.Expect(ParseReview(fixtureReview)).ToEqual("PASS")
			})

			s.It("parses decision.md", func(ctx *specs.Context) {
				decision, err := ParseDecision(fixtureDecision)
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(decision).ToEqual("APPROVE")
			})
		})

		s.Describe("Header", func(s *specs.Spec) {
			s.It("normalizes keys and values", func(ctx *specs.Context) {
				ctx.Expect(Header("  risk :  high \n")).ToEqual(map[string]string{"RISK": "high"})
			})

			s.It("splits on the first colon", func(ctx *specs.Context) {
				ctx.Expect(Header("URL: http://x\n")["URL"]).ToEqual("http://x")
			})

			s.It("ignores lines after the first blank line", func(ctx *specs.Context) {
				ctx.Expect(Header("GATE: PASS\n  \t\nCOMMIT: abc\n")).ToEqual(map[string]string{"GATE": "PASS"})
			})

			s.It("returns no fields for empty content", func(ctx *specs.Context) {
				ctx.Expect(Header("")).To(specs.BeEmpty())
			})
		})

		s.Describe("ParseIntake", func(s *specs.Spec) {
			s.It("splits SURFACES on commas", func(ctx *specs.Context) {
				intake, err := ParseIntake("DECISION: ACCEPT\nRISK: high\nSURFACES: ui, security\n")
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(intake.Surfaces).ToEqual([]string{"ui", "security"})
			})

			s.It("drops empty items and none from SURFACES", func(ctx *specs.Context) {
				intake, _ := ParseIntake("DECISION: ACCEPT\nRISK: high\nSURFACES: ui,, none ,\n")
				ctx.Expect(intake.Surfaces).ToEqual([]string{"ui"})
			})

			s.It("is rescoped only for yes", func(ctx *specs.Context) {
				yes, _ := ParseIntake("DECISION: ACCEPT\nRISK: low\nRESCOPED: yes\n")
				ctx.Expect(yes.Rescoped).To(specs.BeTrue())
				for _, value := range []string{"Yes", "YES", "no", "true", ""} {
					intake, _ := ParseIntake("DECISION: ACCEPT\nRISK: low\nRESCOPED: " + value + "\n")
					ctx.Expect(intake.Rescoped).To(specs.BeFalse())
				}
			})

			cases := []struct{ name, content, value string }{
				{"DECISION: MAYBE", "DECISION: MAYBE\nRISK: low\n", `"MAYBE"`},
				{"RISK: extreme", "DECISION: ACCEPT\nRISK: extreme\n", `"extreme"`},
				{"a missing DECISION", "RISK: low\n", `""`},
			}
			for _, tc := range cases {
				s.It("rejects "+tc.name, func(ctx *specs.Context) {
					_, err := ParseIntake(tc.content)
					ctx.Expect(err).To(specs.Not(specs.BeNil()))
					ctx.Expect(err.Error()).To(specs.Contain("intake.md"))
					ctx.Expect(err.Error()).To(specs.Contain(tc.value))
				})
			}
		})

		s.Describe("ParseBuild", func(s *specs.Spec) {
			s.It("returns BLOCKED", func(ctx *specs.Context) {
				status, err := ParseBuild("  STATUS: BLOCKED  \n\n## Blocked\n")
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(status).ToEqual("BLOCKED")
			})

			s.It("rejects STATUS: DONE", func(ctx *specs.Context) {
				_, err := ParseBuild("STATUS: DONE\n")
				ctx.Expect(err).To(specs.Not(specs.BeNil()))
				ctx.Expect(err.Error()).To(specs.Contain("build.md"))
				ctx.Expect(err.Error()).To(specs.Contain(`"STATUS: DONE"`))
			})
		})

		s.Describe("ParseChecks", func(s *specs.Spec) {
			s.It("returns FAIL and an empty commit", func(ctx *specs.Context) {
				checks, err := ParseChecks("GATE: FAIL\n")
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(checks).ToEqual(Checks{Gate: "FAIL"})
			})

			s.It("rejects GATE: OK", func(ctx *specs.Context) {
				_, err := ParseChecks("GATE: OK\nCOMMIT: abc\n")
				ctx.Expect(err).To(specs.Not(specs.BeNil()))
				ctx.Expect(err.Error()).To(specs.Contain("checks.md"))
				ctx.Expect(err.Error()).To(specs.Contain(`"OK"`))
			})
		})

		s.Describe("ParseReview", func(s *specs.Spec) {
			s.It("returns CHANGES with or without a space", func(ctx *specs.Context) {
				ctx.Expect(ParseReview("VERDICT: CHANGES\n\n## Findings\n")).ToEqual("CHANGES")
				ctx.Expect(ParseReview("VERDICT:PASS")).ToEqual("PASS")
			})

			s.It("returns invalid for an out-of-contract first line", func(ctx *specs.Context) {
				for _, content := range []string{"VERDICT: MAYBE\n", "verdict pass\n", "", "\nVERDICT: PASS\n"} {
					ctx.Expect(ParseReview(content)).ToEqual(Invalid)
				}
				ctx.Expect(Invalid).ToEqual("invalid")
			})
		})

		s.Describe("ParseDecision", func(s *specs.Spec) {
			s.It("returns ESCALATE", func(ctx *specs.Context) {
				decision, err := ParseDecision("ESCALATE\n\n## Why\n")
				ctx.Expect(err).To(specs.BeNil())
				ctx.Expect(decision).ToEqual("ESCALATE")
			})

			s.It("rejects MERGE", func(ctx *specs.Context) {
				_, err := ParseDecision("MERGE\n")
				ctx.Expect(err).To(specs.Not(specs.BeNil()))
				ctx.Expect(err.Error()).To(specs.Contain("decision.md"))
				ctx.Expect(err.Error()).To(specs.Contain(`"MERGE"`))
			})
		})
	})
}
