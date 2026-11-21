// Package config loads and validates config/*.yaml and policy/*.yaml at startup.
//
// Load reads every file strictly (unknown fields are rejected), checks each
// file and the rules between files, and returns every problem at once. A
// service must refuse to start when Load returns an error.
//
// Config and Policy are read-only after loading: their fields are unexported
// and every accessor returns a copy. Policy is changed only by humans editing
// policy/ (AGENTS.md rule 10), never at runtime.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/argus-dev/argus/internal/contracts"
)

// Files read by Load, relative to the root directory.
const (
	ModelsFile    = "config/models.yaml"
	PricingFile   = "config/pricing.yaml"
	RoutingFile   = "config/routing.yaml"
	BudgetsFile   = "config/budgets.yaml"
	TiersFile     = "policy/tiers.yaml"
	AllowlistFile = "policy/allowlist.yaml"
)

// Model classes, weakest first. A tier's floor is the weakest class it may use.
var classes = []string{"small", "mid", "top"}

// Functions a model can be allowed for in policy/allowlist.yaml.
var functions = []string{"writer", "reviewer", "red_team", "adjudicator"}

// Model is one entry in config/models.yaml.
type Model struct {
	Ref      string   `yaml:"ref"`
	Name     string   `yaml:"name"`
	Class    string   `yaml:"class"`
	Backends []string `yaml:"backends"`
}

// Family returns the model family (provider) of the model.
func (m Model) Family() string { return contracts.Family(m.Ref) }

// Price is USD per million tokens for one model (config/pricing.yaml).
type Price struct {
	InputPerMTok       float64  `yaml:"input_per_mtok"`
	OutputPerMTok      float64  `yaml:"output_per_mtok"`
	CachedInputPerMTok *float64 `yaml:"cached_input_per_mtok"`
}

// CachedInput is the cached-input price, or the input price when none is set.
func (p Price) CachedInput() float64 {
	if p.CachedInputPerMTok == nil {
		return p.InputPerMTok
	}
	return *p.CachedInputPerMTok
}

// Route is the static model choice for one tier (config/routing.yaml).
type Route struct {
	Writer      string  `yaml:"writer"`
	Reviewer    string  `yaml:"reviewer"`
	RedTeam     string  `yaml:"red_team"`
	Lambda      float64 `yaml:"lambda"`
	Exploration float64 `yaml:"exploration"`
}

// Budgets are the spending caps in USD (config/budgets.yaml).
type Budgets struct {
	OrgMonthlyUSD float64                    `yaml:"org_monthly_usd"`
	AlertAt       float64                    `yaml:"alert_at"`
	PerTicketUSD  map[contracts.Tier]float64 `yaml:"per_ticket_usd"`
}

// TierPolicy is the policy for one tier (policy/tiers.yaml).
type TierPolicy struct {
	LineCap             int    `yaml:"line_cap"`
	Floor               string `yaml:"floor"`
	MaxAttemptsPerModel int    `yaml:"max_attempts_per_model"`
	MaxEscalations      int    `yaml:"max_escalations"`
}

type modelsFile struct {
	Models []Model `yaml:"models"`
}

type pricingFile struct {
	Currency string           `yaml:"currency"`
	Prices   map[string]Price `yaml:"prices"`
}

type routingFile struct {
	Tiers map[contracts.Tier]Route `yaml:"tiers"`
}

type tiersFile struct {
	HumanReviewTiers []contracts.Tier              `yaml:"human_review_tiers"`
	Tiers            map[contracts.Tier]TierPolicy `yaml:"tiers"`
}

type allowlistFile struct {
	Tier0Writer string                                 `yaml:"tier0_writer"`
	Tiers       map[contracts.Tier]map[string][]string `yaml:"tiers"`
}

// Config is the loaded, validated configuration. It is read-only.
type Config struct {
	models  []Model
	byRef   map[string]Model
	pricing pricingFile
	routing routingFile
	budgets Budgets
	policy  Policy
}

// Policy is the loaded, validated policy. It is read-only.
type Policy struct {
	tiers tiersFile
	allow allowlistFile
}

// Error lists every problem found while loading, each prefixed with its file.
type Error struct {
	Problems []string
}

func (e *Error) Error() string {
	return "invalid config:\n  " + strings.Join(e.Problems, "\n  ")
}

