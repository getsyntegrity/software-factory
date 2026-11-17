package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/getsyntegrity/software-factory/internal/jobs"
)

// fixtureArtifacts names the files of the v2 fixture
// examples/jobs/001-add-a-subtract-function plus a feedback file.
var fixtureArtifacts = []struct {
	artifact Artifact
	path     string
}{
	{Artifact{Kind: KindIssue}, "issue.md"},
	{Artifact{Kind: KindIntake}, "intake.md"},
	{Artifact{Kind: KindSpec}, "spec.md"},
	{Artifact{Kind: KindBuild}, "build.md"},
	{Artifact{Kind: KindDecision}, "decision.md"},
	{Artifact{Kind: KindChecks, Round: 1}, "round-1/checks.md"},
	{Artifact{Kind: KindReview, Round: 1, Reviewer: "code"}, "round-1/review-code.md"},
	{Artifact{Kind: KindFeedback, Number: 1}, "feedback-1.md"},
}

func createJob(t *testing.T, s *Store, requestID string) jobs.Job {
	t.Helper()
	job, err := s.Create(requestID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return job
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s exists (stat err %v), want nothing created", path, err)
	}
}

func TestArtifactWriteReadEveryKind(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job := createJob(t, s, "req-art")
	for _, f := range fixtureArtifacts {
		path, err := f.artifact.Path()
		if err != nil || path != filepath.FromSlash(f.path) {
			t.Errorf("Path(%+v) = %q, %v, want %q", f.artifact, path, err, f.path)
		}
		content := []byte("content of " + f.path + "\n")
		if err := s.WriteArtifact(job.ID, f.artifact, content); err != nil {
			t.Fatalf("WriteArtifact(%s): %v", f.path, err)
		}
		got, err := s.ReadArtifact(job.ID, f.artifact)
		if err != nil || string(got) != string(content) {
			t.Errorf("ReadArtifact(%s) = %q, %v, want %q", f.path, got, err, content)
		}
		disk := readFile(t, filepath.Join(root, job.ID, filepath.FromSlash(f.path)))
		if string(disk) != string(content) {
			t.Errorf("%s on disk = %q, want %q", f.path, disk, content)
		}
	}
}

func TestArtifactRewriteReplacesContent(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job := createJob(t, s, "req-rw")
	a := Artifact{Kind: KindReview, Round: 2, Reviewer: "security"}
	if err := s.WriteArtifact(job.ID, a, []byte("VERDICT: CHANGES\nlong first version\n")); err != nil {
		t.Fatalf("WriteArtifact: %v", err)
	}
	if err := s.WriteArtifact(job.ID, a, []byte("VERDICT: PASS\n")); err != nil {
		t.Fatalf("second WriteArtifact: %v", err)
	}
	if got, err := s.ReadArtifact(job.ID, a); err != nil || string(got) != "VERDICT: PASS\n" {
		t.Errorf("ReadArtifact = %q, %v, want rewritten content", got, err)
	}
}

func TestArtifactInterruptedRenameKeepsPrevious(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job := createJob(t, s, "req-ai")
	for _, a := range []Artifact{{Kind: KindSpec}, {Kind: KindChecks, Round: 1}} {
		if err := s.WriteArtifact(job.ID, a, []byte("first")); err != nil {
			t.Fatalf("WriteArtifact: %v", err)
		}
		rename = func(string, string) error { return errors.New("crash before rename") }
		err := s.WriteArtifact(job.ID, a, []byte("second"))
		rename = os.Rename
		if err == nil {
			t.Fatal("WriteArtifact with failing rename: no error")
		}
		if got, err := s.ReadArtifact(job.ID, a); err != nil || string(got) != "first" {
			t.Errorf("ReadArtifact(%+v) = %q, %v, want previous content", a, got, err)
		}
		rel, _ := a.Path()
		dir := filepath.Dir(filepath.Join(root, job.ID, rel))
		if tmps, _ := filepath.Glob(filepath.Join(dir, "*.tmp-*")); len(tmps) != 0 {
			t.Errorf("leftover temp files %v", tmps)
		}
	}
}

