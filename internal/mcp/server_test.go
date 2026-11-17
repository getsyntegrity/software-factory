package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/getsyntegrity/go-specs/mock"
	"github.com/getsyntegrity/go-specs/specs"
	"github.com/getsyntegrity/software-factory/internal/jobs"
	"github.com/getsyntegrity/software-factory/internal/store"
)

// spyStore is a jobStore over a mock.Controller. A method with no expectation
// fails the test when the server calls it.
type spyStore struct {
	ctrl *mock.Controller
}

func (s spyStore) Create(requestID string) (jobs.Job, error) {
	r := s.ctrl.Method("Create").Call(requestID)
	return mock.Value[jobs.Job](r, 0), r.Err(1)
}

func (s spyStore) Get(id string) (jobs.Job, error) {
	r := s.ctrl.Method("Get").Call(id)
	return mock.Value[jobs.Job](r, 0), r.Err(1)
}

func (s spyStore) List() ([]jobs.Job, error) {
	r := s.ctrl.Method("List").Call()
	return mock.Value[[]jobs.Job](r, 0), r.Err(1)
}

func newSpyStore(ctx *specs.Context) spyStore {
	return spyStore{ctrl: mock.NewController(ctx)}
}

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
// output line, each of which must be JSON-RPC 2.0.
func serve(ctx *specs.Context, s jobStore, lines ...string) []rpcResponse {
	var out bytes.Buffer
	ctx.Expect(Serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out, s, "test")).To(specs.BeNil())
	var responses []rpcResponse
	for _, line := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var r rpcResponse
		ctx.Expect(json.Unmarshal([]byte(line), &r)).To(specs.BeNil())
		ctx.Expect(r.JSONRPC).ToEqual("2.0")
		responses = append(responses, r)
	}
	return responses
}

func call(id int, tool, args string) string {
	return `{"jsonrpc":"2.0","id":` + strconv.Itoa(id) + `,"method":"tools/call","params":{"name":"` + tool + `","arguments":` + args + `}}`
}

const ping = `{"jsonrpc":"2.0","id":2,"method":"ping"}`

// toolResult decodes r as a tool result holding one text item.
func toolResult(ctx *specs.Context, r rpcResponse) callResult {
	ctx.Expect(r.Error).To(specs.BeNil())
	var res callResult
	ctx.Expect(json.Unmarshal(r.Result, &res)).To(specs.BeNil())
	ctx.Expect(res.Content).To(specs.HaveLen(1))
	if len(res.Content) != 1 {
		return res
	}
	ctx.Expect(res.Content[0].Type).ToEqual("text")
	return res
}

// text returns the text of a tool result, or "" when it has none.
func text(res callResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].Text
}

// okJob decodes r as a successful tool result holding a job.
func okJob(ctx *specs.Context, r rpcResponse) jobs.Job {
	res := toolResult(ctx, r)
	ctx.Expect(res.IsError).To(specs.BeFalse())
	var job jobs.Job
	ctx.Expect(json.Unmarshal([]byte(text(res)), &job)).To(specs.BeNil())
	return job
}