// Load reads and validates every config and policy file under root.
func Load(root string) (*Config, error) {
	e := &Error{}
	var (
		mf modelsFile
		pf pricingFile
		rf routingFile
		bf Budgets
		tf tiersFile
		af allowlistFile
	)
	ok := decodeFile(e, root, ModelsFile, &mf)
	ok = decodeFile(e, root, PricingFile, &pf) && ok
	ok = decodeFile(e, root, RoutingFile, &rf) && ok
	ok = decodeFile(e, root, BudgetsFile, &bf) && ok
	ok = decodeFile(e, root, TiersFile, &tf) && ok
	ok = decodeFile(e, root, AllowlistFile, &af) && ok
	if !ok {
		return nil, e // cross-file rules need every file
	}

	c := &Config{models: mf.Models, byRef: map[string]Model{}, pricing: pf, routing: rf, budgets: bf,
		policy: Policy{tiers: tf, allow: af}}
	c.checkModels(e)
	c.checkPricing(e)
	c.checkTiers(e)
	c.checkAllowlist(e)
	c.checkRouting(e)
	c.checkBudgets(e)
	if len(e.Problems) > 0 {
		return nil, e
	}
	return c, nil
}

// decodeFile strictly decodes one YAML document from root/name into v.
func decodeFile(e *Error, root, name string, v any) bool {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		e.add(name, "%v", err)
		return false
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	switch err := dec.Decode(v); {
	case errors.Is(err, io.EOF):
		e.add(name, "file is empty")
		return false
	case err != nil:
		e.add(name, "%v", err)
		return false
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		e.add(name, "must contain only one YAML document")
		return false
	}
	return true
}

func (e *Error) add(file, format string, args ...any) {
	e.Problems = append(e.Problems, file+": "+fmt.Sprintf(format, args...))
}

// check adds a problem when ok is false.
func (e *Error) check(ok bool, file, format string, args ...any) {
	if !ok {
		e.add(file, format, args...)
	}
}

// allTiers checks that a tier-keyed map has exactly tiers 0..4.
func allTiers[V any](e *Error, file, field string, m map[contracts.Tier]V) {
	for t := contracts.Tier(0); t <= 4; t++ {
		_, ok := m[t]
		e.check(ok, file, "%s: tier %d is missing", field, t)
	}
	for t := range m {
		e.check(t.Valid(), file, "%s: tier %d must be 0..4", field, t)
	}
}

func classRank(c string) int { return slices.Index(classes, c) }

func (c *Config) checkModels(e *Error) {
	e.check(len(c.models) > 0, ModelsFile, "models must not be empty")
	for i, m := range c.models {
		at := fmt.Sprintf("models[%d]", i)
		_, _, err := contracts.ParseModelRef(m.Ref)
		e.check(err == nil, ModelsFile, "%s.ref: %v", at, err)
		_, dup := c.byRef[m.Ref]
		e.check(!dup, ModelsFile, "%s.ref %q listed twice", at, m.Ref)
		c.byRef[m.Ref] = m
		e.check(strings.TrimSpace(m.Name) != "", ModelsFile, "%s.name is required", at)
		e.check(classRank(m.Class) >= 0, ModelsFile, "%s.class %q must be one of %v", at, m.Class, classes)
		e.check(len(m.Backends) > 0, ModelsFile, "%s.backends must not be empty", at)
		for _, b := range m.Backends {
			e.check(b == "api" || b == "cursor", ModelsFile, "%s.backends: %q must be api or cursor", at, b)
		}
		if m.Family() == contracts.FamilyCursor {
			e.check(slices.Equal(m.Backends, []string{"cursor"}), ModelsFile,
				"%s: cursor models run only on the cursor backend", at)
		}
	}
}

func (c *Config) checkPricing(e *Error) {
	e.check(c.pricing.Currency == "USD", PricingFile, "currency %q must be USD", c.pricing.Currency)
	for _, m := range c.models {
		_, ok := c.pricing.Prices[m.Ref]
		e.check(ok, PricingFile, "no price for model %q", m.Ref)
	}
	for _, ref := range slices.Sorted(maps.Keys(c.pricing.Prices)) {
		p := c.pricing.Prices[ref]
		_, known := c.byRef[ref]
		e.check(known, PricingFile, "price for unknown model %q", ref)
		e.check(p.InputPerMTok > 0 && p.OutputPerMTok > 0, PricingFile,
			"%s: input_per_mtok and output_per_mtok must be > 0", ref)
		if p.CachedInputPerMTok != nil {
			e.check(*p.CachedInputPerMTok >= 0 && *p.CachedInputPerMTok <= p.InputPerMTok, PricingFile,
				"%s: cached_input_per_mtok must be between 0 and input_per_mtok", ref)
		}
	}
}

