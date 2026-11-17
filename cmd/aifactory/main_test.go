package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
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

func TestGeneralHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
		code, stdout, stderr := call(args...)
		if code != 0 {
			t.Errorf("%q: exit code %d, want 0", args, code)
		}
		if stdout != generalHelpText {
			t.Errorf("%q: stdout\n%s\nwant\n%s", args, stdout, generalHelpText)
		}
		if stderr != "" {
			t.Errorf("%q: stderr %q, want empty", args, stderr)
		}
	}
}

func TestCommandHelp(t *testing.T) {
	for _, name := range []string{"mcp", "serve", "status", "import-v2"} {
		for _, args := range [][]string{{name}, {name, "-h"}, {name, "--help"}} {
			if len(args) == 1 && name == "mcp" {
				continue // without arguments mcp serves; see TestMCPServes
			}
			code, stdout, stderr := call(args...)
			if code != 0 {
				t.Errorf("%q: exit code %d, want 0", args, code)
			}
			if stdout != commandHelpText[name] {
				t.Errorf("%q: stdout\n%s\nwant\n%s", args, stdout, commandHelpText[name])
			}
			if stderr != "" {
				t.Errorf("%q: stderr %q, want empty", args, stderr)
			}
		}
	}
}

func TestUnknownInput(t *testing.T) {
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
		code, stdout, stderr := call(tt.args...)
		if code != 2 {
			t.Errorf("%q: exit code %d, want 2", tt.args, code)
		}
		if stdout != "" {
			t.Errorf("%q: stdout %q, want empty", tt.args, stdout)
		}
		if stderr != tt.wantStderr {
			t.Errorf("%q: stderr\n%s\nwant\n%s", tt.args, stderr, tt.wantStderr)
		}
	}
}

func TestStableOutput(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"serve"}, {"bogus"}, {"status", "foo"}} {
		code1, out1, err1 := call(args...)
		code2, out2, err2 := call(args...)
		if code1 != code2 || out1 != out2 || err1 != err2 {
			t.Errorf("%q: two calls differ", args)
		}
	}
}

func TestMCPServes(t *testing.T) {
	t.Chdir(t.TempDir())
	in := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_job","arguments":{"request_id":"fase0-001"}}}
`
	var out, errOut bytes.Buffer
	if code := run([]string{"mcp"}, strings.NewReader(in), &out, &errOut); code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	if errOut.String() != "" {
		t.Errorf("stderr %q, want empty", errOut.String())
	}
	if lines := strings.Count(out.String(), "\n"); lines != 2 {
		t.Errorf("stdout has %d lines, want 2:\n%s", lines, out.String())
	}
	files, _ := filepath.Glob(filepath.Join(".aifactory", "jobs", "job-*.json"))
	if len(files) != 1 {
		t.Errorf("job files %v, want one", files)
	}
}
