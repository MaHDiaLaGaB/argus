package contracts

import "strings"

// TaskEnvelope is what the Orchestrator sends to a track agent (ARCHITECTURE.md 5.4).
type TaskEnvelope struct {
	SchemaVersion string        `json:"schema_version"`
	RunID         string        `json:"run_id"`
	WorkID        string        `json:"work_id"`
	Role          string        `json:"role"`
	Track         string        `json:"track"`
	Tier          Tier          `json:"tier"`
	Attempt       int           `json:"attempt"`
	Model         ModelChoice   `json:"model"`
	ContextPacket ContextPacket `json:"context_packet"`
	Workspace     Workspace     `json:"workspace"`
	Permissions   []string      `json:"permissions"`
	Budget        Budget        `json:"budget"`
}

// ModelChoice is the model picked by the router, with fallbacks in the same tier floor.
type ModelChoice struct {
	Ref       string   `json:"ref"`
	Backend   string   `json:"backend"`
	Fallbacks []string `json:"fallbacks"`
}

// ContextPacket is the curated context built once by the Orchestrator.
type ContextPacket struct {
	Outcome            string         `json:"outcome"`
	Contract           string         `json:"contract"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
	ScopeFiles         []string       `json:"scope_files"`
	LineCap            int            `json:"line_cap"`
	ProofTest          string         `json:"proof_test"`
	RulesRefs          []string       `json:"rules_refs"`
	FileSummaries      []FileSummary  `json:"file_summaries"`
	ReuseCandidates    []ReuseRef     `json:"reuse_candidates"`
	PriorAttempts      []PriorAttempt `json:"prior_attempts"`
	Skills             []SkillRef     `json:"skills"`
	MemorySnippets     []string       `json:"memory_snippets"`
}

// FileSummary is a cached summary of one file.
type FileSummary struct {
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

// ReuseRef points at existing code the Writer should consider reusing.
type ReuseRef struct {
	Repo   string `json:"repo"`
	Symbol string `json:"symbol"`
	Path   string `json:"path"`
	Why    string `json:"why"`
}

// PriorAttempt records why an earlier attempt failed.
type PriorAttempt struct {
	Attempt int    `json:"attempt"`
	Model   string `json:"model"`
	Failure string `json:"failure"`
}

// SkillRef is a versioned skill included in the context.
type SkillRef struct {
	SkillID string `json:"skill_id"`
	Version int    `json:"version"`
}

// Workspace is where the agent works.
type Workspace struct {
	SandboxID string `json:"sandbox_id"`
	Base      string `json:"base"`
	Branch    string `json:"branch"`
}

// Budget caps one agent call.
type Budget struct {
	MaxUSD     float64 `json:"max_usd"`
	MaxTokens  int     `json:"max_tokens"`
	MaxMinutes int     `json:"max_minutes"`
}

// Validate checks a TaskEnvelope.
func (e *TaskEnvelope) Validate() error {
	c := newChecker("TaskEnvelope")
	c.schema(e.SchemaVersion)
	c.required(e.RunID, "run_id")
	c.required(e.WorkID, "work_id")
	c.oneOf(e.Role, "role", "writer", "reviewer")
	c.oneOf(e.Track, "track", Tracks...)
	c.check(e.Tier.Valid(), "tier %d must be 0..4", e.Tier)
	c.check(e.Attempt >= 1, "attempt must be >= 1")

	_, _, err := ParseModelRef(e.Model.Ref)
	c.check(err == nil, "model.ref: %v", err)
	c.oneOf(e.Model.Backend, "model.backend", "api", "cursor")
	if e.Model.Backend == "cursor" {
		c.check(Family(e.Model.Ref) != "", "cursor backend needs a valid model ref")
	}
	for _, f := range e.Model.Fallbacks {
		_, _, err := ParseModelRef(f)
		c.check(err == nil, "model.fallbacks: %v", err)
	}

	cp := e.ContextPacket
	c.required(cp.Outcome, "context_packet.outcome")
	c.check(len(cp.ScopeFiles) > 0, "context_packet.scope_files must not be empty")
	if e.Tier.Valid() {
		c.check(cp.LineCap == LineCap(e.Tier), "context_packet.line_cap must equal the tier cap %d", LineCap(e.Tier))
	}
	c.check(len(cp.RulesRefs) > 0, "context_packet.rules_refs must not be empty")
	for _, r := range cp.RulesRefs {
		c.check(strings.Contains(r, "@"), "rules ref %q must be pinned with @<sha>", r)
	}

	c.required(e.Workspace.SandboxID, "workspace.sandbox_id")
	c.required(e.Workspace.Base, "workspace.base")
	c.check(strings.HasPrefix(e.Workspace.Branch, "agent/"), "workspace.branch must start with agent/")

	c.check(len(e.Permissions) > 0, "permissions must not be empty")
	if e.Role == "reviewer" {
		for _, p := range e.Permissions {
			c.check(!strings.HasPrefix(p, "fs.rw") && p != "shell" && p != "git.local",
				"reviewer may not have write or shell permission %q", p)
		}
	}

	c.check(e.Budget.MaxUSD > 0, "budget.max_usd must be > 0")
	c.check(e.Budget.MaxTokens > 0, "budget.max_tokens must be > 0")
	c.check(e.Budget.MaxMinutes > 0, "budget.max_minutes must be > 0")
	return c.err()
}