func (c *Config) checkTiers(e *Error) {
	tf := c.policy.tiers
	allTiers(e, TiersFile, "tiers", tf.Tiers)
	for _, t := range slices.Sorted(maps.Keys(tf.Tiers)) {
		tp := tf.Tiers[t]
		if t.Valid() {
			// Caps are enforced by the contracts; policy must agree (changing them needs a schema bump).
			e.check(tp.LineCap == contracts.LineCap(t), TiersFile,
				"tier %d: line_cap %d must equal the contract cap %d", t, tp.LineCap, contracts.LineCap(t))
		}
		e.check(classRank(tp.Floor) >= 0, TiersFile, "tier %d: floor %q must be one of %v", t, tp.Floor, classes)
		e.check(tp.MaxAttemptsPerModel >= 1, TiersFile, "tier %d: max_attempts_per_model must be >= 1", t)
		e.check(tp.MaxEscalations >= 0, TiersFile, "tier %d: max_escalations must be >= 0", t)
	}
	seen := map[contracts.Tier]bool{}
	for _, t := range tf.HumanReviewTiers {
		e.check(t.Valid(), TiersFile, "human_review_tiers: tier %d must be 0..4", t)
		e.check(!seen[t], TiersFile, "human_review_tiers: tier %d listed twice", t)
		seen[t] = true
	}
	// ARCHITECTURE.md Section 2: Tier 0 and Tier 4 always need a human. Others may be added.
	e.check(seen[0] && seen[4], TiersFile, "human_review_tiers must include tiers 0 and 4")
}

func (c *Config) checkAllowlist(e *Error) {
	af := c.policy.allow
	allTiers(e, AllowlistFile, "tiers", af.Tiers)
	_, ok := c.byRef[af.Tier0Writer]
	e.check(ok, AllowlistFile, "tier0_writer: unknown model %q", af.Tier0Writer)
	for _, t := range slices.Sorted(maps.Keys(af.Tiers)) {
		fns := af.Tiers[t]
		floor := ""
		if tp, ok := c.policy.tiers.Tiers[t]; ok {
			floor = tp.Floor
		}
		for _, fn := range slices.Sorted(maps.Keys(fns)) {
			e.check(slices.Contains(functions, fn), AllowlistFile, "tier %d: function %q must be one of %v", t, fn, functions)
			for _, ref := range fns[fn] {
				m, ok := c.byRef[ref]
				e.check(ok, AllowlistFile, "tier %d %s: unknown model %q", t, fn, ref)
				if ok && classRank(floor) >= 0 {
					e.check(classRank(m.Class) >= classRank(floor), AllowlistFile,
						"tier %d %s: %q (%s) is below the tier %d floor %s", t, fn, ref, m.Class, t, floor)
				}
			}
		}
		e.check(len(fns["writer"]) > 0, AllowlistFile, "tier %d needs at least one writer", t)
		e.check(len(fns["reviewer"]) > 0, AllowlistFile, "tier %d needs at least one reviewer", t)
		if t == 0 {
			e.check(slices.Equal(fns["writer"], []string{af.Tier0Writer}), AllowlistFile,
				"tier 0 writer must be only tier0_writer %q (no cheaper alternates)", af.Tier0Writer)
		}
		if len(fns["reviewer"]) == 0 {
			continue
		}
		for _, w := range fns["writer"] {
			cross := slices.ContainsFunc(fns["reviewer"], func(r string) bool { return contracts.Family(r) != contracts.Family(w) })
			e.check(cross, AllowlistFile, "tier %d: writer %q has no reviewer from another family", t, w)
		}
	}
}

