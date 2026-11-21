package contracts

// WriterResult is what a Writer agent returns (ARCHITECTURE.md 5.5).
type WriterResult struct {
	SchemaVersion   string          `json:"schema_version"`
	RunID           string          `json:"run_id"`
	WorkID          string          `json:"work_id"`
	Status          string          `json:"status"`
	Summary         string          `json:"summary"`
	DiffURI         string          `json:"diff_uri"`
	FilesChanged    []string        `json:"files_changed"`
	LinesChanged    int             `json:"lines_changed"`
	CommitSHAs      []string        `json:"commit_shas"`
	Proof           Proof           `json:"proof"`
	SelfCheck       SelfCheck       `json:"self_check"`
	ReuseDecisions  []ReuseDecision `json:"reuse_decisions"`
	NewDependencies []string        `json:"new_dependencies"`
	GrewReport      *string         `json:"grew_report"`
	Usage           Usage           `json:"usage"`
	LearnedCands    []string        `json:"learned_candidates"`
}

// Proof records the fail-before / pass-after test result.
type Proof struct {
	FailsOnParent  bool `json:"fails_on_parent"`
	PassesOnChange bool `json:"passes_on_change"`
}

// SelfCheck is the Writer's own sandbox run.
type SelfCheck struct {
	BuildOK     bool `json:"build_ok"`
	TestsPassed int  `json:"tests_passed"`
	TestsFailed int  `json:"tests_failed"`
	LintClean   bool `json:"lint_clean"`
}

// ReuseDecision says whether a reuse candidate was used.
type ReuseDecision struct {
	Candidate string `json:"candidate"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason"`
}

// Usage is model usage for one step.
type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	USD          float64 `json:"usd"`
	WallSeconds  int     `json:"wall_seconds"`
}

// WriterStatuses are the allowed WriterResult statuses.
var WriterStatuses = []string{"done", "failed", "grew_too_big", "needs_input", "env_broken", "budget_exceeded"}

// Validate checks a WriterResult.
func (r *WriterResult) Validate() error {
	c := newChecker("WriterResult")
	c.schema(r.SchemaVersion)
	c.required(r.RunID, "run_id")
	c.required(r.WorkID, "work_id")
	c.oneOf(r.Status, "status", WriterStatuses...)
	c.required(r.Summary, "summary")
	c.check(r.LinesChanged >= 0, "lines_changed must be >= 0")
	c.check(r.Usage.InputTokens >= 0 && r.Usage.OutputTokens >= 0 && r.Usage.USD >= 0, "usage values must be >= 0")
	for _, f := range r.FilesChanged {
		c.check(safeRelPath(f), "changed file %q must be a clean repo-relative path", f)
	}
	for i, d := range r.ReuseDecisions {
		c.required(d.Candidate, "reuse_decisions["+itoa(i)+"].candidate")
		c.oneOf(d.Decision, "reuse_decisions["+itoa(i)+"].decision", "reused", "not_reused")
		if d.Decision == "not_reused" {
			c.required(d.Reason, "reuse_decisions["+itoa(i)+"].reason")
		}
	}
	switch r.Status {
	case "done":
		c.required(r.DiffURI, "diff_uri")
		c.check(len(r.FilesChanged) > 0, "files_changed must not be empty when status is done")
		c.check(len(r.CommitSHAs) > 0, "commit_shas must not be empty when status is done")
		c.check(r.SelfCheck.BuildOK, "self_check.build_ok must be true when status is done")
		c.check(r.SelfCheck.TestsFailed == 0, "self_check.tests_failed must be 0 when status is done")
	case "grew_too_big":
		c.check(r.GrewReport != nil && *r.GrewReport != "", "grew_report is required when status is grew_too_big")
	}
	return c.err()
}

