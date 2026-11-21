// Command argus is the CLI for the Argus AI development pipeline.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/argus-dev/argus/internal/config"
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
	case "schema":
		return cmdSchema(args[1:], stdout, stderr)
	case "config":
		return cmdConfig(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		printf(stderr, "unknown command %q\n\n", args[0])
		usage(stderr)
		return 2
	}
}

// readUserFile reads a file the user named on the command line. Reading any
// path the caller chooses is the purpose of the CLI, so the traversal rule does not apply.
func readUserFile(path string) ([]byte, error) {
	return os.ReadFile(path) // #nosec G304 G703 -- path is the CLI user's own argument
}

// printf and println write CLI output; a failed write to stdout/stderr is not actionable.
func printf(w io.Writer, format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }

func println(w io.Writer, args ...any) { _, _ = fmt.Fprintln(w, args...) }

func usage(w io.Writer) {
	printf(w, `usage:
  argus version
  argus validate --kind <%s> <file.json>
  argus gate <gate.json>
  argus schema [--kind <kind>] [--out <dir>]
  argus config check [--root <dir>]
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
	data, err := readUserFile(fs.Arg(0))
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
	data, err := readUserFile(args[0])
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

// cmdSchema prints one generated JSON Schema, or writes every schema to --out.
func cmdSchema(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("schema", flag.ContinueOnError)
	fs.SetOutput(stderr)
	kind := fs.String("kind", "", "contract kind to print")
	out := fs.String("out", "", "directory to write <kind>.json for every kind")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || (*kind == "") == (*out == "") {
		usage(stderr)
		return 2
	}
	if *kind != "" {
		b, err := contracts.Schema(*kind)
		if err != nil {
			println(stderr, err)
			return 2
		}
		_, _ = stdout.Write(b)
		return 0
	}
	if err := os.MkdirAll(*out, 0o750); err != nil {
		println(stderr, err)
		return 1
	}
	for _, k := range contracts.Kinds() {
		b, err := contracts.Schema(k)
		if err != nil {
			println(stderr, err)
			return 1
		}
		if err := os.WriteFile(filepath.Join(*out, k+".json"), b, 0o600); err != nil {
			println(stderr, err)
			return 1
		}
	}
	printf(stdout, "wrote %d schemas to %s\n", len(contracts.Kinds()), *out)
	return 0
}

// cmdConfig loads config/ and policy/ under --root and reports every problem.
// Services run the same check at startup and refuse to start on failure.
func cmdConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "check" {
		usage(stderr)
		return 2
	}
	fs := flag.NewFlagSet("config check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "directory that holds config/ and policy/")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		usage(stderr)
		return 2
	}
	c, err := config.Load(*root)
	if err != nil {
		println(stderr, err)
		return 1
	}
	printf(stdout, "ok: config and policy in %s are valid (%d models)\n", *root, len(c.Models()))
	return 0
}
