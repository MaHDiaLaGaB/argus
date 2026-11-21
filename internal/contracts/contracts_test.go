package contracts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T, kind string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "valid", kind+".json"))
	if err != nil {
		t.Fatalf("read fixture %s: %v", kind, err)
	}
	return b
}

// decodeAs loads the valid fixture of kind into v (without validation), so tests can mutate it.
func decodeAs(t *testing.T, kind string, v any) {
	t.Helper()
	if err := json.Unmarshal(loadFixture(t, kind), v); err != nil {
		t.Fatalf("unmarshal %s: %v", kind, err)
	}
}

func wantProblem(t *testing.T, err error, substr string) {
	t.Helper()
	if substr == "" {
		if err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
		return
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError containing %q, got %v", substr, err)
	}
	if !strings.Contains(ve.Error(), substr) {
		t.Fatalf("expected problem containing %q, got %v", substr, ve.Problems)
	}
}

func TestValidFixturesDecode(t *testing.T) {
	for _, kind := range Kinds() {
		t.Run(kind, func(t *testing.T) {
			v, ok := New(kind)
			if !ok {
				t.Fatalf("unknown kind %s", kind)
			}
			if err := Decode(loadFixture(t, kind), v); err != nil {
				t.Fatalf("valid fixture rejected: %v", err)
			}
		})
	}
}

func TestDecodeIsStrict(t *testing.T) {
	data := loadFixture(t, "ticket")
	withExtra := strings.Replace(string(data), `"kind"`, `"surprise": 1, "kind"`, 1)
	if err := Decode([]byte(withExtra), &Ticket{}); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
	if err := Decode(append(data, []byte(`{}`)...), &Ticket{}); err == nil {
		t.Fatal("expected trailing data error")
	}
}

func TestWorkTicketRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(w *WorkTicket)
		want   string
	}{
		{"valid", func(w *WorkTicket) {}, ""},
		{"wrong line cap", func(w *WorkTicket) { w.LineCap = 400 }, "must equal the tier 2 cap 200"},
		{"tier 1 cap", func(w *WorkTicket) { w.Tier = 1; w.LineCap = 30 }, ""},
		{"no scope", func(w *WorkTicket) { w.ScopeFiles = nil }, "scope_files must name at least one file"},
		{"absolute path", func(w *WorkTicket) { w.ScopeFiles = []string{"/etc/passwd"} }, "clean repo-relative path"},
		{"escapes repo", func(w *WorkTicket) { w.ScopeFiles = []string{"../secrets.txt"} }, "clean repo-relative path"},
		{"duplicate file", func(w *WorkTicket) { w.ScopeFiles = []string{"a.go", "a.go"} }, "listed twice"},
		{"code ticket touches CI", func(w *WorkTicket) { w.ScopeFiles = append(w.ScopeFiles, ".github/workflows/ci.yml") }, "only Tier 0 tickets may touch it"},
		{"code ticket touches Dockerfile", func(w *WorkTicket) { w.ScopeFiles = []string{"Dockerfile"} }, "only Tier 0 tickets may touch it"},
		{"tier 0 must be devops", func(w *WorkTicket) {
			w.Tier, w.LineCap, w.ScopeFiles = 0, 200, []string{".github/workflows/ci.yml"}
		}, "must be in the devops track"},
		{"tier 0 only CI files", func(w *WorkTicket) {
			w.Tier, w.LineCap, w.Track, w.ScopeFiles = 0, 200, "devops", []string{"src/main.go"}
		}, "may only touch CI/build config"},
		{"tier 0 valid without proof test", func(w *WorkTicket) {
			w.Tier, w.LineCap, w.Track, w.ProofTest = 0, 200, "devops", ""
			w.ScopeFiles = []string{".github/workflows/ci.yml"}
		}, ""},
		{"missing proof test", func(w *WorkTicket) { w.ProofTest = "" }, "proof_test is required"},
		{"new deps below tier 3", func(w *WorkTicket) { w.NewDependenciesAllowed = true }, "only allowed at Tier 3 or 4"},
		{"new deps at tier 3", func(w *WorkTicket) { w.Tier, w.LineCap, w.NewDependenciesAllowed = 3, 400, true }, ""},
		{"outcome too long", func(w *WorkTicket) { w.Outcome = strings.Repeat("x", MaxOutcomeChars+1) }, "split the ticket"},
		{"self dependency", func(w *WorkTicket) { w.DependsOn = []string{w.WorkID} }, "cannot depend on itself"},
		{"bad tier", func(w *WorkTicket) { w.Tier = 7 }, "must be 0..4"},
		{"bad track", func(w *WorkTicket) { w.Track = "magic" }, "track"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w WorkTicket
			decodeAs(t, "workticket", &w)
			tc.mutate(&w)
			wantProblem(t, w.Validate(), tc.want)
		})
	}
}

