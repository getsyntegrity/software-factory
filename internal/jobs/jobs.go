package jobs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// The state and next operation of a new job.
const (
	StateIntake   = "intake"
	NextRunIntake = "run_intake"
)

// Job is one factory job as stored on disk.
type Job struct {
	ID            string `json:"id"`
	RequestID     string `json:"request_id"`
	State         string `json:"state"`
	NextOperation string `json:"next_operation"`
}

var idPattern = regexp.MustCompile(`^job-[0-9a-f]{16}$`)

// New returns a new job for requestID, in state intake.
func New(requestID string) (Job, error) {
	if strings.TrimSpace(requestID) == "" {
		return Job{}, errors.New("jobs: request_id is empty")
	}
	return Job{
		ID:            IDFor(requestID),
		RequestID:     requestID,
		State:         StateIntake,
		NextOperation: NextRunIntake,
	}, nil
}

// IDFor returns the job ID for requestID: "job-" and the first 16 hex
// characters of its SHA-256.
func IDFor(requestID string) string {
	sum := sha256.Sum256([]byte(requestID))
	return "job-" + hex.EncodeToString(sum[:])[:16]
}

// ValidID reports whether id has the format IDFor produces.
func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

// Validate returns an error naming the first required field of job that is
// missing or empty.
func Validate(job Job) error {
	fields := []struct{ name, value string }{
		{"id", job.ID},
		{"request_id", job.RequestID},
		{"state", job.State},
		{"next_operation", job.NextOperation},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("jobs: field %q is missing or empty", field.name)
		}
	}
	return nil
}
