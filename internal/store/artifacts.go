package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/getsyntegrity/software-factory/internal/jobs"
)

// ErrArtifactNotFound is wrapped by errors for an artifact with no file.
var ErrArtifactNotFound = errors.New("artifact not found")

// Artifact kinds, named after their files in the v2 job layout.
const (
	KindIssue    = "issue"
	KindIntake   = "intake"
	KindSpec     = "spec"
	KindBuild    = "build"
	KindDecision = "decision"
	KindFeedback = "feedback"
	KindChecks   = "checks"
	KindReview   = "review"
)

var reviewerPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Artifact identifies one file in a job's folder. Round applies to checks
// and review, Number to feedback, and Reviewer to review.
type Artifact struct {
	Kind     string
	Round    int
	Number   int
	Reviewer string
}

// Path returns the artifact's path relative to the job folder <root>/<id>/,
// or an error when the artifact is not one of the layout's files.
func (a Artifact) Path() (string, error) {
	switch a.Kind {
	case KindIssue, KindIntake, KindSpec, KindBuild, KindDecision:
		return a.Kind + ".md", nil
	case KindFeedback:
		if a.Number < 1 {
			return "", fmt.Errorf("store: feedback number %d is below 1", a.Number)
		}
		return "feedback-" + strconv.Itoa(a.Number) + ".md", nil
	case KindChecks, KindReview:
		if a.Round < 1 {
			return "", fmt.Errorf("store: %s round %d is below 1", a.Kind, a.Round)
		}
		round := "round-" + strconv.Itoa(a.Round)
		if a.Kind == KindChecks {
			return filepath.Join(round, "checks.md"), nil
		}
		if !reviewerPattern.MatchString(a.Reviewer) {
			return "", fmt.Errorf("store: invalid reviewer name %q", a.Reviewer)
		}
		return filepath.Join(round, "review-"+a.Reviewer+".md"), nil
	}
	return "", fmt.Errorf("store: unknown artifact kind %q", a.Kind)
}

// WriteArtifact replaces the artifact's content for the job with the given
// id. It holds the job's lock, requires the job to exist, creates the job and
// round folders when missing, and writes atomically.
func (s *Store) WriteArtifact(id string, a Artifact, content []byte) error {
	rel, err := a.Path()
	if err != nil {
		return err
	}
	if !jobs.ValidID(id) {
		return fmt.Errorf("store: job %q: %w", id, ErrNotFound)
	}
	unlock, err := lockJob(s.root, id)
	if err != nil {
		return err
	}
	defer unlock()
	if _, err := s.read(id); err != nil {
		return err
	}
	path := filepath.Join(s.root, id, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	return writeAtomic(path, content)
}

// ReadArtifact returns the artifact's content for the job with the given id.
func (s *Store) ReadArtifact(id string, a Artifact) ([]byte, error) {
	rel, err := a.Path()
	if err != nil {
		return nil, err
	}
	if !jobs.ValidID(id) {
		return nil, fmt.Errorf("store: job %q: %w", id, ErrNotFound)
	}
	data, err := os.ReadFile(filepath.Join(s.root, id, rel))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("store: job %q artifact %s: %w", id, rel, ErrArtifactNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	return data, nil
}
