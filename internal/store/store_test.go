package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

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
