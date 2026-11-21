package contracts

import (
	"math"
	"time"
)

// Score weights (sum to 1.0).
const (
	WeightFirstPass      = 0.30
	WeightScope          = 0.25
	WeightTestProof      = 0.20
	WeightCostSpeed      = 0.10
	WeightReviewSurvival = 0.15
)

// ScoreRecord is one line in the score file, written after every merge (ARCHITECTURE.md 5.9).
// Scores are used only to route models.
type ScoreRecord struct {
	SchemaVersion     string       `json:"schema_version"`
	TS                time.Time    `json:"ts"`
	Org               string       `json:"org"`
	Ticket            string       `json:"ticket"`
	Repo              string       `json:"repo"`
	Track             string       `json:"track"`
	Tier              Tier         `json:"tier"`
	WorkType          string       `json:"work_type"`
	Writer            string       `json:"writer"`
	Backend           string       `json:"backend"`
	Reviewer          string       `json:"reviewer"`
	Adjudicator       string       `json:"adjudicator"`
	Attempts          int          `json:"attempts"`
	FilesChanged      int          `json:"files_changed"`
	LinesChanged      int          `json:"lines_changed"`
	ScopePass         bool         `json:"scope_pass"`
	TestProofPass     bool         `json:"test_proof_pass"`
	Findings          FindingTally `json:"findings"`
	InputTokens       int          `json:"input_tokens"`
	OutputTokens      int          `json:"output_tokens"`
	USD               float64      `json:"usd"`
	DurationS         int          `json:"duration_s"`
	Dims              ScoreDims    `json:"dims"`
	Total             float64      `json:"total"`
	RevertedWithin14d *bool        `json:"reverted_within_14d"`
}

// FindingTally counts review findings that were accepted (fixed) or rejected (dismissed).
type FindingTally struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

// ScoreDims are the five scoring dimensions, each in [0,1].
type ScoreDims struct {
	FirstPass      float64 `json:"first_pass"`
	Scope          float64 `json:"scope"`
	TestProof      float64 `json:"test_proof"`
	CostSpeed      float64 `json:"cost_speed"`
	ReviewSurvival float64 `json:"review_survival"`
}

// ComputeTotal returns the weighted total, rounded to 4 decimals.
// A change reverted within 14 days scores 0.
func ComputeTotal(d ScoreDims, reverted bool) float64 {
	if reverted {
		return 0
	}
	t := WeightFirstPass*d.FirstPass + WeightScope*d.Scope + WeightTestProof*d.TestProof +
		WeightCostSpeed*d.CostSpeed + WeightReviewSurvival*d.ReviewSurvival
	return math.Round(t*10000) / 10000
}

// FirstPassScore is 1.0 for green CI on the first attempt, minus 0.5 per extra attempt.
func FirstPassScore(attempts int) float64 {
	if attempts <= 1 {
		return 1
	}
	return math.Max(0, 1-0.5*float64(attempts-1))
}

// Validate checks a ScoreRecord, including the family rules and the total.
func (s *ScoreRecord) Validate() error {
	c := newChecker("ScoreRecord")
	c.schema(s.SchemaVersion)
	c.check(!s.TS.IsZero(), "ts is required")
	c.required(s.Org, "org")
	c.required(s.Ticket, "ticket")
	c.check(validRepo(s.Repo), "repo %q must look like owner/name", s.Repo)
	c.oneOf(s.Track, "track", Tracks...)
	c.check(s.Tier.Valid(), "tier %d must be 0..4", s.Tier)
	c.required(s.WorkType, "work_type")
	c.oneOf(s.Backend, "backend", "api", "cursor")
	for field, ref := range map[string]string{"writer": s.Writer, "reviewer": s.Reviewer, "adjudicator": s.Adjudicator} {
		_, _, err := ParseModelRef(ref)
		c.check(err == nil, "%s: %v", field, err)
	}
	if Family(s.Writer) != "" {
		c.check(Family(s.Reviewer) != Family(s.Writer), "reviewer must be a different family than writer")
		c.check(Family(s.Adjudicator) != Family(s.Writer), "adjudicator must be a different family than writer")
	}
	c.check(s.Attempts >= 1, "attempts must be >= 1")
	c.check(s.FilesChanged >= 0 && s.LinesChanged >= 0, "files_changed and lines_changed must be >= 0")
	c.check(s.USD >= 0 && s.InputTokens >= 0 && s.OutputTokens >= 0 && s.DurationS >= 0, "usage values must be >= 0")
	d := s.Dims
	c.check(in01(d.FirstPass) && in01(d.Scope) && in01(d.TestProof) && in01(d.CostSpeed) && in01(d.ReviewSurvival),
		"every dims value must be between 0 and 1")
	reverted := s.RevertedWithin14d != nil && *s.RevertedWithin14d
	want := ComputeTotal(d, reverted)
	c.check(math.Abs(s.Total-want) < 0.0005, "total %.4f does not match computed %.4f", s.Total, want)
	return c.err()
}
