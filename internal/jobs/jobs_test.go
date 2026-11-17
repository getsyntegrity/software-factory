package jobs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	job, err := New("req-1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sum := sha256.Sum256([]byte("req-1"))
	wantID := "job-" + hex.EncodeToString(sum[:])[:16]
	want := Job{ID: wantID, RequestID: "req-1", State: StateIntake, NextOperation: NextRunIntake}
	if job != want {
		t.Errorf("New = %+v, want %+v", job, want)
	}
	if StateIntake != "intake" || NextRunIntake != "run_intake" {
		t.Errorf("constants = %q, %q", StateIntake, NextRunIntake)
	}
	again, _ := New("req-1")
	if again.ID != job.ID {
		t.Errorf("ID not deterministic: %q then %q", job.ID, again.ID)
	}
	if !ValidID(job.ID) {
		t.Errorf("ValidID(%q) = false", job.ID)
	}
}

func TestNewRejectsEmptyRequestID(t *testing.T) {
	for _, requestID := range []string{"", "   ", "\t\n"} {
		if _, err := New(requestID); err == nil {
			t.Errorf("New(%q): no error", requestID)
		}
	}
}

func TestJSONFields(t *testing.T) {
	job, _ := New("req-1")
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 {
		t.Errorf("fields = %v, want exactly 4", fields)
	}
	for _, name := range []string{"id", "request_id", "state", "next_operation"} {
		if s, ok := fields[name].(string); !ok || s == "" {
			t.Errorf("field %q = %v, want a non-empty string", name, fields[name])
		}
	}
}

func TestValidate(t *testing.T) {
	valid, _ := New("req-1")
	if err := Validate(valid); err != nil {
		t.Errorf("Validate(valid) = %v", err)
	}
	cases := map[string]func(*Job){
		"id":             func(j *Job) { j.ID = "" },
		"request_id":     func(j *Job) { j.RequestID = " " },
		"state":          func(j *Job) { j.State = "" },
		"next_operation": func(j *Job) { j.NextOperation = "" },
	}
	for field, mutate := range cases {
		job := valid
		mutate(&job)
		err := Validate(job)
		if err == nil || !strings.Contains(err.Error(), field) {
			t.Errorf("Validate without %s = %v, want an error naming it", field, err)
		}
	}
}

func TestValidID(t *testing.T) {
	for _, id := range []string{"", "../x", "job-", "job-0123456789abcdeg", "job-0123456789ABCDEF", "job-0123456789abcdef0"} {
		if ValidID(id) {
			t.Errorf("ValidID(%q) = true", id)
		}
	}
	if !ValidID("job-0123456789abcdef") {
		t.Error("ValidID rejects a well-formed ID")
	}
}
