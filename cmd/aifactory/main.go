// Command aifactory is the entry point for the Go implementation.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/getsyntegrity/software-factory/internal/mcp"
	"github.com/getsyntegrity/software-factory/internal/store"
)

// version is set at build time.
var version = "dev"

const notImplemented = "This command is not implemented yet.\n"

// command is a subcommand, its one-line description, the details its help
// ends with, and what it runs without arguments (nil prints the help).
type command struct {
	name, summary, details string
	run                    func(stdin io.Reader, stdout, stderr io.Writer) int
}

// commands lists the subcommands in the order the help shows them.
var commands = []command{
	{"mcp", "Serve the factory tools over MCP",
		"Speaks MCP over stdin and stdout. Tools: create_job, get_job, list_jobs.\n" +
			"Jobs are stored in .aifactory/jobs under the current directory.\n",
		func(stdin io.Reader, stdout, stderr io.Writer) int { return serveMCP(stdin, stdout, stderr) }},
	{"serve", "Serve the factory dashboard over HTTP", notImplemented, nil},
	{"status", "Show the state of the factory jobs", notImplemented, nil},
	{"import-v2", "Import jobs from a Software Factory v2 checkout", notImplemented, nil},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run dispatches args (without the program name) and returns the exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) || args[0] == "help" {
		fmt.Fprint(stdout, generalHelp())
		return 0
	}
	for _, c := range commands {
		if args[0] != c.name {
			continue
		}
		rest := args[1:]
		if len(rest) == 0 && c.run != nil {
			return c.run(stdin, stdout, stderr)
		}
		if len(rest) == 0 || (len(rest) == 1 && isHelp(rest[0])) {
			fmt.Fprint(stdout, commandHelp(c))
			return 0
		}
		fmt.Fprintf(stderr, "aifactory %s: unexpected argument %q\n", c.name, rest[0])
		fmt.Fprint(stderr, commandHelp(c))
		return 2
	}
	fmt.Fprintf(stderr, "aifactory: unknown command %q\n", args[0])
	fmt.Fprint(stderr, generalHelp())
	return 2
}

func isHelp(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func generalHelp() string {
	s := "aifactory " + version + "\n\n" +
		"Usage: aifactory <command> [-h|--help]\n\n" +
		"Commands:\n"
	for _, c := range commands {
		s += fmt.Sprintf("  %-10s %s\n", c.name, c.summary)
	}
	return s
}

func commandHelp(c command) string {
	return "Usage: aifactory " + c.name + " [-h|--help]\n\n" +
		c.summary + ".\n\n" +
		c.details
}

// serveMCP is what aifactory mcp runs; tests replace it.
var serveMCP = runMCP

// runMCP serves MCP on stdin and stdout with the store in .aifactory/jobs.
func runMCP(stdin io.Reader, stdout, stderr io.Writer) int {
	s, err := store.Open(filepath.Join(".aifactory", "jobs"))
	if err != nil {
		fmt.Fprintf(stderr, "aifactory mcp: %v\n", err)
		return 1
	}
	if err := mcp.Serve(stdin, stdout, s, version); err != nil {
		fmt.Fprintf(stderr, "aifactory mcp: %v\n", err)
		return 1
	}
	return 0
}
