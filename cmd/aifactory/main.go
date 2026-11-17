// Command aifactory is the entry point for the Go implementation.
package main

import (
	"fmt"
	"io"
	"os"
)

// version is set at build time.
var version = "dev"

// command is a subcommand and its one-line description.
type command struct {
	name, summary string
}

// commands lists the subcommands in the order the help shows them.
var commands = []command{
	{"mcp", "Serve the factory tools over MCP"},
	{"serve", "Serve the factory dashboard over HTTP"},
	{"status", "Show the state of the factory jobs"},
	{"import-v2", "Import jobs from a Software Factory v2 checkout"},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches args (without the program name) and returns the exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) || args[0] == "help" {
		fmt.Fprint(stdout, generalHelp())
		return 0
	}
	for _, c := range commands {
		if args[0] != c.name {
			continue
		}
		rest := args[1:]
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
		"This command is not implemented yet.\n"
}
