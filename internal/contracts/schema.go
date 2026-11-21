package contracts

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// JSONSchemaDialect is the JSON Schema draft the generated schemas follow.
const JSONSchemaDialect = "https://json-schema.org/draft/2020-12/schema"

// SchemaDir is where generated schemas are checked in, relative to the repo root.
const SchemaDir = "contracts/schema"

var (
	tierType = reflect.TypeOf(Tier(0))
	timeType = reflect.TypeOf(time.Time{})
)

// Schema returns the generated JSON Schema for a contract kind, as indented JSON
// ending in a newline. It is derived from the Go struct by reflection, so the
// structs stay the single source of truth (ARCHITECTURE.md Section 5).
//
// The schema describes shape only: field names, types, nullability and no
// unknown fields. Cross-field rules stay in Validate.
func Schema(kind string) ([]byte, error) {
	v, ok := New(kind)
	if !ok {
		return nil, fmt.Errorf("unknown kind %q", kind)
	}
	t := reflect.TypeOf(v).Elem()
	root := typeSchema(t)
	root["$schema"] = JSONSchemaDialect
	root["$id"] = kind + ".json"
	root["title"] = t.Name()
	root["description"] = fmt.Sprintf("argus %s contract, schema_version %s. Generated from internal/contracts; do not edit.", t.Name(), SchemaVersion)
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// typeSchema maps a Go type to a JSON Schema object.
func typeSchema(t reflect.Type) map[string]any {
	switch {
	case t == tierType:
		return map[string]any{"type": "integer", "minimum": 0, "maximum": 4}
	case t == timeType:
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.Pointer:
		s := typeSchema(t.Elem())
		s["type"] = []any{s["type"], "null"}
		return s
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		// encoding/json writes a nil slice as null.
		return map[string]any{"type": []any{"array", "null"}, "items": typeSchema(t.Elem())}
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			panic("contracts: schema: map key must be string in " + t.String())
		}
		return map[string]any{"type": []any{"object", "null"}, "additionalProperties": typeSchema(t.Elem())}
	case reflect.Struct:
		return structSchema(t)
	default:
		panic("contracts: schema: unsupported type " + t.String())
	}
}

// structSchema lists every JSON field as a required property and forbids
// unknown fields, matching Decode, which rejects unknown fields.
func structSchema(t reflect.Type) map[string]any {
	props := map[string]any{}
	required := []string{}
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		s := typeSchema(f.Type)
		if name == "schema_version" && f.Type.Kind() == reflect.String {
			s = map[string]any{"type": "string", "const": SchemaVersion}
		}
		props[name] = s
		if !strings.Contains(opts, "omitempty") {
			required = append(required, name)
		}
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}