func TestSharedFiles(t *testing.T) {
	a := WorkTicket{WorkID: "A", Repo: "o/r", ScopeFiles: []string{"x.go", "y.go"}}
	b := WorkTicket{WorkID: "B", Repo: "o/r", ScopeFiles: []string{"y.go"}}
	c := WorkTicket{WorkID: "C", Repo: "o/other", ScopeFiles: []string{"y.go"}}
	got := SharedFiles([]WorkTicket{a, b, c})
	if len(got) != 1 || len(got["o/r:y.go"]) != 2 {
		t.Fatalf("expected only o/r:y.go shared by A and B, got %v", got)
	}
}

func TestEnvelopeRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(e *TaskEnvelope)
		want   string
	}{
		{"valid", func(e *TaskEnvelope) {}, ""},
		{"bad model ref", func(e *TaskEnvelope) { e.Model.Ref = "claude-sonnet" }, "provider:name"},
		{"unknown provider", func(e *TaskEnvelope) { e.Model.Ref = "acme:model" }, "unknown provider"},
		{"branch not agent", func(e *TaskEnvelope) { e.Workspace.Branch = "master" }, "must start with agent/"},
		{"unpinned rules", func(e *TaskEnvelope) { e.ContextPacket.RulesRefs = []string{"org:AGENTS.md"} }, "pinned with @<sha>"},
		{"reviewer with shell", func(e *TaskEnvelope) { e.Role = "reviewer" }, "reviewer may not have write or shell permission"},
		{"reviewer read only", func(e *TaskEnvelope) { e.Role = "reviewer"; e.Permissions = []string{"repo.read", "diff.read"} }, ""},
		{"zero budget", func(e *TaskEnvelope) { e.Budget.MaxUSD = 0 }, "budget.max_usd"},
		{"cap mismatch", func(e *TaskEnvelope) { e.ContextPacket.LineCap = 30 }, "line_cap must equal the tier cap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var e TaskEnvelope
			decodeAs(t, "envelope", &e)
			tc.mutate(&e)
			wantProblem(t, e.Validate(), tc.want)
		})
	}
}

func TestWriterResultRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(r *WriterResult)
		want   string
	}{
		{"valid", func(r *WriterResult) {}, ""},
		{"done without diff", func(r *WriterResult) { r.DiffURI = "" }, "diff_uri is required"},
		{"done with failing tests", func(r *WriterResult) { r.SelfCheck.TestsFailed = 2 }, "tests_failed must be 0"},
		{"grew without report", func(r *WriterResult) { r.Status = "grew_too_big" }, "grew_report is required"},
		{"not reused without reason", func(r *WriterResult) { r.ReuseDecisions[0].Decision = "not_reused" }, "reason is required"},
		{"bad status", func(r *WriterResult) { r.Status = "kinda_done" }, "status"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r WriterResult
			decodeAs(t, "writer-result", &r)
			tc.mutate(&r)
			wantProblem(t, r.Validate(), tc.want)
		})
	}
}

