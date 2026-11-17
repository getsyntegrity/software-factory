package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getsyntegrity/software-factory/internal/artifacts"
	"github.com/getsyntegrity/software-factory/internal/jobs"
	"github.com/getsyntegrity/software-factory/internal/state"
)

// artifactSource is what syncJob reads a job's artifacts from. *Store
// satisfies it.
type artifactSource interface {
	ReadArtifact(id string, a Artifact) ([]byte, error)
	Reviewers(id string, round int) ([]string, error)
}

// reviewStates are the states whose reviews are synced from the round.
var reviewStates = map[string]bool{
	state.Review:     true,
	state.Approve:    true,
	state.Merging:    true,
	state.Merged:     true,
	state.NeedsHuman: true,
}

// Sync rereads the artifacts of the job with the given id and stores the
// intake, gate and reviews derived from them, through Update: under the
// job's lock, and nothing is written when the derivation fails.
func (s *Store) Sync(id string) (jobs.Job, error) {
	return s.Update(id, func(job jobs.Job) (jobs.Job, error) {
		return syncJob(job, s)
	})
}

// Reviewers returns the sorted reviewer names that have a review file in the
// job's round, or an empty list when the round folder does not exist.
func (s *Store) Reviewers(id string, round int) ([]string, error) {
	checks, err := Artifact{Kind: KindChecks, Round: round}.Path()
	if err != nil {
		return nil, err
	}
	if !jobs.ValidID(id) {
		return nil, fmt.Errorf("store: job %q: %w", id, ErrNotFound)
	}
	entries, err := os.ReadDir(filepath.Join(s.root, id, filepath.Dir(checks)))
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	reviewers := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "review-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		reviewer := strings.TrimSuffix(strings.TrimPrefix(name, "review-"), ".md")
		if reviewerPattern.MatchString(reviewer) {
			reviewers = append(reviewers, reviewer)
		}
	}
	sort.Strings(reviewers)
	return reviewers, nil
}

// syncJob returns job with intake, gate and reviews derived from its
// artifacts in src, as v2 `job sync` does. A missing intake.md keeps intake;
// round 0 keeps gate and reviews; a missing checks.md empties gate; reviews
// are replaced only in the review states.
func syncJob(job jobs.Job, src artifactSource) (jobs.Job, error) {
	content, err := src.ReadArtifact(job.ID, Artifact{Kind: KindIntake})
	switch {
	case err == nil:
		intake, err := artifacts.ParseIntake(string(content))
		if err != nil {
			return jobs.Job{}, err
		}
		job.Intake = &jobs.Intake{Decision: intake.Decision, Risk: intake.Risk, Surfaces: intake.Surfaces, Rescoped: intake.Rescoped}
	case !errors.Is(err, ErrArtifactNotFound):
		return jobs.Job{}, err
	}
	if job.Round < 1 {
		return job, nil
	}
	content, err = src.ReadArtifact(job.ID, Artifact{Kind: KindChecks, Round: job.Round})
	switch {
	case err == nil:
		checks, err := artifacts.ParseChecks(string(content))
		if err != nil {
			return jobs.Job{}, err
		}
		job.Gate = checks.Gate
	case errors.Is(err, ErrArtifactNotFound):
		job.Gate = ""
	default:
		return jobs.Job{}, err
	}
	if !reviewStates[job.State] {
		return job, nil
	}
	reviewers, err := src.Reviewers(job.ID, job.Round)
	if err != nil {
		return jobs.Job{}, err
	}
	reviews := map[string]string{}
	for _, reviewer := range reviewers {
		content, err := src.ReadArtifact(job.ID, Artifact{Kind: KindReview, Round: job.Round, Reviewer: reviewer})
		if errors.Is(err, ErrArtifactNotFound) {
			continue
		}
		if err != nil {
			return jobs.Job{}, err
		}
		reviews[reviewer] = artifacts.ParseReview(string(content))
	}
	job.Reviews = reviews
	return job, nil
}