// CheckAgainst compares a finished WriterResult with its WorkTicket using
// deterministic rules only: scope, line cap, test proof and dependencies.
// It returns the list of violations (empty means the result respects the ticket).
func (r *WriterResult) CheckAgainst(w *WorkTicket) []string {
	var v []string
	scope := map[string]bool{}
	for _, f := range w.ScopeFiles {
		scope[f] = true
	}
	for _, f := range r.FilesChanged {
		if !scope[f] {
			v = append(v, "file outside scope: "+f)
		}
	}
	if r.LinesChanged > w.LineCap {
		v = append(v, "lines changed "+itoa(r.LinesChanged)+" exceed line cap "+itoa(w.LineCap))
	}
	if w.Tier != 0 && (!r.Proof.FailsOnParent || !r.Proof.PassesOnChange) {
		v = append(v, "test proof missing: proof test must fail on parent and pass on change")
	}
	if len(r.NewDependencies) > 0 && !w.NewDependenciesAllowed {
		v = append(v, "new dependencies not allowed for this ticket")
	}
	return v
}

// ReviewReport is the Reviewer agent's output (ARCHITECTURE.md 5.6).
type ReviewReport struct {
	SchemaVersion string          `json:"schema_version"`
	WorkID        string          `json:"work_id"`
	PR            string          `json:"pr"`
	ReviewerModel string          `json:"reviewer_model"`
	WriterModel   string          `json:"writer_model"`
	Verdict       string          `json:"verdict"`
	Findings      []Finding       `json:"findings"`
	Checklist     ReviewChecklist `json:"checklist"`
}

// Finding is one review comment.
type Finding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Rule     string `json:"rule"`
	Text     string `json:"text"`
}

// ReviewChecklist is the fixed checklist every review answers.
type ReviewChecklist struct {
	InScope          bool `json:"in_scope"`
	SmallestDiff     bool `json:"smallest_diff"`
	TestProvesChange bool `json:"test_proves_change"`
	NoNewDeps        bool `json:"no_new_deps"`
	ReuseOK          bool `json:"reuse_ok"`
	SecurityOK       bool `json:"security_ok"`
}

// Validate checks a ReviewReport, including the cross-family rule.
func (r *ReviewReport) Validate() error {
	c := newChecker("ReviewReport")
	c.schema(r.SchemaVersion)
	c.required(r.WorkID, "work_id")
	c.required(r.PR, "pr")
	_, _, errR := ParseModelRef(r.ReviewerModel)
	c.check(errR == nil, "reviewer_model: %v", errR)
	_, _, errW := ParseModelRef(r.WriterModel)
	c.check(errW == nil, "writer_model: %v", errW)
	if errR == nil && errW == nil {
		c.check(Family(r.ReviewerModel) != Family(r.WriterModel),
			"reviewer %q must be from a different model family than writer %q", r.ReviewerModel, r.WriterModel)
	}
	c.oneOf(r.Verdict, "verdict", "pass", "changes_required", "block")
	ids := map[string]bool{}
	blocking := 0
	for i, f := range r.Findings {
		p := "findings[" + itoa(i) + "]"
		c.required(f.ID, p+".id")
		c.check(!ids[f.ID], "%s.id %q is duplicated", p, f.ID)
		ids[f.ID] = true
		c.oneOf(f.Severity, p+".severity", "blocking", "major", "minor", "nit")
		c.oneOf(f.Rule, p+".rule", "scope", "smallest-diff", "security", "correctness", "reuse", "style", "tests")
		c.required(f.Text, p+".text")
		if f.Severity == "blocking" {
			blocking++
		}
	}
	if r.Verdict == "pass" {
		c.check(blocking == 0, "verdict pass is not allowed with %d blocking findings", blocking)
		cl := r.Checklist
		c.check(cl.InScope && cl.SmallestDiff && cl.TestProvesChange && cl.NoNewDeps && cl.ReuseOK && cl.SecurityOK,
			"verdict pass requires every checklist item to be true")
	}
	return c.err()
}