func TestWriterResultCheckAgainstTicket(t *testing.T) {
	var w WorkTicket
	decodeAs(t, "workticket", &w)
	var r WriterResult
	decodeAs(t, "writer-result", &r)
	if v := r.CheckAgainst(&w); len(v) != 0 {
		t.Fatalf("expected no violations, got %v", v)
	}
	r.FilesChanged = append(r.FilesChanged, "src/billing/invoice.py")
	r.LinesChanged = 250
	r.Proof.FailsOnParent = false
	r.NewDependencies = []string{"requests"}
	v := r.CheckAgainst(&w)
	for _, want := range []string{"outside scope", "exceed line cap", "test proof missing", "new dependencies"} {
		if !strings.Contains(strings.Join(v, "|"), want) {
			t.Errorf("missing violation %q in %v", want, v)
		}
	}
}

func TestReviewRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(r *ReviewReport)
		want   string
	}{
		{"valid", func(r *ReviewReport) {}, ""},
		{"same family", func(r *ReviewReport) { r.ReviewerModel = "anthropic:claude-opus-5.5" }, "different model family"},
		{"pass with blocking", func(r *ReviewReport) { r.Findings[0].Severity = "blocking" }, "not allowed with 1 blocking"},
		{"pass with failed checklist", func(r *ReviewReport) { r.Checklist.SecurityOK = false }, "every checklist item"},
		{"changes required with blocking", func(r *ReviewReport) {
			r.Verdict = "changes_required"
			r.Findings[0].Severity = "blocking"
		}, ""},
		{"duplicate finding id", func(r *ReviewReport) { r.Findings = append(r.Findings, r.Findings[0]) }, "duplicated"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r ReviewReport
			decodeAs(t, "review", &r)
			tc.mutate(&r)
			wantProblem(t, r.Validate(), tc.want)
		})
	}
}

func TestGateEvaluate(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(g *GateResult)
		wantState  string
		wantReason string
	}{
		{"all green tier 2", func(g *GateResult) {}, "ready_to_merge", ""},
		{"scope fails", func(g *GateResult) { g.CIChecks["scope"] = "fail" }, "blocked", "CI check scope: fail"},
		{"missing check", func(g *GateResult) { delete(g.CIChecks, "gitleaks") }, "blocked", "missing CI check: gitleaks"},
		{"extra check failing", func(g *GateResult) { g.CIChecks["e2e"] = "fail" }, "blocked", "CI check e2e: fail"},
		{"ai review changes", func(g *GateResult) { g.AIReview = "changes_required" }, "blocked", "AI review: changes_required"},
		{"security veto", func(g *GateResult) { g.SecurityReview = "veto" }, "blocked", "security veto"},
		{"ci suspect", func(g *GateResult) { g.CISuspect = true }, "blocked", "ci-suspect"},
		{"tier 4 needs human", func(g *GateResult) {
			g.Tier, g.HumanReviewRequired, g.HumanReview = 4, true, "pending"
		}, "blocked", "human review: pending"},
		{"tier 4 approved", func(g *GateResult) {
			g.Tier, g.HumanReviewRequired, g.HumanReview = 4, true, "approved"
		}, "ready_to_merge", ""},
		{"tier 0 without test proof", func(g *GateResult) {
			g.Tier, g.HumanReviewRequired, g.HumanReview, g.SecurityReview = 0, true, "approved", "not_required"
			g.CIChecks["test_proof"] = "not_required"
		}, "ready_to_merge", ""},
		{"tier 2 cannot skip test proof", func(g *GateResult) { g.CIChecks["test_proof"] = "not_required" }, "blocked", "test_proof: not_required"},
		{"tier 1 needs no security", func(g *GateResult) { g.Tier, g.SecurityReview = 1, "not_required" }, "ready_to_merge", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var g GateResult
			decodeAs(t, "gate", &g)
			tc.mutate(&g)
			state, reasons := g.Evaluate()
			if state != tc.wantState {
				t.Fatalf("state = %s, want %s (reasons %v)", state, tc.wantState, reasons)
			}
			if tc.wantReason != "" && !strings.Contains(strings.Join(reasons, "|"), tc.wantReason) {
				t.Fatalf("reasons %v missing %q", reasons, tc.wantReason)
			}
		})
	}
}

