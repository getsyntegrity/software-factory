package store

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsyntegrity/software-factory/internal/jobs"
)

func open(t *testing.T, root string) *Store {
	t.Helper()
	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q): %v", root, err)
	}
	return s
}

func jsonFiles(t *testing.T, root string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestOpen(t *testing.T) {
	root := filepath.Join(t.TempDir(), "a", "b")
	open(t, root)
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Errorf("root not created: %v", err)
	}
	if _, err := Open(""); err == nil {
		t.Error("Open(\"\"): no error")
	}
}

func TestCreateIdempotent(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	first, err := s.Create("req-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	second, err := s.Create("req-1")
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	if first != second {
		t.Errorf("second Create = %+v, want %+v", second, first)
	}
	want, _ := jobs.New("req-1")
	if first != want {
		t.Errorf("Create = %+v, want %+v", first, want)
	}
	list, err := s.List()
	if err != nil || len(list) != 1 {
		t.Errorf("List = %v, %v, want one job", list, err)
	}
	if files := jsonFiles(t, root); len(files) != 1 || files[0] != filepath.Join(root, first.ID+".json") {
		t.Errorf("json files = %v, want only %s.json", files, first.ID)
	}
}

func TestRestartIdempotent(t *testing.T) {
	root := t.TempDir()
	original, err := open(t, root).Create("req-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	path := filepath.Join(root, original.ID+".json")
	before := readFile(t, path)

	s := open(t, root)
	got, err := s.Get(original.ID)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got != original || got.State != "intake" || got.NextOperation != "run_intake" {
		t.Errorf("Get after reopen = %+v, want %+v", got, original)
	}
	again, err := s.Create("req-1")
	if err != nil || again != original {
		t.Errorf("Create after reopen = %+v, %v, want %+v", again, err, original)
	}
	if list, err := s.List(); err != nil || len(list) != 1 {
		t.Errorf("List after reopen = %v, %v, want one job", list, err)
	}
	if after := readFile(t, path); !bytes.Equal(before, after) {
		t.Errorf("file rewritten:\n%s\nwant\n%s", after, before)
	}
}

func TestCorruptFile(t *testing.T) {
	cases := map[string]string{
		"broken syntax":          `{"id":`,
		"empty":                  ``,
		"truncated":              `{"id":"job-0123456789abcdef","request_id":"req-1","st`,
		"missing next_operation": `{"id":"x","request_id":"req-1","state":"intake"}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			s := open(t, root)
			want, _ := jobs.New("req-1")
			path := filepath.Join(root, want.ID+".json")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := s.Get(want.ID)
			if !errors.Is(err, ErrCorrupt) || errors.Is(err, ErrNotFound) {
				t.Errorf("Get = %v, want ErrCorrupt and not ErrNotFound", err)
			}
			if err == nil || !strings.Contains(err.Error(), path) {
				t.Errorf("Get error %v does not name %s", err, path)
			}
			if list, err := s.List(); !errors.Is(err, ErrCorrupt) || list != nil {
				t.Errorf("List = %v, %v, want nil and ErrCorrupt", list, err)
			}
			if _, err := s.Create("req-1"); !errors.Is(err, ErrCorrupt) {
				t.Errorf("Create = %v, want ErrCorrupt", err)
			}
			if got := readFile(t, path); string(got) != content {
				t.Errorf("file changed to %q, want %q", got, content)
			}
		})
	}
}

func TestGetNotFound(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	if err := os.WriteFile(filepath.Join(filepath.Dir(root), "x.json"), []byte(`{"id":`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"job-0123456789abcdef", "", "../x"} {
		_, err := s.Get(id)
		if !errors.Is(err, ErrNotFound) || errors.Is(err, ErrCorrupt) {
			t.Errorf("Get(%q) = %v, want ErrNotFound and not ErrCorrupt", id, err)
		}
	}
}

func TestCreateRejectsEmptyRequestID(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	for _, requestID := range []string{"", "   "} {
		if _, err := s.Create(requestID); err == nil {
			t.Errorf("Create(%q): no error", requestID)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("root holds %d entries, want none", len(entries))
	}
}

func TestAtomicWriteLeavesOnlyFinalFiles(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	for _, requestID := range []string{"req-b", "req-a", "req-c"} {
		if _, err := s.Create(requestID); err != nil {
			t.Fatalf("Create(%q): %v", requestID, err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		id, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok || !jobs.ValidID(id) {
			t.Errorf("leftover file %q", entry.Name())
		}
	}
	if len(entries) != 3 {
		t.Errorf("root holds %d entries, want 3", len(entries))
	}

	if err := os.WriteFile(filepath.Join(root, "job-x.json.tmp123"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("List has %d jobs, want 3", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].ID >= list[i].ID {
			t.Errorf("List not sorted by id: %q before %q", list[i-1].ID, list[i].ID)
		}
	}
}

func TestListEmpty(t *testing.T) {
	list, err := open(t, filepath.Join(t.TempDir(), "new")).List()
	if err != nil || len(list) != 0 {
		t.Errorf("List = %v, %v, want empty and no error", list, err)
	}
}

func TestConcurrentCreate(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	const n = 20
	ids := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job, err := s.Create("req-c")
			ids[i], errs[i] = job.ID, err
		}()
	}
	wg.Wait()
	for i := range n {
		if errs[i] != nil {
			t.Errorf("goroutine %d: %v", i, errs[i])
		}
		if ids[i] != ids[0] {
			t.Errorf("goroutine %d: id %q, want %q", i, ids[i], ids[0])
		}
	}
	if files := jsonFiles(t, root); len(files) != 1 {
		t.Errorf("json files = %v, want exactly one", files)
	}
}

// helperEnv selects what TestHelperProcess does when the test binary is
// re-executed as a child process.
const helperEnv = "STORE_HELPER"

// TestHelperProcess is not a real test: it runs the store operation named by
// STORE_HELPER in a child process started by helper.
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv(helperEnv)
	if mode == "" {
		return
	}
	s, err := Open(os.Getenv("STORE_ROOT"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	arg := os.Getenv("STORE_ARG")
	switch mode {
	case "update":
		for range 25 {
			if _, err := s.Update(arg, increment); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	case "create":
		job, err := s.Create(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(job.ID)
	case "lock":
		if _, err := lockJob(s.root, arg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("locked")
		time.Sleep(time.Minute)
	}
	os.Exit(0)
}

// helper returns a child process of the test binary running mode with arg
// against root.
func helper(root, mode, arg string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), helperEnv+"="+mode, "STORE_ROOT="+root, "STORE_ARG="+arg)
	cmd.Stderr = os.Stderr
	return cmd
}

// increment treats State as a counter and advances it by one, sleeping in
// between so that unserialized writers would lose updates.
func increment(job jobs.Job) (jobs.Job, error) {
	n, err := strconv.Atoi(job.State)
	if err != nil {
		return job, err
	}
	time.Sleep(time.Millisecond)
	job.State = strconv.Itoa(n + 1)
	return job, nil
}

func setState(state string) func(jobs.Job) (jobs.Job, error) {
	return func(job jobs.Job) (jobs.Job, error) {
		job.State = state
		return job, nil
	}
}

func TestUpdateSerializesAcrossProcesses(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job, err := s.Create("req-u")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := s.Update(job.ID, setState("0")); err != nil {
		t.Fatalf("Update: %v", err)
	}
	const children = 4
	cmds := make([]*exec.Cmd, children)
	for i := range cmds {
		cmds[i] = helper(root, "update", job.ID)
		if err := cmds[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Errorf("child %d: %v", i, err)
		}
	}
	got, err := s.Get(job.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if want := strconv.Itoa(children * 25); got.State != want {
		t.Errorf("counter = %s, want %s", got.State, want)
	}
}

func TestCreateAcrossProcesses(t *testing.T) {
	root := t.TempDir()
	const children = 6
	cmds := make([]*exec.Cmd, children)
	outs := make([]bytes.Buffer, children)
	for i := range cmds {
		cmds[i] = helper(root, "create", "req-p")
		cmds[i].Stdout = &outs[i]
		if err := cmds[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Errorf("child %d: %v", i, err)
		}
	}
	want := jobs.IDFor("req-p")
	for i := range outs {
		if got := strings.TrimSpace(outs[i].String()); got != want {
			t.Errorf("child %d: id %q, want %q", i, got, want)
		}
	}
	if files := jsonFiles(t, root); len(files) != 1 {
		t.Errorf("json files = %v, want exactly one", files)
	}
}

func TestUpdateRejectedWritesNothing(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job, err := s.Create("req-r")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	path := filepath.Join(root, job.ID+".json")
	before := readFile(t, path)
	changes := map[string]func(jobs.Job) (jobs.Job, error){
		"change error": func(j jobs.Job) (jobs.Job, error) { return j, errors.New("boom") },
		"empty state":  setState(""),
		"changed id": func(j jobs.Job) (jobs.Job, error) {
			j.ID = jobs.IDFor("other")
			return j, nil
		},
		"changed request id": func(j jobs.Job) (jobs.Job, error) {
			j.RequestID = "other"
			return j, nil
		},
	}
	for name, change := range changes {
		if _, err := s.Update(job.ID, change); err == nil {
			t.Errorf("%s: Update returned no error", name)
		}
		if after := readFile(t, path); !bytes.Equal(before, after) {
			t.Errorf("%s: file changed to %s, want %s", name, after, before)
		}
	}
	if files := jsonFiles(t, root); len(files) != 1 {
		t.Errorf("json files = %v, want exactly one", files)
	}
	for _, id := range []string{jobs.IDFor("missing"), "../x"} {
		if _, err := s.Update(id, setState("x")); !errors.Is(err, ErrNotFound) {
			t.Errorf("Update(%q) = %v, want ErrNotFound", id, err)
		}
	}
}

func TestUpdateInterruptedRenameKeepsPrevious(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job, err := s.Create("req-i")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	rename = func(string, string) error { return errors.New("crash before rename") }
	_, err = s.Update(job.ID, setState("next"))
	rename = os.Rename
	if err == nil {
		t.Fatal("Update with failing rename: no error")
	}
	if got, err := s.Get(job.ID); err != nil || got != job {
		t.Errorf("Get = %+v, %v, want %+v", got, err, job)
	}
	if tmps, _ := filepath.Glob(filepath.Join(root, "*.tmp-*")); len(tmps) != 0 {
		t.Errorf("leftover temp files %v", tmps)
	}
	if got, err := s.Update(job.ID, setState("next")); err != nil || got.State != "next" {
		t.Errorf("Update after restore = %+v, %v", got, err)
	}
}

func TestAbandonedTempFileIsHarmless(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job, err := s.Create("req-t")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	crash := filepath.Join(root, "."+job.ID+".tmp-crash")
	if err := os.WriteFile(crash, []byte(`{"id":"job-`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Get(job.ID); err != nil || got != job {
		t.Errorf("Get = %+v, %v, want %+v", got, err, job)
	}
	if list, err := s.List(); err != nil || len(list) != 1 || list[0] != job {
		t.Errorf("List = %v, %v, want only %+v", list, err, job)
	}
	if _, err := s.Update(job.ID, setState("next")); err != nil {
		t.Errorf("Update: %v", err)
	}
	if _, err := s.Create("req-t2"); err != nil {
		t.Errorf("Create: %v", err)
	}
	if list, err := s.List(); err != nil || len(list) != 2 {
		t.Errorf("List = %v, %v, want two jobs", list, err)
	}
}

// within fails the test when op does not return within a few seconds.
func within(t *testing.T, op func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- op() }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("operation failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("operation blocked")
	}
}

func TestKilledLockHolderDoesNotBlock(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	job, err := s.Create("req-k")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	cmd := helper(root, "lock", job.ID)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if line, err := bufio.NewReader(out).ReadString('\n'); err != nil || line != "locked\n" {
		t.Fatalf("child said %q, %v", line, err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	cmd.Wait()
	within(t, func() error {
		_, err := s.Update(job.ID, setState("after-kill"))
		return err
	})
	if list, err := s.List(); err != nil || len(list) != 1 {
		t.Errorf("List = %v, %v, want one job and no lock file", list, err)
	}
}

func TestDistinctJobsDoNotBlock(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	a, err := s.Create("req-a")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	b, err := s.Create("req-b")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	unlock, err := lockJob(root, a.ID)
	if err != nil {
		t.Fatalf("lockJob: %v", err)
	}
	defer unlock()
	within(t, func() error {
		_, err := s.Update(b.ID, setState("free"))
		return err
	})
}
