// Package contracts defines the messages exchanged between pipeline components
// (ARCHITECTURE.md Section 5) and the deterministic rules that govern them.
//
// Every contract type has a Validate method. Decode reads JSON strictly
// (unknown fields are rejected) and then validates, so a malformed message
// from a component or a model never enters the pipeline.
package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
)

// SchemaVersion is the current version of every contract in this package.
const SchemaVersion = "3"

// Validator is implemented by every contract type.
type Validator interface {
	Validate() error
}

// ValidationError lists every problem found in a message, not only the first.
type ValidationError struct {
	Type     string
	Problems []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: invalid: %s", e.Type, strings.Join(e.Problems, "; "))
}

// Decode parses data into v, rejecting unknown fields and trailing data, then validates v.
func Decode(data []byte, v Validator) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if dec.More() {
		return errors.New("decode: trailing data after JSON object")
	}
	return v.Validate()
}

// checker collects validation problems.
type checker struct {
	typ      string
	problems []string
}

func newChecker(typ string) *checker { return &checker{typ: typ} }

func (c *checker) check(ok bool, format string, args ...any) {
	if !ok {
		c.problems = append(c.problems, fmt.Sprintf(format, args...))
	}
}

func (c *checker) required(value, field string) {
	c.check(strings.TrimSpace(value) != "", "%s is required", field)
}

func (c *checker) oneOf(value, field string, allowed ...string) {
	c.check(slices.Contains(allowed, value), "%s %q must be one of %v", field, value, allowed)
}

func (c *checker) schema(v string) {
	c.check(v == SchemaVersion, "schema_version %q must be %q", v, SchemaVersion)
}

func (c *checker) err() error {
	if len(c.problems) == 0 {
		return nil
	}
	return &ValidationError{Type: c.typ, Problems: c.problems}
}

// Tier is the risk tier of a piece of work (ARCHITECTURE.md Section 8).
type Tier int

// Valid reports whether t is between 0 and 4.
func (t Tier) Valid() bool { return t >= 0 && t <= 4 }

// LineCap returns the maximum changed lines for a tier.
// Tier 0 (project CI) 200, Tier 1 30, Tier 2 200, Tier 3 and 4 400.
func LineCap(t Tier) int {
	switch t {
	case 0:
		return 200
	case 1:
		return 30
	case 2:
		return 200
	case 3, 4:
		return 400
	default:
		return 0
	}
}

// Model families. Writer, Reviewer and Adjudicator must come from different families.
const (
	FamilyAnthropic = "anthropic"
	FamilyOpenAI    = "openai"
	FamilyXAI       = "xai"
	FamilyCursor    = "cursor"
)

var knownProviders = []string{FamilyAnthropic, FamilyOpenAI, FamilyXAI, FamilyCursor}

// ParseModelRef splits "provider:name" and checks the provider is known.
func ParseModelRef(ref string) (provider, name string, err error) {
	provider, name, ok := strings.Cut(ref, ":")
	if !ok || provider == "" || name == "" {
		return "", "", fmt.Errorf("model ref %q must look like provider:name", ref)
	}
	if !slices.Contains(knownProviders, provider) {
		return "", "", fmt.Errorf("model ref %q has unknown provider (allowed %v)", ref, knownProviders)
	}
	return provider, name, nil
}

// Family returns the model family of a model ref, or "" if the ref is invalid.
func Family(ref string) string {
	p, _, err := ParseModelRef(ref)
	if err != nil {
		return ""
	}
	return p
}

// IsCIPath reports whether a repo-relative path is project CI or build/deploy
// configuration. These paths are Tier 0 only and may never appear in the
// scope of a Tier 1-4 WorkTicket (ARCHITECTURE.md Sections 8 and 11.4).
func IsCIPath(p string) bool {
	p = strings.TrimPrefix(path.Clean(p), "./")
	base := path.Base(p)
	switch {
	case strings.HasPrefix(p, ".github/"),
		strings.HasPrefix(p, ".argus/"),
		strings.HasPrefix(p, ".devcontainer/"),
		strings.HasPrefix(p, ".gitlab-ci"),
		strings.HasPrefix(base, "Dockerfile"),
		strings.HasSuffix(base, ".dockerfile"),
		base == "docker-compose.yml", base == "docker-compose.yaml",
		base == "compose.yml", base == "compose.yaml",
		base == ".dockerignore":
		return true
	}
	return false
}

// safeRelPath reports whether p is a clean, repo-relative path that stays inside the repo.
func safeRelPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") {
		return false
	}
	clean := path.Clean(p)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && clean == p
}

func in01(x float64) bool { return x >= 0 && x <= 1 }