func TestGateValidateRejectsWrongState(t *testing.T) {
	var g GateResult
	decodeAs(t, "gate", &g)
	g.CIChecks["scope"] = "fail" // state in fixture still says ready_to_merge
	wantProblem(t, g.Validate(), "does not match evaluated state")
}

func TestHumanReviewTiers(t *testing.T) {
	for tier, want := range map[Tier]bool{0: true, 1: false, 2: false, 3: false, 4: true} {
		if got := HumanReviewRequiredFor(tier, DefaultHumanReviewTiers); got != want {
			t.Errorf("tier %d: human review = %v, want %v", tier, got, want)
		}
	}
}

func TestMergeBriefRules(t *testing.T) {
	var b MergeBrief
	decodeAs(t, "merge-brief", &b)
	wantProblem(t, b.Validate(), "")
	b.RiskNotes = nil
	wantProblem(t, b.Validate(), "risk_notes are required")
	decodeAs(t, "merge-brief", &b)
	b.Tier = 4
	wantProblem(t, b.Validate(), "human_checks are required for Tier 4")
}

func TestScore(t *testing.T) {
	d := ScoreDims{FirstPass: 1, Scope: 1, TestProof: 1, CostSpeed: 0.72, ReviewSurvival: 0.9}
	if got := ComputeTotal(d, false); got != 0.957 {
		t.Fatalf("total = %v, want 0.957", got)
	}
	if got := ComputeTotal(d, true); got != 0 {
		t.Fatalf("reverted total = %v, want 0", got)
	}
	if s := WeightFirstPass + WeightScope + WeightTestProof + WeightCostSpeed + WeightReviewSurvival; s < 0.9999 || s > 1.0001 {
		t.Fatalf("weights sum to %v", s)
	}
	for attempts, want := range map[int]float64{1: 1, 2: 0.5, 3: 0, 5: 0} {
		if got := FirstPassScore(attempts); got != want {
			t.Errorf("FirstPassScore(%d) = %v, want %v", attempts, got, want)
		}
	}

	var s ScoreRecord
	decodeAs(t, "score", &s)
	s.Total = 0.99
	wantProblem(t, s.Validate(), "does not match computed")
	decodeAs(t, "score", &s)
	s.Adjudicator = "anthropic:claude-haiku-5.5"
	wantProblem(t, s.Validate(), "adjudicator must be a different family")
	decodeAs(t, "score", &s)
	yes := true
	s.RevertedWithin14d = &yes
	s.Total = 0
	wantProblem(t, s.Validate(), "")
}

func TestIsCIPath(t *testing.T) {
	ci := []string{".github/workflows/ci.yml", "Dockerfile", "api/Dockerfile.dev", ".devcontainer/devcontainer.json", "docker-compose.yml", ".argus/profile.yaml", "deploy/app.dockerfile"}
	code := []string{"src/main.go", "tests/test_api.py", "docs/github.md", "Makefile", "web/package.json"}
	for _, p := range ci {
		if !IsCIPath(p) {
			t.Errorf("%s should be CI", p)
		}
	}
	for _, p := range code {
		if IsCIPath(p) {
			t.Errorf("%s should not be CI", p)
		}
	}
}

func TestParseModelRef(t *testing.T) {
	if p, n, err := ParseModelRef("xai:grok-4.7"); err != nil || p != "xai" || n != "grok-4.7" {
		t.Fatalf("got %s %s %v", p, n, err)
	}
	for _, bad := range []string{"", "grok", ":x", "xai:", "google:gemini"} {
		if _, _, err := ParseModelRef(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
