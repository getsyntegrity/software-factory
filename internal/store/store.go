package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getsyntegrity/software-factory/internal/jobs"
)

var (
	// ErrNotFound is wrapped by errors for a job with no file.
	ErrNotFound = errors.New("job not found")
	// ErrCorrupt is wrapped by errors for a job file that cannot be read as a valid job.
	ErrCorrupt = errors.New("corrupt job file")
)

// rename moves a written temp file onto its job file. Tests replace it to
// simulate a writer that crashes before the rename.
var rename = os.Rename

// Store keeps each job at <root>/<id>.json. Each job has a single writer
// across processes: Create and Update hold an exclusive per-job lock on
// <root>/.<id>.lock from before they read the job until after they write it.
// Readers take no lock; atomic writes never show them a partial job.
type Store struct {
	root string
}

// Open returns a store rooted at root, creating the directory if missing.
func Open(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("store: root is empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	return &Store{root: root}, nil
}

// Create returns the job for requestID. An existing job is returned as
// stored; otherwise a new job is written.
func (s *Store) Create(requestID string) (jobs.Job, error) {
	job, err := jobs.New(requestID)
	if err != nil {
		return jobs.Job{}, err
	}
	unlock, err := lockJob(s.root, job.ID)
	if err != nil {
		return jobs.Job{}, err
	}
	defer unlock()
	existing, err := s.read(job.ID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return jobs.Job{}, err
	}
	if err := s.write(job); err != nil {
		return jobs.Job{}, err
	}
	return job, nil
}

// Get returns the stored job with the given id.
func (s *Store) Get(id string) (jobs.Job, error) {
	if !jobs.ValidID(id) {
		return jobs.Job{}, fmt.Errorf("store: job %q: %w", id, ErrNotFound)
	}
	return s.read(id)
}

// Update replaces the job with the given id by change applied to it, holding
// the job's lock from the read until the write. The result is rejected, and
// nothing written, when change fails, the result is invalid, or it changes
// the job's ID or RequestID.
func (s *Store) Update(id string, change func(jobs.Job) (jobs.Job, error)) (jobs.Job, error) {
	if !jobs.ValidID(id) {
		return jobs.Job{}, fmt.Errorf("store: job %q: %w", id, ErrNotFound)
	}
	unlock, err := lockJob(s.root, id)
	if err != nil {
		return jobs.Job{}, err
	}
	defer unlock()
	current, err := s.read(id)
	if err != nil {
		return jobs.Job{}, err
	}
	next, err := change(current)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("store: update job %q: %w", id, err)
	}
	if err := jobs.Validate(next); err != nil {
		return jobs.Job{}, fmt.Errorf("store: update job %q: %w", id, err)
	}
	if next.ID != current.ID || next.RequestID != current.RequestID {
		return jobs.Job{}, fmt.Errorf("store: update job %q: id and request_id cannot change", id)
	}
	if err := s.write(next); err != nil {
		return jobs.Job{}, err
	}
	return next, nil
}

// List returns every stored job sorted by id. It fails as a whole when any
// job file is corrupt.
func (s *Store) List() ([]jobs.Job, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	list := []jobs.Job{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		job, err := s.load(filepath.Join(s.root, entry.Name()))
		if err != nil {
			return nil, err
		}
		list = append(list, job)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.root, id+".json")
}

func (s *Store) read(id string) (jobs.Job, error) {
	job, err := s.load(s.path(id))
	if errors.Is(err, fs.ErrNotExist) {
		return jobs.Job{}, fmt.Errorf("store: job %q: %w", id, ErrNotFound)
	}
	return job, err
}

func (s *Store) load(path string) (jobs.Job, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("store: %w", err)
	}
	var job jobs.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return jobs.Job{}, fmt.Errorf("store: %s: %w: %v", path, ErrCorrupt, err)
	}
	if err := jobs.Validate(job); err != nil {
		return jobs.Job{}, fmt.Errorf("store: %s: %w: %v", path, ErrCorrupt, err)
	}
	return job, nil
}

// write stores job atomically: a temp file in root is written, synced,
// closed and renamed onto <root>/<id>.json.
func (s *Store) write(job jobs.Job) (err error) {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	tmp, err := os.CreateTemp(s.root, "."+job.ID+".tmp-*")
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	if err = rename(tmp.Name(), s.path(job.ID)); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	return nil
}
