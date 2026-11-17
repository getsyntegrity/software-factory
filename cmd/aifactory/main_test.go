package main

import (
	"bytes"
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

This command is not implemented yet.
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
	code = run(args, &out, &errOut)
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
