package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/argus-dev/argus/internal/contracts"
)

const validRoot = "testdata/valid"

// copyValid copies the valid fixture tree into a temp dir so a test can break one file.
func copyValid(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(validRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(validRoot, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// edit replaces old with new in root/file; it fails the test if old is not found.
func edit(t *testing.T, root, file, old, new string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(file))
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), old) {
		t.Fatalf("%s does not contain %q", file, old)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(b), old, new, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadValid(t *testing.T) {
	c, err := Load(validRoot)
	if err != nil {
		t.Fatalf("valid fixtures rejected: %v", err)
	}
	if n := len(c.Models()); n != 9 {
		t.Fatalf("models = %d, want 9", n)
	}
	m, ok := c.Model("xai:grok-4.7")
	if !ok || m.Class != "top" || m.Family() != contracts.FamilyXAI {
		t.Fatalf("model lookup: %+v %v", m, ok)
	}
	p, ok := c.Price("anthropic:claude-opus-5.5")
	if !ok || p.InputPerMTok != 4 || p.CachedInput() != 0.4 {
		t.Fatalf("price: %+v %v", p, ok)
	}
	if p, _ := c.Price("anthropic:claude-sonnet-5.5"); p.CachedInput() != 2 {
		t.Fatalf("cached input defaults to input price, got %v", p.CachedInput())
	}
	r, ok := c.Route(2)
	if !ok || r.Writer != "anthropic:claude-sonnet-5.5" || r.Reviewer != "openai:gpt-6.1-sol" {
		t.Fatalf("route: %+v %v", r, ok)
	}
	if b := c.Budgets(); b.OrgMonthlyUSD != 600 || b.PerTicketUSD[4] != 25 {
		t.Fatalf("budgets: %+v", b)
	}

	pol := c.Policy()
	if !slices.Equal(pol.HumanReviewTiers(), []contracts.Tier{0, 4}) {
		t.Fatalf("human review tiers: %v", pol.HumanReviewTiers())
	}
	for tier, want := range map[contracts.Tier]bool{0: true, 1: false, 3: false, 4: true} {
		if got := pol.HumanReviewRequired(tier); got != want {
			t.Errorf("tier %d human review = %v, want %v", tier, got, want)
		}
	}
	if pol.Tier0Writer() != "anthropic:claude-opus-5.5" {
		t.Fatalf("tier0 writer: %s", pol.Tier0Writer())
	}
	if tp, _ := pol.Tier(1); tp.LineCap != 30 || tp.Floor != "small" {
		t.Fatalf("tier 1 policy: %+v", tp)
	}
	if got := pol.Allowed(3, "red_team"); !slices.Equal(got, []string{"xai:grok-4.7"}) {
		t.Fatalf("allowed: %v", got)
	}
}

// Policy is read-only at runtime: callers get copies, never the loaded data.
func TestPolicyReadOnly(t *testing.T) {
	c, err := Load(validRoot)
	if err != nil {
		t.Fatal(err)
	}
	pol := c.Policy()
	pol.HumanReviewTiers()[0] = 3
	pol.Allowed(2, "writer")[0] = "xai:evil"
	c.Budgets().PerTicketUSD[1] = 999
	c.Models()[0].Ref = "xai:evil"
	if !slices.Equal(c.Policy().HumanReviewTiers(), []contracts.Tier{0, 4}) ||
		c.Policy().Allowed(2, "writer")[0] != "anthropic:claude-sonnet-5.5" ||
		c.Budgets().PerTicketUSD[1] != 0.5 ||
		c.Models()[0].Ref != "anthropic:claude-opus-5.5" {
		t.Fatal("loaded config was changed through a returned value")
	}
}

func TestLoadRejects(t *testing.T) {
	const (
		models  = "config/models.yaml"
		pricing = "config/pricing.yaml"
		routing = "config/routing.yaml"
		budgets = "config/budgets.yaml"
		tiers   = "policy/tiers.yaml"
		allow   = "policy/allowlist.yaml"
	)
	cases := []struct {
		name, file, old, new, want string
	}{
		// Decoding.
		{"unknown field", models, "class: top, backends", "class: top, colour: red, backends", "field colour not found"},
		{"bad yaml", budgets, "alert_at: 0.8", "alert_at: [", "config/budgets.yaml"},
		{"empty file", budgets, "org_monthly_usd: 600\nalert_at: 0.8\nper_ticket_usd: {0: 8, 1: 0.5, 2: 4, 3: 12, 4: 25}\n", "", "config/budgets.yaml: file is empty"},
		{"two documents", budgets, "alert_at: 0.8", "alert_at: 0.8\n---\nalert_at: 0.9", "only one YAML document"},

		// models.yaml
		{"bad model ref", models, `"xai:grok-4.6"`, `"grok-4.6"`, "provider:name"},
		{"duplicate model", models, `"xai:grok-4.6"`, `"xai:grok-4.7"`, "listed twice"},
		{"bad class", models, "name: Grok 4.6, class: mid", "name: Grok 4.6, class: huge", `class "huge"`},
		{"cursor model on api", models, "class: small, backends: [cursor]", "class: small, backends: [api]", "cursor models run only on the cursor backend"},
		{"no backends", models, "name: Grok 4.6, class: mid, backends: [api]", "name: Grok 4.6, class: mid, backends: []", "backends must not be empty"},

		// pricing.yaml
		{"missing price", pricing, `  "xai:grok-4.6": {input_per_mtok: 2, output_per_mtok: 6}` + "\n", "", `no price for model "xai:grok-4.6"`},
		{"price for unknown model", pricing, `"xai:grok-4.6"`, `"xai:grok-9"`, `price for unknown model "xai:grok-9"`},
		{"zero price", pricing, `"xai:grok-4.6": {input_per_mtok: 2`, `"xai:grok-4.6": {input_per_mtok: 0`, "must be > 0"},
		{"cached above input", pricing, "cached_input_per_mtok: 0.4", "cached_input_per_mtok: 5", "cached_input_per_mtok"},
		{"wrong currency", pricing, "currency: USD", "currency: EUR", "currency"},

		// policy/tiers.yaml
		{"cap differs from contract", tiers, "1: {line_cap: 30", "1: {line_cap: 50", "must equal the contract cap 30"},
		{"missing tier", tiers, "  4: {line_cap: 400, floor: top, max_attempts_per_model: 2, max_escalations: 1}\n", "", "tier 4 is missing"},
		{"bad floor", tiers, "2: {line_cap: 200, floor: mid", "2: {line_cap: 200, floor: any", `floor "any"`},
		{"human review drops tier 4", tiers, "human_review_tiers: [0, 4]", "human_review_tiers: [0]", "must include tiers 0 and 4"},
		{"human review adds tier 3", tiers, "human_review_tiers: [0, 4]", "human_review_tiers: [0, 3, 4]", ""},
		{"human review bad tier", tiers, "human_review_tiers: [0, 4]", "human_review_tiers: [0, 4, 9]", "tier 9 must be 0..4"},
		{"zero attempts", tiers, "3: {line_cap: 400, floor: top, max_attempts_per_model: 2", "3: {line_cap: 400, floor: top, max_attempts_per_model: 0", "max_attempts_per_model must be >= 1"},

		// policy/allowlist.yaml
		{"allowlist unknown model", allow, `reviewer: ["openai:gpt-6.1-sol", "anthropic:claude-sonnet-5.5"]`, `reviewer: ["openai:gpt-6.1-sol", "openai:gpt-7"]`, `unknown model "openai:gpt-7"`},
		{"below tier floor", allow, `writer: ["anthropic:claude-sonnet-5.5", "openai:gpt-6.1-sol", "xai:grok-4.6"]`, `writer: ["anthropic:claude-haiku-5.5"]`, "below the tier 2 floor mid"},
		{"tier 0 alternates", allow, `writer: ["anthropic:claude-opus-5.5"]`, `writer: ["anthropic:claude-opus-5.5", "openai:gpt-6-astra"]`, "tier 0 writer must be only tier0_writer"},
		{"no reviewer", allow, `    reviewer: ["openai:gpt-6.1-sol", "anthropic:claude-sonnet-5.5"]` + "\n", "", "tier 2 needs at least one reviewer"},
		{"unknown function", allow, "red_team:", "cheerleader:", `function "cheerleader"`},
		{"no cross-family reviewer", allow, `reviewer: ["openai:gpt-6.1-sol", "anthropic:claude-sonnet-5.5"]`, `reviewer: ["anthropic:claude-sonnet-5.5"]`, "has no reviewer from another family"},

		// config/routing.yaml
		{"route not allowlisted", routing, `2: {writer: "anthropic:claude-sonnet-5.5"`, `2: {writer: "anthropic:claude-opus-5.5"`, "not allowed as writer at tier 2"},
		{"route same family", routing, `reviewer: "openai:gpt-6.1-sol"`, `reviewer: "anthropic:claude-sonnet-5.5"`, "same family"},
		{"exploration at tier 3", routing, `red_team: "xai:grok-4.7", lambda: 0.1, exploration: 0`, `red_team: "xai:grok-4.7", lambda: 0.1, exploration: 0.1`, "exploration must be 0 at tier 3"},
		{"exploration too high", routing, "lambda: 0.5, exploration: 0.1", "lambda: 0.5, exploration: 0.5", "at most 0.1"},
		{"lambda at tier 4", routing, `4: {writer: "anthropic:claude-opus-5.5", reviewer: "openai:gpt-6-astra", red_team: "xai:grok-4.7", lambda: 0`, `4: {writer: "anthropic:claude-opus-5.5", reviewer: "openai:gpt-6-astra", red_team: "xai:grok-4.7", lambda: 1`, "lambda must be 0 at tier 4"},

		// config/budgets.yaml
		{"alert out of range", budgets, "alert_at: 0.8", "alert_at: 1.5", "alert_at"},
		{"ticket cap over org cap", budgets, "4: 25}", "4: 2500}", "exceeds org_monthly_usd"},
		{"missing ticket cap", budgets, ", 4: 25}", "}", "per_ticket_usd: tier 4 is missing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := copyValid(t)
			edit(t, root, tc.file, tc.old, tc.new)
			_, err := Load(root)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected valid, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	root := copyValid(t)
	if err := os.Remove(filepath.Join(root, "policy", "allowlist.yaml")); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	var ce *Error
	if !errors.As(err, &ce) || !strings.Contains(err.Error(), "policy/allowlist.yaml") {
		t.Fatalf("want *Error naming policy/allowlist.yaml, got %v", err)
	}
}

// Every problem is reported at once, not only the first.
func TestLoadReportsAllProblems(t *testing.T) {
	root := copyValid(t)
	edit(t, root, "config/budgets.yaml", "alert_at: 0.8", "alert_at: 7")
	edit(t, root, "policy/tiers.yaml", "1: {line_cap: 30", "1: {line_cap: 31")
	_, err := Load(root)
	var ce *Error
	if !errors.As(err, &ce) || len(ce.Problems) != 2 {
		t.Fatalf("want 2 problems, got %v", err)
	}
}

// TestRepoConfigValid loads the real config/ and policy/ at the repo root,
// so CI fails if a human edit breaks them.
func TestRepoConfigValid(t *testing.T) {
	c, err := Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pol := c.Policy()
	for tier := contracts.Tier(0); tier <= 4; tier++ {
		for _, fn := range []string{"writer", "reviewer", "red_team"} {
			for _, ref := range pol.Allowed(tier, fn) {
				if _, ok := c.Price(ref); !ok {
					t.Errorf("tier %d %s %s has no price", tier, fn, ref)
				}
			}
		}
	}
}
