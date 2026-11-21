package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// repoSchemaDir is SchemaDir seen from this package directory.
var repoSchemaDir = filepath.Join("..", "..", filepath.FromSlash(SchemaDir))

// TestSchemaFilesUpToDate is the drift check: CI fails if contracts/schema is stale.
func TestSchemaFilesUpToDate(t *testing.T) {
	for _, kind := range Kinds() {
		t.Run(kind, func(t *testing.T) {
			want, err := Schema(kind)
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(repoSchemaDir, kind+".json"))
			if err != nil {
				t.Fatalf("%v (run: make schema)", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("%s/%s.json is stale (run: make schema)", SchemaDir, kind)
			}
		})
	}
	entries, err := os.ReadDir(repoSchemaDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !slices.Contains(Kinds(), strings.TrimSuffix(e.Name(), ".json")) {
			t.Errorf("%s/%s has no contract kind (delete it)", SchemaDir, e.Name())
		}
	}
}

func TestSchemaUnknownKind(t *testing.T) {
	if _, err := Schema("nope"); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestSchemaShape(t *testing.T) {
	cases := []struct {
		kind, path string
		want       string // JSON of the schema node at path
	}{
		{"ticket", "schema_version", `{"const":"3","type":"string"}`},
		{"ticket", "repo_hint", `{"type":["string","null"]}`},
		{"ticket", "created_at", `{"format":"date-time","type":"string"}`},
		{"workticket", "tier", `{"maximum":4,"minimum":0,"type":"integer"}`},
		{"workticket", "line_cap", `{"type":"integer"}`},
		{"workticket", "scope_files", `{"items":{"type":"string"},"type":["array","null"]}`},
		{"gate", "ci_checks", `{"additionalProperties":{"type":"string"},"type":["object","null"]}`},
		{"score", "total", `{"type":"number"}`},
		{"score", "reverted_within_14d", `{"type":["boolean","null"]}`},
		{"envelope", "budget.max_usd", `{"type":"number"}`},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"."+tc.path, func(t *testing.T) {
			node := loadSchema(t, tc.kind)
			for _, p := range strings.Split(tc.path, ".") {
				node = node["properties"].(map[string]any)[p].(map[string]any)
			}
			got, _ := json.Marshal(node)
			if string(got) != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
	root := loadSchema(t, "review")
	if root["additionalProperties"] != false || root["$schema"] != JSONSchemaDialect || root["title"] != "ReviewReport" {
		t.Fatalf("bad root: %v", root)
	}
}

// TestValidFixturesMatchSchema checks every valid fixture against its schema,
// so the schema never rejects a message the Go decoder accepts.
func TestValidFixturesMatchSchema(t *testing.T) {
	for _, kind := range Kinds() {
		t.Run(kind, func(t *testing.T) {
			var doc any
			if err := json.Unmarshal(loadFixture(t, kind), &doc); err != nil {
				t.Fatal(err)
			}
			if problems := conform(loadSchema(t, kind), doc, kind); len(problems) > 0 {
				t.Fatalf("fixture does not match schema:\n%s", strings.Join(problems, "\n"))
			}
		})
	}
}

func TestConformRejects(t *testing.T) {
	cases := []struct{ name, want string }{
		{"unknown field", "unknown field surprise"},
		{"missing field", "missing title"},
		{"wrong type", "want [string]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(loadFixture(t, "ticket"), &doc); err != nil {
				t.Fatal(err)
			}
			switch tc.name {
			case "unknown field":
				doc["surprise"] = 1.0
			case "missing field":
				delete(doc, "title")
			case "wrong type":
				doc["title"] = 5.0
			}
			got := strings.Join(conform(loadSchema(t, "ticket"), doc, "ticket"), "|")
			if !strings.Contains(got, tc.want) {
				t.Fatalf("problems %q missing %q", got, tc.want)
			}
		})
	}
}

func loadSchema(t *testing.T, kind string) map[string]any {
	t.Helper()
	b, err := Schema(kind)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// conform is a minimal checker for the schema subset Schema emits:
// type, const, minimum, maximum, properties, required, additionalProperties, items.
func conform(s map[string]any, v any, at string) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, at+": "+fmt.Sprintf(format, args...)) }

	var types []string
	switch ty := s["type"].(type) {
	case string:
		types = []string{ty}
	case []any:
		for _, x := range ty {
			types = append(types, x.(string))
		}
	}
	got := jsonType(v)
	numberOK := got == "integer" && slices.Contains(types, "number") // an integer is also a number
	if !slices.Contains(types, got) && !numberOK {
		add("type %s, want %v", got, types)
		return problems
	}
	if c, ok := s["const"]; ok && v != c {
		add("value %v, want const %v", v, c)
	}
	if n, ok := v.(float64); ok {
		if lo, ok := s["minimum"].(float64); ok && n < lo {
			add("%v below minimum %v", n, lo)
		}
		if hi, ok := s["maximum"].(float64); ok && n > hi {
			add("%v above maximum %v", n, hi)
		}
	}
	switch x := v.(type) {
	case map[string]any:
		props, _ := s["properties"].(map[string]any)
		req, _ := s["required"].([]any)
		for _, r := range req {
			if _, ok := x[r.(string)]; !ok {
				add("missing %s", r)
			}
		}
		for k, val := range x {
			if ps, ok := props[k]; ok {
				problems = append(problems, conform(ps.(map[string]any), val, at+"."+k)...)
				continue
			}
			switch ap := s["additionalProperties"].(type) {
			case bool:
				if !ap {
					add("unknown field %s", k)
				}
			case map[string]any:
				problems = append(problems, conform(ap, val, at+"."+k)...)
			}
		}
	case []any:
		items, _ := s["items"].(map[string]any)
		for i, el := range x {
			problems = append(problems, conform(items, el, fmt.Sprintf("%s[%d]", at, i))...)
		}
	}
	return problems
}

func jsonType(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		if x == float64(int64(x)) {
			return "integer"
		}
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "unknown"
}
