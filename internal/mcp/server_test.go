package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/getsyntegrity/software-factory/internal/jobs"
	"github.com/getsyntegrity/software-factory/internal/store"
)

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type callResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

// serve runs the server on the given lines and returns one response per
// output line.
func serve(t *testing.T, root string, lines ...string) []rpcResponse {
	t.Helper()
	s, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out, s, "test"); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var responses []rpcResponse
	for _, line := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var r rpcResponse
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("output line %q is not JSON: %v", line, err)
		}
		if r.JSONRPC != "2.0" {
			t.Fatalf("output line %q: jsonrpc %q, want 2.0", line, r.JSONRPC)
		}
		responses = append(responses, r)
	}
	return responses
}

func call(id int, tool, args string) string {
	return `{"jsonrpc":"2.0","id":` + strconv.Itoa(id) + `,"method":"tools/call","params":{"name":"` + tool + `","arguments":` + args + `}}`
}

func toolResult(t *testing.T, r rpcResponse) callResult {
	t.Helper()
	if r.Error != nil {
		t.Fatalf("id %s: error %+v, want a result", r.ID, *r.Error)
	}
	var res callResult
	if err := json.Unmarshal(r.Result, &res); err != nil {
		t.Fatalf("id %s: result %s: %v", r.ID, r.Result, err)
	}
	if len(res.Content) != 1 || res.Content[0].Type != "text" {
		t.Fatalf("id %s: content %+v, want one text item", r.ID, res.Content)
	}
	return res
}

func okJob(t *testing.T, r rpcResponse) jobs.Job {
	t.Helper()
	res := toolResult(t, r)
	if res.IsError {
		t.Fatalf("id %s: isError true: %s", r.ID, res.Content[0].Text)
	}
	var job jobs.Job
	if err := json.Unmarshal([]byte(res.Content[0].Text), &job); err != nil {
		t.Fatalf("id %s: text %q: %v", r.ID, res.Content[0].Text, err)
	}
	return job
}

const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`

func TestInitialize(t *testing.T) {
	tests := []struct{ requested, want string }{
		{"2025-06-18", "2025-06-18"},
		{"2024-11-05", "2024-11-05"},
		{"2025-03-26", "2025-03-26"},
		{"2025-11-25", "2025-11-25"},
		{"1999-01-01", "2025-11-25"},
	}
	for _, tt := range tests {
		line := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` + tt.requested + `"}}`
		rs := serve(t, t.TempDir(), line)
		if len(rs) != 1 || string(rs[0].ID) != "1" {
			t.Fatalf("%s: responses %+v", tt.requested, rs)
		}
		var res struct {
			ProtocolVersion string `json:"protocolVersion"`
			Capabilities    struct {
				Tools *struct{} `json:"tools"`
			} `json:"capabilities"`
			ServerInfo struct{ Name, Version string } `json:"serverInfo"`
		}
		if err := json.Unmarshal(rs[0].Result, &res); err != nil {
			t.Fatal(err)
		}
		if res.ProtocolVersion != tt.want || res.Capabilities.Tools == nil ||
			res.ServerInfo.Name != "aifactory" || res.ServerInfo.Version != "test" {
			t.Errorf("%s: result %s", tt.requested, rs[0].Result)
		}
	}
}

func TestNotificationHasNoResponse(t *testing.T) {
	rs := serve(t, t.TempDir(), `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if len(rs) != 0 {
		t.Errorf("responses %+v, want none", rs)
	}
}

func TestPing(t *testing.T) {
	rs := serve(t, t.TempDir(), `{"jsonrpc":"2.0","id":"a","method":"ping"}`)
	if len(rs) != 1 || string(rs[0].ID) != `"a"` || string(rs[0].Result) != "{}" {
		t.Errorf("responses %+v, want one empty result", rs)
	}
}

func TestToolsList(t *testing.T) {
	rs := serve(t, t.TempDir(), initialize, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if len(rs) != 2 {
		t.Fatalf("got %d responses, want 2", len(rs))
	}
	var res struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema struct {
				Type                 string `json:"type"`
				AdditionalProperties *bool  `json:"additionalProperties"`
			} `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(rs[1].Result, &res); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		s := tool.InputSchema
		if tool.Description == "" || s.Type != "object" || s.AdditionalProperties == nil || *s.AdditionalProperties {
			t.Errorf("tool %s: not strict: %+v", tool.Name, tool)
		}
	}
	if strings.Join(names, ",") != "create_job,get_job,list_jobs" {
		t.Errorf("tools %v, want create_job, get_job, list_jobs", names)
	}
}

