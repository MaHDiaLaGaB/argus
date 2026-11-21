// Command argus is the CLI for the Argus AI development pipeline.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/argus-dev/argus/internal/contracts"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "version":
		printf(stdout, "argus %s (contracts schema %s)\n", version, contracts.SchemaVersion)
		return 0
	case "validate":
		return cmdValidate(args[1:], stdout, stderr)
	case "gate":
		return cmdGate(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		printf(stderr, "unknown command %q\n\n", args[0])
		usage(stderr)
		return 2
	}
}

// printf and println write CLI output; a failed write to stdout/stderr is not actionable.
func printf(w io.Writer, format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }

func println(w io.Writer, args ...any) { _, _ = fmt.Fprintln(w, args...) }

func usage(w io.Writer) {
	printf(w, `usage:
  argus version
  argus validate --kind <%s> <file.json>
  argus gate <gate.json>
`, strings.Join(contracts.Kinds(), "|"))
}

func cmdValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	kind := fs.String("kind", "", "contract kind")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *kind == "" || fs.NArg() != 1 {
		usage(stderr)
		return 2
	}
	v, ok := contracts.New(*kind)
	if !ok {
		printf(stderr, "unknown kind %q (allowed: %s)\n", *kind, strings.Join(contracts.Kinds(), ", "))
		return 2
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		println(stderr, err)
		return 1
	}
	if err := contracts.Decode(data, v); err != nil {
		println(stderr, err)
		return 1
	}
	printf(stdout, "ok: %s is a valid %s\n", fs.Arg(0), *kind)
	return 0
}

// cmdGate evaluates a gate input (state fields are ignored) and prints the decision.
func cmdGate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		usage(stderr)
		return 2
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		println(stderr, err)
		return 1
	}
	var g contracts.GateResult
	if err := json.Unmarshal(data, &g); err != nil {
		println(stderr, "decode:", err)
		return 1
	}
	g.State, g.BlockedReasons = g.Evaluate()
	out, _ := json.MarshalIndent(struct {
		State   string   `json:"state"`
		Reasons []string `json:"blocked_reasons"`
	}{g.State, g.BlockedReasons}, "", "  ")
	println(stdout, string(out))
	if g.State != "ready_to_merge" {
		return 1
	}
	return 0
}
