package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtures = "../../internal/contracts/testdata/valid"

func TestRunVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"version"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "contracts schema 3") {
		t.Fatalf("unexpected output %q", out.String())
	}
}

func TestRunValidate(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"validate", "--kind", "workticket", filepath.Join(fixtures, "workticket.json")}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}

	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte(`{"schema_version":"3"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errb.Reset()
	if code := run([]string{"validate", "--kind", "workticket", bad}, &out, &errb); code != 1 {
		t.Fatalf("expected exit 1 for invalid ticket, got %d", code)
	}
	if !strings.Contains(errb.String(), "scope_files") {
		t.Fatalf("expected scope_files problem, got %q", errb.String())
	}
}

func TestRunGate(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"gate", filepath.Join(fixtures, "gate.json")}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s %s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "ready_to_merge") {
		t.Fatalf("unexpected output %q", out.String())
	}
}

func TestRunUnknown(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"fly"}, &out, &errb); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestRunSchema(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"schema", "--kind", "ticket"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"title": "Ticket"`) {
		t.Fatalf("unexpected output %q", out.String())
	}

	dir := t.TempDir()
	out.Reset()
	if code := run([]string{"schema", "--out", dir}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "workticket.json")); err != nil {
		t.Fatal(err)
	}

	for _, bad := range [][]string{{"schema"}, {"schema", "--kind", "nope"}, {"schema", "--kind", "ticket", "--out", dir}} {
		if code := run(bad, &out, &errb); code != 2 {
			t.Errorf("%v: expected exit 2, got %d", bad, code)
		}
	}
}
