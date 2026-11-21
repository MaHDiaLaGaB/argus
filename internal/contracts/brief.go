package contracts

// MergeBrief tells the human who merges what changed and how to merge (ARCHITECTURE.md 5.8).
// It is filled by code from data; a cheap model writes only OneLine and RiskNotes.
type MergeBrief struct {
	SchemaVersion     string   `json:"schema_version"`
	PR                string   `json:"pr"`
	Tier              Tier     `json:"tier"`
	OneLine           string   `json:"one_line"`
	WhatChanged       []string `json:"what_changed"`
	Evidence          []string `json:"evidence"`
	Risk              string   `json:"risk"`
	RiskNotes         []string `json:"risk_notes"`
	MergeInstructions []string `json:"merge_instructions"`
	AfterMerge        []string `json:"after_merge"`
	Rollback          string   `json:"rollback"`
	HumanChecks       []string `json:"human_checks"`
}

// Validate checks a MergeBrief.
func (b *MergeBrief) Validate() error {
	c := newChecker("MergeBrief")
	c.schema(b.SchemaVersion)
	c.required(b.PR, "pr")
	c.check(b.Tier.Valid(), "tier %d must be 0..4", b.Tier)
	c.required(b.OneLine, "one_line")
	c.check(len(b.OneLine) <= 140, "one_line must be at most 140 characters")
	c.check(len(b.WhatChanged) > 0, "what_changed must not be empty")
	c.check(len(b.Evidence) > 0, "evidence must not be empty")
	c.oneOf(b.Risk, "risk", "low", "medium", "high")
	if b.Risk != "low" {
		c.check(len(b.RiskNotes) > 0, "risk_notes are required when risk is %s", b.Risk)
	}
	c.check(len(b.MergeInstructions) > 0, "merge_instructions must not be empty")
	c.required(b.Rollback, "rollback")
	if HumanReviewRequiredFor(b.Tier, DefaultHumanReviewTiers) {
		c.check(len(b.HumanChecks) > 0, "human_checks are required for Tier %d", b.Tier)
	}
	return c.err()
}
