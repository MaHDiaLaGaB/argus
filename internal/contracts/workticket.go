package contracts

import "strconv"

// Tracks an agent can work in (ARCHITECTURE.md 7.1).
var Tracks = []string{"backend", "frontend", "mobile", "design", "devops", "data", "docs", "test", "security"}

// MaxOutcomeChars keeps a WorkTicket describable in a short paragraph (threshold rule).
const MaxOutcomeChars = 600

// WorkTicket is one small, provable unit of work produced by the Decomposer (ARCHITECTURE.md 5.2).
type WorkTicket struct {
	SchemaVersion          string   `json:"schema_version"`
	WorkID                 string   `json:"work_id"`
	ParentTicket           string   `json:"parent_ticket"`
	Repo                   string   `json:"repo"`
	Track                  string   `json:"track"`
	Tier                   Tier     `json:"tier"`
	TierReasons            []string `json:"tier_reasons"`
	Outcome                string   `json:"outcome"`
	Contract               string   `json:"contract"`
	ScopeFiles             []string `json:"scope_files"`
	LineCap                int      `json:"line_cap"`
	ProofTest              string   `json:"proof_test"`
	DependsOn              []string `json:"depends_on"`
	NewDependenciesAllowed bool     `json:"new_dependencies_allowed"`
	AcceptanceCriteria     []string `json:"acceptance_criteria"`
	Split                  string   `json:"split"`
}

// Validate enforces the WorkTicket rules in code.
func (w *WorkTicket) Validate() error {
	c := newChecker("WorkTicket")
	c.schema(w.SchemaVersion)
	c.required(w.WorkID, "work_id")
	c.required(w.ParentTicket, "parent_ticket")
	c.check(validRepo(w.Repo), "repo %q must look like owner/name", w.Repo)
	c.oneOf(w.Track, "track", Tracks...)
	c.check(w.Tier.Valid(), "tier %d must be 0..4", w.Tier)
	c.check(len(w.TierReasons) > 0, "tier_reasons must not be empty")
	c.required(w.Outcome, "outcome")
	c.check(len(w.Outcome) <= MaxOutcomeChars, "outcome must be at most %d characters (split the ticket)", MaxOutcomeChars)
	c.required(w.Contract, "contract")
	c.oneOf(w.Split, "split", "vertical", "horizontal")
	c.check(len(w.AcceptanceCriteria) > 0, "acceptance_criteria must not be empty")

	if w.Tier.Valid() {
		c.check(w.LineCap == LineCap(w.Tier), "line_cap %d must equal the tier %d cap %d", w.LineCap, w.Tier, LineCap(w.Tier))
	}

	// Scope: named, safe, no duplicates; CI paths only in Tier 0 devops tickets.
	c.check(len(w.ScopeFiles) > 0, "scope_files must name at least one file")
	seen := map[string]bool{}
	for _, f := range w.ScopeFiles {
		c.check(safeRelPath(f), "scope file %q must be a clean repo-relative path", f)
		c.check(!seen[f], "scope file %q listed twice", f)
		seen[f] = true
		if w.Tier != 0 {
			c.check(!IsCIPath(f), "scope file %q is CI/build config; only Tier 0 tickets may touch it", f)
		}
	}
	if w.Tier == 0 {
		c.check(w.Track == "devops", "Tier 0 (project CI) tickets must be in the devops track")
		for _, f := range w.ScopeFiles {
			c.check(IsCIPath(f), "Tier 0 ticket may only touch CI/build config, not %q", f)
		}
	}

	// Proof: one test that fails before and passes after. Tier 0 CI changes prove themselves by a green pipeline run.
	if w.Tier != 0 {
		c.required(w.ProofTest, "proof_test")
	}

	// No new dependencies below Tier 3.
	if w.NewDependenciesAllowed {
		c.check(w.Tier >= 3, "new dependencies are only allowed at Tier 3 or 4")
	}

	for _, d := range w.DependsOn {
		c.check(d != w.WorkID, "work ticket cannot depend on itself")
	}
	return c.err()
}

// SharedFiles returns files that appear in more than one ticket of the set.
// Tickets that run in parallel must not share a file (ARCHITECTURE.md 6.2).
func SharedFiles(tickets []WorkTicket) map[string][]string {
	owners := map[string][]string{}
	for _, t := range tickets {
		for _, f := range t.ScopeFiles {
			owners[t.Repo+":"+f] = append(owners[t.Repo+":"+f], t.WorkID)
		}
	}
	shared := map[string][]string{}
	for f, ids := range owners {
		if len(ids) > 1 {
			shared[f] = ids
		}
	}
	return shared
}

func itoa(i int) string { return strconv.Itoa(i) }