const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`

func storedJob() jobs.Job {
	return jobs.Job{ID: jobs.IDFor("fase0-001"), RequestID: "fase0-001", State: "intake", NextOperation: "run_intake"}
}

func TestServer(t *testing.T) {
	specs.Describe(t, "the MCP server", func(s *specs.Spec) {
		s.Describe("negotiating the protocol version", func(s *specs.Spec) {
			cases := []struct{ requested, want string }{
				{"2025-06-18", "2025-06-18"},
				{"2024-11-05", "2024-11-05"},
				{"2025-03-26", "2025-03-26"},
				{"2025-11-25", "2025-11-25"},
				{"1999-01-01", "2025-11-25"},
			}
			for _, tc := range cases {
				s.It("answers "+tc.want+" with tools and server info when asked for "+tc.requested, func(ctx *specs.Context) {
					line := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` + tc.requested + `"}}`
					rs := serve(ctx, newSpyStore(ctx), line)
					ctx.Expect(rs).To(specs.HaveLen(1))
					if len(rs) != 1 {
						return
					}
					ctx.Expect(string(rs[0].ID)).ToEqual("1")
					var res struct {
						ProtocolVersion string `json:"protocolVersion"`
						Capabilities    struct {
							Tools *struct{} `json:"tools"`
						} `json:"capabilities"`
						ServerInfo struct{ Name, Version string } `json:"serverInfo"`
					}
					ctx.Expect(json.Unmarshal(rs[0].Result, &res)).To(specs.BeNil())
					ctx.Expect(res.ProtocolVersion).ToEqual(tc.want)
					ctx.Expect(res.Capabilities.Tools).To(specs.Not(specs.BeNil()))
					ctx.Expect(res.ServerInfo.Name).ToEqual("aifactory")
					ctx.Expect(res.ServerInfo.Version).ToEqual("test")
				})
			}
		})

		s.Describe("answering notifications and pings", func(s *specs.Spec) {
			s.It("sends no response to a notification", func(ctx *specs.Context) {
				rs := serve(ctx, newSpyStore(ctx), `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
				ctx.Expect(rs).To(specs.HaveLen(0))
			})

			s.It("answers a ping with an empty result and the same id", func(ctx *specs.Context) {
				rs := serve(ctx, newSpyStore(ctx), `{"jsonrpc":"2.0","id":"a","method":"ping"}`)
				ctx.Expect(rs).To(specs.HaveLen(1))
				if len(rs) != 1 {
					return
				}
				ctx.Expect(string(rs[0].ID)).ToEqual(`"a"`)
				ctx.Expect(string(rs[0].Result)).ToEqual("{}")
			})
		})

		s.Describe("listing the available tools", func(s *specs.Spec) {
			s.It("offers create_job, get_job and list_jobs, each described with a strict object schema", func(ctx *specs.Context) {
				rs := serve(ctx, newSpyStore(ctx), initialize, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
				ctx.Expect(rs).To(specs.HaveLen(2))
				if len(rs) != 2 {
					return
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
				ctx.Expect(json.Unmarshal(rs[1].Result, &res)).To(specs.BeNil())
				var names []string
				for _, tool := range res.Tools {
					names = append(names, tool.Name)
					ctx.Expect(tool.Description).To(specs.Not(specs.BeEmpty()))
					ctx.Expect(tool.InputSchema.Type).ToEqual("object")
					ctx.Expect(tool.InputSchema.AdditionalProperties).To(specs.Not(specs.BeNil()))
					if tool.InputSchema.AdditionalProperties != nil {
						ctx.Expect(*tool.InputSchema.AdditionalProperties).To(specs.BeFalse())
					}
				}
				ctx.Expect(strings.Join(names, ",")).ToEqual("create_job,get_job,list_jobs")
			})
		})

		s.Describe("creating, getting and listing jobs through the store", func(s *specs.Spec) {
			s.It("creates the job for a request ID and returns it, again on a repeated call", func(ctx *specs.Context) {
				spy := newSpyStore(ctx)
				spy.ctrl.Method("Create").Expect("fase0-001").Times(2).Return(storedJob(), nil)
				rs := serve(ctx, spy,
					call(1, "create_job", `{"request_id":"fase0-001"}`),
					call(2, "create_job", `{"request_id":"fase0-001"}`),
				)
				ctx.Expect(rs).To(specs.HaveLen(2))
				if len(rs) != 2 {
					return
				}
				ctx.Expect(okJob(ctx, rs[0])).ToEqual(storedJob())
				ctx.Expect(okJob(ctx, rs[1])).ToEqual(storedJob())
			})

			s.It("lists the jobs the store returns", func(ctx *specs.Context) {
				spy := newSpyStore(ctx)
				spy.ctrl.Method("List").Expect().Return([]jobs.Job{storedJob()}, nil)
				rs := serve(ctx, spy, call(1, "list_jobs", `{}`))
				ctx.Expect(rs).To(specs.HaveLen(1))
				if len(rs) != 1 {
					return
				}
				res := toolResult(ctx, rs[0])
				ctx.Expect(res.IsError).To(specs.BeFalse())
				want := `{"jobs":[{"id":"` + storedJob().ID + `","request_id":"fase0-001","state":"intake","next_operation":"run_intake"}]}`
				ctx.Expect(text(res)).ToEqual(want)
			})

			s.It("gets the job with the given id from the store", func(ctx *specs.Context) {
				spy := newSpyStore(ctx)
				spy.ctrl.Method("Get").Expect(storedJob().ID).Return(storedJob(), nil)
				rs := serve(ctx, spy, call(1, "get_job", `{"id":"`+storedJob().ID+`"}`))
				ctx.Expect(rs).To(specs.HaveLen(1))
				if len(rs) != 1 {
					return
				}
				ctx.Expect(okJob(ctx, rs[0])).ToEqual(storedJob())
			})

			s.It("lists an empty store as an empty jobs array", func(ctx *specs.Context) {
				spy := newSpyStore(ctx)
				spy.ctrl.Method("List").Expect().Return([]jobs.Job{}, nil)
				rs := serve(ctx, spy, call(1, "list_jobs", `{}`))
				ctx.Expect(rs).To(specs.HaveLen(1))
				if len(rs) != 1 {
					return
				}
				res := toolResult(ctx, rs[0])
				ctx.Expect(res.IsError).To(specs.BeFalse())
				ctx.Expect(text(res)).ToEqual(`{"jobs":[]}`)
			})

			s.It("reports a store failure as a tool error with the store's message", func(ctx *specs.Context) {
				spy := newSpyStore(ctx)
				spy.ctrl.Method("List").Expect().Return(nil, errors.New("store: corrupt job file"))
				rs := serve(ctx, spy, call(1, "list_jobs", `{}`))
				ctx.Expect(rs).To(specs.HaveLen(1))
				if len(rs) != 1 {
					return
				}
				res := toolResult(ctx, rs[0])
				ctx.Expect(res.IsError).To(specs.BeTrue())
				ctx.Expect(text(res)).To(specs.Contain("store: corrupt job file"))
			})
		})

		s.Describe("calling a tool with invalid arguments", func(s *specs.Spec) {
			// get is the id the store is asked for and reports missing; when
			// empty, validation fails before the store and it gets no call.
			cases := []struct{ tool, args, wantText, get string }{
				{"create_job", `{}`, "request_id", ""},
				{"create_job", `{"request_id":""}`, "request_id", ""},
				{"create_job", `{"request_id":"  "}`, "request_id", ""},
				{"create_job", `{"request_id":"x","extra":1}`, "extra", ""},
				{"create_job", `{"request_id":5}`, "request_id", ""},
				{"get_job", `{}`, "id", ""},
				{"get_job", `{"id":5}`, "id", ""},
				{"get_job", `{"id":"nope"}`, "not found", "nope"},
				{"get_job", `{"id":"` + jobs.IDFor("missing") + `"}`, "not found", jobs.IDFor("missing")},
				{"list_jobs", `{"x":1}`, `"x"`, ""},
			}
			for _, tc := range cases {
				name := "rejects " + tc.tool + " " + tc.args + " naming " + tc.wantText + " and still answers a ping"
				if tc.get == "" {
					name += ", without calling the store"
				}
				s.It(name, func(ctx *specs.Context) {
					spy := newSpyStore(ctx)
					if tc.get != "" {
						spy.ctrl.Method("Get").Expect(tc.get).Return(jobs.Job{}, errors.Join(errors.New("spy: "+tc.get), store.ErrNotFound))
					}
					rs := serve(ctx, spy, call(1, tc.tool, tc.args), ping)
					ctx.Expect(rs).To(specs.HaveLen(2))
					if len(rs) != 2 {
						return
					}
					res := toolResult(ctx, rs[0])
					ctx.Expect(res.IsError).To(specs.BeTrue())
					ctx.Expect(text(res)).To(specs.Contain(tc.wantText))
					ctx.Expect(string(rs[1].Result)).ToEqual("{}")
				})
			}
		})

		s.Describe("answering malformed or unknown requests", func(s *specs.Spec) {
			s.It("answers an unknown method, an unknown tool and bad JSON with errors, then still answers a ping", func(ctx *specs.Context) {
				rs := serve(ctx, newSpyStore(ctx),
					`{"jsonrpc":"2.0","id":1,"method":"bogus"}`,
					call(2, "bogus_tool", `{}`),
					`not json`,
					`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
				)
				ctx.Expect(rs).To(specs.HaveLen(4))
				if len(rs) != 4 {
					return
				}
				want := []struct {
					id   string
					code int
				}{{"1", -32601}, {"2", -32602}, {"null", -32700}}
				for i, w := range want {
					ctx.Expect(string(rs[i].ID)).ToEqual(w.id)
					ctx.Expect(rs[i].Error).To(specs.Not(specs.BeNil()))
					if rs[i].Error != nil {
						ctx.Expect(rs[i].Error.Code).ToEqual(w.code)
					}
				}
				ctx.Expect(string(rs[3].Result)).ToEqual("{}")
			})
		})
	})
}