func (c *Config) checkRouting(e *Error) {
	allTiers(e, RoutingFile, "tiers", c.routing.Tiers)
	for _, t := range slices.Sorted(maps.Keys(c.routing.Tiers)) {
		r := c.routing.Tiers[t]
		c.checkRouteModel(e, t, "writer", r.Writer, "")
		c.checkRouteModel(e, t, "reviewer", r.Reviewer, r.Writer)
		if r.RedTeam != "" {
			c.checkRouteModel(e, t, "red_team", r.RedTeam, r.Writer)
		}
		if t == 0 {
			e.check(r.Writer == c.policy.allow.Tier0Writer, RoutingFile, "tier 0: writer must be tier0_writer")
		}
		e.check(r.Lambda >= 0, RoutingFile, "tier %d: lambda must be >= 0", t)
		e.check(t != 4 || r.Lambda == 0, RoutingFile, "tier %d: lambda must be 0 at tier 4 (cost is ignored)", t)
		e.check(r.Exploration >= 0 && r.Exploration <= 0.1, RoutingFile, "tier %d: exploration must be between 0 and at most 0.1", t)
		if t == 0 || t == 3 || t == 4 {
			e.check(r.Exploration == 0, RoutingFile, "tier %d: exploration must be 0 at tier %d", t, t)
		}
	}
}

// checkRouteModel checks a routed model is allowlisted for (tier, fn) and, when
// writer is set, comes from a different family than the writer.
func (c *Config) checkRouteModel(e *Error, t contracts.Tier, fn, ref, writer string) {
	e.check(slices.Contains(c.policy.allow.Tiers[t][fn], ref), RoutingFile,
		"tier %d: %q is not allowed as %s at tier %d", t, ref, fn, t)
	if writer != "" {
		e.check(contracts.Family(ref) != contracts.Family(writer), RoutingFile,
			"tier %d: %s %q is the same family as writer %q", t, fn, ref, writer)
	}
}

func (c *Config) checkBudgets(e *Error) {
	b := c.budgets
	e.check(b.OrgMonthlyUSD > 0, BudgetsFile, "org_monthly_usd must be > 0")
	e.check(b.AlertAt > 0 && b.AlertAt < 1, BudgetsFile, "alert_at %v must be between 0 and 1", b.AlertAt)
	allTiers(e, BudgetsFile, "per_ticket_usd", b.PerTicketUSD)
	for _, t := range slices.Sorted(maps.Keys(b.PerTicketUSD)) {
		v := b.PerTicketUSD[t]
		e.check(v > 0, BudgetsFile, "per_ticket_usd: tier %d must be > 0", t)
		e.check(v <= b.OrgMonthlyUSD, BudgetsFile, "per_ticket_usd: tier %d cap %v exceeds org_monthly_usd", t, v)
	}
}

// Models returns a copy of every configured model.
func (c *Config) Models() []Model {
	out := make([]Model, len(c.models))
	for i, m := range c.models {
		m.Backends = slices.Clone(m.Backends)
		out[i] = m
	}
	return out
}

// Model looks up a model by ref.
func (c *Config) Model(ref string) (Model, bool) {
	m, ok := c.byRef[ref]
	m.Backends = slices.Clone(m.Backends)
	return m, ok
}

// Price looks up a model's price by ref.
func (c *Config) Price(ref string) (Price, bool) {
	p, ok := c.pricing.Prices[ref]
	if p.CachedInputPerMTok != nil {
		v := *p.CachedInputPerMTok
		p.CachedInputPerMTok = &v
	}
	return p, ok
}

// Route returns the static route for a tier.
func (c *Config) Route(t contracts.Tier) (Route, bool) {
	r, ok := c.routing.Tiers[t]
	return r, ok
}

// Budgets returns a copy of the budget caps.
func (c *Config) Budgets() Budgets {
	b := c.budgets
	b.PerTicketUSD = maps.Clone(b.PerTicketUSD)
	return b
}

// Policy returns the read-only policy.
func (c *Config) Policy() Policy { return c.policy }

// HumanReviewTiers returns a copy of policy.human_review_tiers.
func (p Policy) HumanReviewTiers() []contracts.Tier { return slices.Clone(p.tiers.HumanReviewTiers) }

// HumanReviewRequired reports whether a tier needs human review.
func (p Policy) HumanReviewRequired(t contracts.Tier) bool {
	return contracts.HumanReviewRequiredFor(t, p.tiers.HumanReviewTiers)
}

// Tier returns the policy for one tier.
func (p Policy) Tier(t contracts.Tier) (TierPolicy, bool) {
	tp, ok := p.tiers.Tiers[t]
	return tp, ok
}

// Tier0Writer is the single model allowed to write Tier 0 (project CI) changes.
func (p Policy) Tier0Writer() string { return p.allow.Tier0Writer }

// Allowed returns a copy of the models allowed for (tier, function).
func (p Policy) Allowed(t contracts.Tier, function string) []string {
	return slices.Clone(p.allow.Tiers[t][function])
}
