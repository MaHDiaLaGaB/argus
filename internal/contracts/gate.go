package contracts

import (
	"slices"
	"sort"
)

// RequiredChecks are the pipeline CI gates every PR must pass (ARCHITECTURE.md 11.4).
var RequiredChecks = []string{"scope", "line_cap", "lint_tests", "test_proof", "gitleaks", "dep_audit", "dup_check"}

// DefaultHumanReviewTiers is the default for policy.human_review_tiers.
var DefaultHumanReviewTiers = []Tier{0, 4}

// GateResult is the tier-gate decision for a PR (ARCHITECTURE.md 5.7).
// State is decided by Evaluate, which is pure code: no model is involved.
type GateResult struct {
	SchemaVersion       string            `json:"schema_version"`
	PR                  string            `json:"pr"`
	Tier                Tier              `json:"tier"`
	CIChecks            map[string]string `json:"ci_checks"`
	AIReview            string            `json:"ai_review"`
	SecurityReview      string            `json:"security_review"`
	HumanReviewRequired bool              `json:"human_review_required"`
	HumanReview         string            `json:"human_review"`
	CISuspect           bool              `json:"ci_suspect"`
	State               string            `json:"state"`
	BlockedReasons      []string          `json:"blocked_reasons"`
}

// HumanReviewRequiredFor reports whether a tier needs human review under the given policy.
func HumanReviewRequiredFor(t Tier, policyTiers []Tier) bool {
	return slices.Contains(policyTiers, t)
}

// Evaluate decides whether the PR is ready to merge. It never looks at State.
func (g *GateResult) Evaluate() (state string, reasons []string) {
	names := make([]string, 0, len(g.CIChecks))
	for n := range g.CIChecks {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, req := range RequiredChecks {
		v, ok := g.CIChecks[req]
		switch {
		case !ok:
			reasons = append(reasons, "missing CI check: "+req)
		case v == "pass":
		case v == "not_required" && req == "test_proof" && g.Tier == 0:
		default:
			reasons = append(reasons, "CI check "+req+": "+v)
		}
	}
	for _, n := range names {
		if !slices.Contains(RequiredChecks, n) && g.CIChecks[n] != "pass" {
			reasons = append(reasons, "CI check "+n+": "+g.CIChecks[n])
		}
	}
	if g.CISuspect {
		reasons = append(reasons, "ci-suspect: waiting for a human to check CI")
	}
	if g.AIReview != "pass" {
		reasons = append(reasons, "AI review: "+g.AIReview)
	}
	if g.Tier >= 2 && g.SecurityReview != "pass" {
		reasons = append(reasons, "security review: "+g.SecurityReview)
	}
	if g.SecurityReview == "veto" {
		reasons = append(reasons, "security veto")
	}
	if g.HumanReviewRequired && g.HumanReview != "approved" {
		reasons = append(reasons, "human review: "+g.HumanReview)
	}
	if len(reasons) == 0 {
		return "ready_to_merge", nil
	}
	return "blocked", reasons
}

// Validate checks enums and that State and BlockedReasons match Evaluate.
func (g *GateResult) Validate() error {
	c := newChecker("GateResult")
	c.schema(g.SchemaVersion)
	c.required(g.PR, "pr")
	c.check(g.Tier.Valid(), "tier %d must be 0..4", g.Tier)
	for n, v := range g.CIChecks {
		c.oneOf(v, "ci_checks."+n, "pass", "fail", "pending", "error", "not_required")
	}
	c.oneOf(g.AIReview, "ai_review", "pass", "changes_required", "block", "pending")
	c.oneOf(g.SecurityReview, "security_review", "pass", "changes_required", "veto", "pending", "not_required")
	c.oneOf(g.HumanReview, "human_review", "approved", "changes_requested", "pending", "not_required")
	if !g.HumanReviewRequired {
		c.check(g.HumanReview == "not_required" || g.HumanReview == "approved",
			"human_review %q given but not required for this tier", g.HumanReview)
	}
	if g.Tier >= 2 {
		c.check(g.SecurityReview != "not_required", "security review is required from Tier 2")
	}
	c.oneOf(g.State, "state", "ready_to_merge", "blocked")
	state, reasons := g.Evaluate()
	c.check(g.State == state, "state %q does not match evaluated state %q", g.State, state)
	c.check(slices.Equal(g.BlockedReasons, reasons) || (len(g.BlockedReasons) == 0 && len(reasons) == 0),
		"blocked_reasons do not match evaluated reasons %v", reasons)
	return c.err()
}