func TestCreateGetList(t *testing.T) {
	root := t.TempDir()
	rs := serve(t, root,
		call(1, "create_job", `{"request_id":"fase0-001"}`),
		call(2, "create_job", `{"request_id":"fase0-001"}`),
		call(3, "list_jobs", `{}`),
	)
	if len(rs) != 3 {
		t.Fatalf("got %d responses, want 3", len(rs))
	}
	created := okJob(t, rs[0])
	want := jobs.Job{ID: jobs.IDFor("fase0-001"), RequestID: "fase0-001", State: "intake", NextOperation: "run_intake"}
	if !reflect.DeepEqual(created, want) {
		t.Errorf("created %+v, want %+v", created, want)
	}
	if again := okJob(t, rs[1]); !reflect.DeepEqual(again, want) {
		t.Errorf("repeated create %+v, want %+v", again, want)
	}
	list := toolResult(t, rs[2])
	wantList := `{"jobs":[{"id":"` + want.ID + `","request_id":"fase0-001","state":"intake","next_operation":"run_intake"}]}`
	if list.IsError || list.Content[0].Text != wantList {
		t.Errorf("list %+v, want %s", list, wantList)
	}
	if files, _ := filepath.Glob(filepath.Join(root, "*.json")); len(files) != 1 {
		t.Errorf("job files %v, want one", files)
	}

	// A new session on the same store sees the job.
	rs = serve(t, root, call(4, "get_job", `{"id":"`+want.ID+`"}`))
	if got := okJob(t, rs[0]); !reflect.DeepEqual(got, want) {
		t.Errorf("get %+v, want %+v", got, want)
	}
}

func TestListEmpty(t *testing.T) {
	rs := serve(t, t.TempDir(), call(1, "list_jobs", `{}`))
	if res := toolResult(t, rs[0]); res.IsError || res.Content[0].Text != `{"jobs":[]}` {
		t.Errorf("list %+v, want {\"jobs\":[]}", res)
	}
}

func TestInvalidToolInput(t *testing.T) {
	tests := []struct{ tool, args, wantText string }{
		{"create_job", `{}`, "request_id"},
		{"create_job", `{"request_id":""}`, "request_id"},
		{"create_job", `{"request_id":"  "}`, "request_id"},
		{"create_job", `{"request_id":"x","extra":1}`, "extra"},
		{"create_job", `{"request_id":5}`, "request_id"},
		{"get_job", `{}`, "id"},
		{"get_job", `{"id":5}`, "id"},
		{"get_job", `{"id":"nope"}`, "not found"},
		{"get_job", `{"id":"` + jobs.IDFor("missing") + `"}`, "not found"},
		{"list_jobs", `{"x":1}`, `"x"`},
	}
	for _, tt := range tests {
		rs := serve(t, t.TempDir(), call(1, tt.tool, tt.args), `{"jsonrpc":"2.0","id":2,"method":"ping"}`)
		if len(rs) != 2 {
			t.Fatalf("%s %s: got %d responses, want 2", tt.tool, tt.args, len(rs))
		}
		res := toolResult(t, rs[0])
		if !res.IsError || !strings.Contains(res.Content[0].Text, tt.wantText) {
			t.Errorf("%s %s: result %+v, want isError naming %s", tt.tool, tt.args, res, tt.wantText)
		}
		if string(rs[1].Result) != "{}" {
			t.Errorf("%s %s: ping after it got %+v", tt.tool, tt.args, rs[1])
		}
	}
}

func TestStoreFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "job-0000000000000000.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	rs := serve(t, root, call(1, "list_jobs", `{}`))
	if res := toolResult(t, rs[0]); !res.IsError || !strings.Contains(res.Content[0].Text, "corrupt") {
		t.Errorf("list %+v, want isError with the store error", res)
	}
}

func TestProtocolErrors(t *testing.T) {
	rs := serve(t, t.TempDir(),
		`{"jsonrpc":"2.0","id":1,"method":"bogus"}`,
		call(2, "bogus_tool", `{}`),
		`not json`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
	)
	if len(rs) != 4 {
		t.Fatalf("got %d responses, want 4", len(rs))
	}
	want := []struct {
		id   string
		code int
	}{{"1", -32601}, {"2", -32602}, {"null", -32700}}
	for i, w := range want {
		if string(rs[i].ID) != w.id || rs[i].Error == nil || rs[i].Error.Code != w.code {
			t.Errorf("response %d: %+v, want id %s code %d", i, rs[i], w.id, w.code)
		}
	}
	if string(rs[3].Result) != "{}" {
		t.Errorf("ping after errors: %+v", rs[3])
	}
}