func TestArtifactAbandonedTempFileIsHarmless(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job := createJob(t, s, "req-at")
	spec := Artifact{Kind: KindSpec}
	checks := Artifact{Kind: KindChecks, Round: 1}
	for _, a := range []Artifact{spec, checks} {
		if err := s.WriteArtifact(job.ID, a, []byte("valid")); err != nil {
			t.Fatalf("WriteArtifact: %v", err)
		}
	}
	for _, crash := range []string{
		filepath.Join(root, job.ID, ".spec.md.tmp-crash"),
		filepath.Join(root, job.ID, "round-1", ".checks.md.tmp-crash"),
		filepath.Join(root, job.ID, "."+job.ID+".tmp-crash.json"),
	} {
		if err := os.WriteFile(crash, []byte("half"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []Artifact{spec, checks} {
		if got, err := s.ReadArtifact(job.ID, a); err != nil || string(got) != "valid" {
			t.Errorf("ReadArtifact(%+v) = %q, %v, want valid", a, got, err)
		}
	}
	if got, err := s.Get(job.ID); err != nil || got != job {
		t.Errorf("Get = %+v, %v, want %+v", got, err, job)
	}
	if list, err := s.List(); err != nil || len(list) != 1 || list[0] != job {
		t.Errorf("List = %v, %v, want only %+v", list, err, job)
	}
}

func TestArtifactInvalidIdentifierWritesNothing(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job := createJob(t, s, "req-inv")
	invalid := []Artifact{
		{Kind: "notes"},
		{Kind: ""},
		{Kind: KindChecks, Round: 0},
		{Kind: KindChecks, Round: -1},
		{Kind: KindReview, Round: 0, Reviewer: "code"},
		{Kind: KindReview, Round: 1, Reviewer: "../x"},
		{Kind: KindReview, Round: 1, Reviewer: "a/b"},
		{Kind: KindReview, Round: 1, Reviewer: ""},
		{Kind: KindReview, Round: 1, Reviewer: "Code"},
		{Kind: KindFeedback, Number: 0},
		{Kind: KindFeedback, Number: -2},
	}
	for _, a := range invalid {
		if _, err := a.Path(); err == nil {
			t.Errorf("Path(%+v): no error", a)
		}
		if err := s.WriteArtifact(job.ID, a, []byte("x")); err == nil {
			t.Errorf("WriteArtifact(%+v): no error", a)
		}
		if _, err := s.ReadArtifact(job.ID, a); err == nil {
			t.Errorf("ReadArtifact(%+v): no error", a)
		}
		assertMissing(t, filepath.Join(root, job.ID))
	}
	if entries, _ := os.ReadDir(root); len(entries) != 1 {
		t.Errorf("root holds %d entries, want only the job file", len(entries))
	}
}

func TestArtifactMissingJobWritesNothing(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	for _, id := range []string{jobs.IDFor("missing"), "../x", "job-"} {
		for _, a := range []Artifact{{Kind: KindIssue}, {Kind: KindChecks, Round: 1}} {
			err := s.WriteArtifact(id, a, []byte("x"))
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("WriteArtifact(%q, %+v) = %v, want ErrNotFound", id, a, err)
			}
		}
	}
	assertMissing(t, filepath.Join(root, jobs.IDFor("missing")))
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Errorf("root holds %v, want nothing created", entries)
	}
}

func TestArtifactCorruptJobRejectsWrite(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	id := jobs.IDFor("req-c")
	if err := os.WriteFile(filepath.Join(root, id+".json"), []byte(`{"id":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteArtifact(id, Artifact{Kind: KindSpec}, []byte("x")); !errors.Is(err, ErrCorrupt) {
		t.Errorf("WriteArtifact = %v, want ErrCorrupt", err)
	}
	assertMissing(t, filepath.Join(root, id))
}

func TestArtifactMissingIsNotMissingJob(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job := createJob(t, s, "req-m")
	for _, a := range []Artifact{{Kind: KindBuild}, {Kind: KindReview, Round: 3, Reviewer: "ux"}} {
		_, err := s.ReadArtifact(job.ID, a)
		if !errors.Is(err, ErrArtifactNotFound) {
			t.Errorf("ReadArtifact(%+v) = %v, want ErrArtifactNotFound", a, err)
		}
		if errors.Is(err, ErrNotFound) {
			t.Errorf("ReadArtifact(%+v) = %v, must not be ErrNotFound", a, err)
		}
	}
}

func TestArtifactFoldersKeepJobOperations(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	a := createJob(t, s, "req-1")
	b := createJob(t, s, "req-2")
	for _, f := range fixtureArtifacts {
		if err := s.WriteArtifact(a.ID, f.artifact, []byte("x")); err != nil {
			t.Fatalf("WriteArtifact: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, a.ID, "round-1", ".review-code.md.tmp-crash"), []byte("half"), 0o644); err != nil {
		t.Fatal(err)
	}
	if again, err := s.Create("req-1"); err != nil || again != a {
		t.Errorf("Create = %+v, %v, want %+v", again, err, a)
	}
	if got, err := s.Get(a.ID); err != nil || got != a {
		t.Errorf("Get = %+v, %v, want %+v", got, err, a)
	}
	next, err := s.Update(a.ID, setState("next"))
	if err != nil || next.State != "next" {
		t.Fatalf("Update = %+v, %v", next, err)
	}
	if list, err := s.List(); err != nil || len(list) != 2 || list[0].ID > list[1].ID ||
		(list[0] != next && list[1] != next) || (list[0] != b && list[1] != b) {
		t.Errorf("List = %v, %v, want exactly %+v and %+v", list, err, next, b)
	}
	if files := jsonFiles(t, root); len(files) != 2 {
		t.Errorf("json files = %v, want the two jobs at <root>/<id>.json", files)
	}
	if got, err := s.ReadArtifact(a.ID, Artifact{Kind: KindIssue}); err != nil || string(got) != "x" {
		t.Errorf("ReadArtifact after Update = %q, %v", got, err)
	}
}
