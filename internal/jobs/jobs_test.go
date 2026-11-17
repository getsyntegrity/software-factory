package jobs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/getsyntegrity/go-specs/specs"
)

func TestJobs(t *testing.T) {
	specs.Describe(t, "the job record", func(s *specs.Spec) {
		s.Describe("creating a job from a request ID", func(s *specs.Spec) {
			s.It("returns a job in state intake with the sha256-derived ID", func(ctx *specs.Context) {
				job, err := New("req-1")
				ctx.Expect(err).To(specs.BeNil())
				sum := sha256.Sum256([]byte("req-1"))
				wantID := "job-" + hex.EncodeToString(sum[:])[:16]
				want := Job{ID: wantID, RequestID: "req-1", State: StateIntake, NextOperation: NextRunIntake}
				ctx.Expect(job).ToEqual(want)
			})

			s.It("uses the intake state and run_intake operation", func(ctx *specs.Context) {
				ctx.Expect(StateIntake).ToEqual("intake")
				ctx.Expect(NextRunIntake).ToEqual("run_intake")
			})

			s.It("derives the same ID for the same request ID", func(ctx *specs.Context) {
				first, _ := New("req-1")
				again, _ := New("req-1")
				ctx.Expect(again.ID).ToEqual(first.ID)
			})

			s.It("derives an ID that ValidID accepts", func(ctx *specs.Context) {
				job, _ := New("req-1")
				ctx.Expect(ValidID(job.ID)).To(specs.BeTrue())
			})

			s.It("rejects empty request IDs", func(ctx *specs.Context) {
				for _, requestID := range []string{"", "   ", "\t\n"} {
					_, err := New(requestID)
					ctx.Expect(err).To(specs.Not(specs.BeNil()))
				}
			})
		})

		s.Describe("encoding a job as JSON", func(s *specs.Spec) {
			s.It("has exactly four non-empty string fields", func(ctx *specs.Context) {
				job, _ := New("req-1")
				data, err := json.Marshal(job)
				ctx.Expect(err).To(specs.BeNil())
				var fields map[string]any
				ctx.Expect(json.Unmarshal(data, &fields)).To(specs.BeNil())
				ctx.Expect(fields).To(specs.HaveLen(4))
				for _, name := range []string{"id", "request_id", "state", "next_operation"} {
					value, ok := fields[name].(string)
					ctx.Expect(ok).To(specs.BeTrue())
					ctx.Expect(value).To(specs.Not(specs.BeEmpty()))
				}
			})

			s.It("reads a job written before round, intake, gate and reviews", func(ctx *specs.Context) {
				var job Job
				data := `{"id":"job-0123456789abcdef","request_id":"r","state":"intake","next_operation":"run_intake"}`
				ctx.Expect(json.Unmarshal([]byte(data), &job)).To(specs.BeNil())
				want := Job{ID: "job-0123456789abcdef", RequestID: "r", State: StateIntake, NextOperation: NextRunIntake}
				ctx.Expect(job).ToEqual(want)
				ctx.Expect(Validate(job)).To(specs.BeNil())
			})

			s.It("writes the synced fields when set", func(ctx *specs.Context) {
				job, _ := New("req-1")
				job.Round = 1
				job.Intake = &Intake{Decision: "ACCEPT", Risk: "low", Surfaces: []string{}}
				job.Gate = "PASS"
				job.Reviews = map[string]string{"code": "PASS"}
				data, err := json.Marshal(job)
				ctx.Expect(err).To(specs.BeNil())
				var fields map[string]any
				ctx.Expect(json.Unmarshal(data, &fields)).To(specs.BeNil())
				ctx.Expect(fields["round"]).ToEqual(float64(1))
				ctx.Expect(fields["intake"]).ToEqual(map[string]any{"decision": "ACCEPT", "risk": "low", "surfaces": []any{}, "rescoped": false})
				ctx.Expect(fields["gate"]).ToEqual("PASS")
				ctx.Expect(fields["reviews"]).ToEqual(map[string]any{"code": "PASS"})
			})
		})

		s.Describe("validating a job", func(s *specs.Spec) {
			valid, _ := New("req-1")

			s.It("accepts a valid job", func(ctx *specs.Context) {
				ctx.Expect(Validate(valid)).To(specs.BeNil())
			})

			cases := []struct {
				field  string
				mutate func(*Job)
			}{
				{"id", func(j *Job) { j.ID = "" }},
				{"request_id", func(j *Job) { j.RequestID = " " }},
				{"state", func(j *Job) { j.State = "" }},
				{"next_operation", func(j *Job) { j.NextOperation = "" }},
			}
			for _, tc := range cases {
				s.It("names a missing "+tc.field, func(ctx *specs.Context) {
					job := valid
					tc.mutate(&job)
					err := Validate(job)
					ctx.Expect(err).To(specs.Not(specs.BeNil()))
					ctx.Expect(err.Error()).To(specs.Contain(tc.field))
				})
			}
		})

		s.Describe("checking the format of a job ID", func(s *specs.Spec) {
			s.It("rejects malformed IDs", func(ctx *specs.Context) {
				for _, id := range []string{"", "../x", "job-", "job-0123456789abcdeg", "job-0123456789ABCDEF", "job-0123456789abcdef0"} {
					ctx.Expect(ValidID(id)).To(specs.BeFalse())
				}
			})

			s.It("accepts a well-formed ID", func(ctx *specs.Context) {
				ctx.Expect(ValidID("job-0123456789abcdef")).To(specs.BeTrue())
			})
		})
	})
}
