package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/getsyntegrity/software-factory/internal/jobs"
	"github.com/getsyntegrity/software-factory/internal/store"
)

// JSON-RPC 2.0 error codes.
const (
	codeParse          = -32700
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// latestVersion is answered when the client asks for an unknown version.
const latestVersion = "2025-11-25"

var protocolVersions = []string{"2024-11-05", "2025-03-26", "2025-06-18", latestVersion}

type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

var tools = []tool{
	{
		Name: "create_job",
		Description: "Create a factory job for request_id. Reuse the same request_id when repeating " +
			"the same request: a repeat returns the existing job instead of creating another.",
		InputSchema: objectSchema(map[string]any{"request_id": map[string]any{"type": "string", "minLength": 1}}),
	},
	{
		Name:        "get_job",
		Description: "Get a factory job by id, including its state and next_operation.",
		InputSchema: objectSchema(map[string]any{"id": map[string]any{"type": "string"}}),
	},
	{
		Name:        "list_jobs",
		Description: "List every factory job, sorted by id.",
		InputSchema: objectSchema(map[string]any{}),
	},
}

// objectSchema returns a strict object schema whose properties are all required.
func objectSchema(properties map[string]any) map[string]any {
	required := []string{}
	for name := range properties {
		required = append(required, name)
	}
	sort.Strings(required)
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

// jobStore is what the server runs job operations on. A pointer to a
// store.Store satisfies it.
type jobStore interface {
	Create(requestID string) (jobs.Job, error)
	Get(id string) (jobs.Job, error)
	List() ([]jobs.Job, error)
}

// Serve reads newline-delimited JSON-RPC 2.0 messages from in and writes one
// response per line to out, using s for every job operation. It returns nil
// when in reaches EOF.
func Serve(in io.Reader, out io.Writer, s jobStore, version string) error {
	r := bufio.NewReader(in)
	enc := json.NewEncoder(out)
	for {
		line, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			if resp, ok := handle(line, s, version); ok {
				if err := enc.Encode(resp); err != nil {
					return err
				}
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// handle returns the response to one message, and false for a notification.
func handle(line []byte, s jobStore, version string) (response, bool) {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParse, "parse error"}}, true
	}
	if len(req.ID) == 0 {
		return response{}, false
	}
	resp := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		json.Unmarshal(req.Params, &p)
		v := latestVersion
		if slices.Contains(protocolVersions, p.ProtocolVersion) {
			v = p.ProtocolVersion
		}
		resp.Result = map[string]any{
			"protocolVersion": v,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "aifactory", "version": version},
		}
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": tools}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp.Error = &rpcError{codeInvalidParams, "invalid params: " + err.Error()}
			break
		}
		result, ok := callTool(p.Name, p.Arguments, s)
		if !ok {
			resp.Error = &rpcError{codeInvalidParams, fmt.Sprintf("unknown tool %q", p.Name)}
			break
		}
		resp.Result = result
	default:
		resp.Error = &rpcError{codeMethodNotFound, fmt.Sprintf("method %q not found", req.Method)}
	}
	return resp, true
}

// callTool runs the named tool and returns its result, and false for an
// unknown tool.
func callTool(name string, raw json.RawMessage, s jobStore) (map[string]any, bool) {
	var fields []string
	switch name {
	case "create_job":
		fields = []string{"request_id"}
	case "get_job":
		fields = []string{"id"}
	case "list_jobs":
	default:
		return nil, false
	}
	args, err := stringArgs(raw, fields)
	if err != nil {
		return textResult(err.Error(), true), true
	}
	var value any
	switch name {
	case "create_job":
		if strings.TrimSpace(args["request_id"]) == "" {
			return textResult(`field "request_id" is empty`, true), true
		}
		value, err = s.Create(args["request_id"])
	case "get_job":
		value, err = s.Get(args["id"])
		if errors.Is(err, store.ErrNotFound) {
			return textResult(fmt.Sprintf("job %q not found", args["id"]), true), true
		}
	case "list_jobs":
		var list []jobs.Job
		list, err = s.List()
		value = map[string]any{"jobs": list}
	}
	if err != nil {
		return textResult(err.Error(), true), true
	}
	data, err := json.Marshal(value)
	if err != nil {
		return textResult(err.Error(), true), true
	}
	return textResult(string(data), false), true
}

// stringArgs decodes raw as an object holding exactly the given string fields.
func stringArgs(raw json.RawMessage, fields []string) (map[string]string, error) {
	obj := map[string]json.RawMessage{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, errors.New("arguments must be an object")
		}
	}
	names := make([]string, 0, len(obj))
	for name := range obj {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !slices.Contains(fields, name) {
			return nil, fmt.Errorf("unknown field %q", name)
		}
	}
	args := map[string]string{}
	for _, name := range fields {
		v, ok := obj[name]
		if !ok {
			return nil, fmt.Errorf("missing required field %q", name)
		}
		var str string
		if err := json.Unmarshal(v, &str); err != nil || string(v) == "null" {
			return nil, fmt.Errorf("field %q must be a string", name)
		}
		args[name] = str
	}
	return args, nil
}

func textResult(text string, isError bool) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isError,
	}
}
