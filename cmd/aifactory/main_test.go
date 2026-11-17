package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/getsyntegrity/go-specs/mock"
	"github.com/getsyntegrity/go-specs/specs"
)

const generalHelpText = `aifactory dev

Usage: aifactory <command> [-h|--help]

Commands:
  mcp        Serve the factory tools over MCP
  serve      Serve the factory dashboard over HTTP
  status     Show the state of the factory jobs
  import-v2  Import jobs from a Software Factory v2 checkout
`

var commandHelpText = map[string]string{
	"mcp": `Usage: aifactory mcp [-h|--help]

Serve the factory tools over MCP.

Speaks MCP over stdin and stdout. Tools: create_job, get_job, list_jobs.
Jobs are stored in .aifactory/jobs under the current directory.
`,
	"serve": `Usage: aifactory serve [-h|--help]

Serve the factory dashboard over HTTP.

This command is not implemented yet.
`,
	"status": `Usage: aifactory status [-h|--help]

Show the state of the factory jobs.

This command is not implemented yet.
`,
	"import-v2": `Usage: aifactory import-v2 [-h|--help]

Import jobs from a Software Factory v2 checkout.

This command is not implemented yet.
`,
}

func call(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, strings.NewReader(""), &out, &errOut)
	return code, out.String(), errOut.String()
}

// spyServeMCP stands in for serveMCP over a mock.Controller.
type spyServeMCP struct {
	ctrl *mock.Controller
}

func (s spyServeMCP) Serve(stdin io.Reader, stdout, stderr io.Writer) int {
	r := s.ctrl.Method("Serve").Call(stdin, stdout, stderr)
	return mock.Value[int](r, 0)
}

// same matches exactly the value want, not just an equal one.
func same(want any) mock.ArgMatcher {
	return mock.Match(fmt.Sprintf("the same value as %p", want), func(v any) bool { return v == want })
}

func TestCommandLine(t *testing.T) {
	specs.Describe(t, "the aifactory command line", func(s *specs.Spec) {
		s.Describe("asking for the general help", func(s *specs.Spec) {
			for _, args := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
				s.It(fmt.Sprintf("prints the general help for %q", args), func(ctx *specs.Context) {
					code, stdout, stderr := call(args...)
					ctx.Expect(code).ToEqual(0)
					ctx.Expect(stdout).ToEqual(generalHelpText)
					ctx.Expect(stderr).ToEqual("")
				})
			}
		})

		s.Describe("asking for a command's help", func(s *specs.Spec) {
			for _, name := range []string{"mcp", "serve", "status", "import-v2"} {
				for _, args := range [][]string{{name}, {name, "-h"}, {name, "--help"}} {
					if len(args) == 1 && name == "mcp" {
						continue // without arguments mcp serves; see "running the MCP server"
					}
					s.It(fmt.Sprintf("prints the %s help for %q", name, args), func(ctx *specs.Context) {
						code, stdout, stderr := call(args...)
						ctx.Expect(code).ToEqual(0)
						ctx.Expect(stdout).ToEqual(commandHelpText[name])
						ctx.Expect(stderr).ToEqual("")
					})
				}
			}
		})

		s.Describe("passing input the command line does not expect", func(s *specs.Spec) {
			tests := []struct {
				args       []string
				wantStderr string
			}{
				{[]string{"bogus"}, "aifactory: unknown command \"bogus\"\n" + generalHelpText},
				{[]string{"--bogus"}, "aifactory: unknown command \"--bogus\"\n" + generalHelpText},
				{[]string{"status", "foo"}, "aifactory status: unexpected argument \"foo\"\n" + commandHelpText["status"]},
				{[]string{"mcp", "foo"}, "aifactory mcp: unexpected argument \"foo\"\n" + commandHelpText["mcp"]},
				{[]string{"mcp", "-h", "x"}, "aifactory mcp: unexpected argument \"-h\"\n" + commandHelpText["mcp"]},
				{[]string{"import-v2", "--bogus"}, "aifactory import-v2: unexpected argument \"--bogus\"\n" + commandHelpText["import-v2"]},
			}
			for _, tt := range tests {
				s.It(fmt.Sprintf("exits 2 with the error and help on stderr for %q", tt.args), func(ctx *specs.Context) {
					code, stdout, stderr := call(tt.args...)
					ctx.Expect(code).ToEqual(2)
					ctx.Expect(stdout).ToEqual("")
					ctx.Expect(stderr).ToEqual(tt.wantStderr)
				})
			}
		})

		s.Describe("running the same arguments twice", func(s *specs.Spec) {
			for _, args := range [][]string{nil, {"--help"}, {"serve"}, {"bogus"}, {"status", "foo"}} {
				s.It(fmt.Sprintf("gives the same code, stdout and stderr for %q", args), func(ctx *specs.Context) {
					code1, out1, err1 := call(args...)
					code2, out2, err2 := call(args...)
					ctx.Expect(code2).ToEqual(code1)
					ctx.Expect(out2).ToEqual(out1)
					ctx.Expect(err2).ToEqual(err1)
				})
			}
		})

		s.Describe("running the MCP server", func(s *specs.Spec) {
			s.It("hands stdin, stdout and stderr to the server and returns its exit code", func(ctx *specs.Context) {
				stdin := strings.NewReader("")
				var stdout, stderr bytes.Buffer
				ctrl := mock.NewController(ctx)
				ctrl.Method("Serve").Expect(same(stdin), same(&stdout), same(&stderr)).Times(1).Return(7)
				original := serveMCP
				serveMCP = spyServeMCP{ctrl: ctrl}.Serve
				ctx.Cleanup(func() { serveMCP = original })

				code := run([]string{"mcp"}, stdin, &stdout, &stderr)
				ctx.Expect(code).ToEqual(7)
				ctx.Expect(stderr.String()).ToEqual("")
			})
		})
	})
}
