#!/usr/bin/env bash
# Creates labels, milestones, epics and issues for argus in GitHub.
# Usage: scripts/create-github-issues.sh owner/repo   (needs gh auth with repo scope)
# Safe to read before running: it only creates labels, milestones and issues.
set -euo pipefail
REPO="${1:?usage: $0 owner/repo}"
gh repo view "$REPO" >/dev/null

label() { gh label create "$1" --repo "$REPO" --color "$2" --description "$3" --force >/dev/null; }
label epic 5319e7 'Epic: groups issues'
for t in 0 1 2 3 4; do label "tier:$t" c5def5 "Pipeline tier $t"; done
for s in S M L; do label "size:$s" ededed "Size $s"; done
label 'E01' fbca04 'Foundation and contracts'
label 'E02' fbca04 'Model clients (plain HTTPS)'
label 'E03' fbca04 'Sandbox runner'
label 'E04' fbca04 'Agents core'
label 'E05' fbca04 'GitHub layer'
label 'E06' fbca04 'Templates v1 (be-go, be-fastapi)'
label 'E07' fbca04 'Phase 0 CLI and acceptance'
label 'E08' fbca04 'Pipeline CI gates (build once, freeze)'
label 'E09' fbca04 'Orchestrator and workflows'
label 'E10' fbca04 'Intake'
label 'E11' fbca04 'Tracks and communication'
label 'E12' fbca04 'Storage and observability'
label 'E13' fbca04 'Templates v2 and argus init'
label 'E14' fbca04 'Existing repo onboarding'
label 'E15' fbca04 'New project flow'
label 'E16' fbca04 'Code index and reuse'
label 'E17' fbca04 'Scoring and routing'
label 'E18' fbca04 'Mobile templates'
label 'E19' fbca04 'Cursor backend'
label 'E20' fbca04 'Multi-repo'
label 'E21' fbca04 'Learning: skills and memory'
label 'E22' fbca04 'Security hardening'
label 'E23' fbca04 'Pilot rollout'

milestone() { gh api "repos/$REPO/milestones" -f title="$1" -f description="$2" >/dev/null 2>&1 || true; }
milestone 'Phase 0 - One ticket end to end' 'Go skeleton, contracts, model clients, sandbox, Writer + Reviewer, GitHub layer, first templates. Accept: 6 of 10 benchmark tickets reach a green tier-gate with no human edits.'
milestone 'Phase 1 - Intake, decomposition, gates' 'Frozen pipeline CI gates, Temporal orchestrator, intake, all tracks, more templates.'
milestone 'Phase 2 - Onboarding, new projects, routing' 'Existing repos, new projects with argus init, code index and reuse, scoring and routing, mobile templates, Cursor backend.'
milestone 'Phase 3 - Multi-repo, learning, hardening' 'Org map and change sets, skills and memory, security hardening.'
milestone 'Phase 4 - Pilot on a real team'"'"'s repos' 'Humans roll out to one pilot repo, then expand.'

new_issue() { gh issue create --repo "$REPO" --title "$1" --body "$2" --label "$3" --milestone "$4" | sed "s#.*/##"; }

# E01
EPIC_E01=$(new_issue '[E01] Foundation and contracts' 'Repo skeleton, contracts every component speaks, config and policy loading.' 'epic,E01' 'Phase 0 - One ticket end to end')
CHECK_E01=""
B='go.mod, Makefile verbs (setup build test test-one lint fmt run), AGENTS.md, README, CI on master with golangci-lint, govulncheck.

## Acceptance
- [ ] make lint test build pass in CI
- [ ] Branch master protected

Part of #EPICNUM (E01). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E01}
N=$(new_issue '[E01.01] Go repo skeleton' "$B" 'E01,tier:1,size:S' 'Phase 0 - One ticket end to end')
gh issue close "$N" --repo "$REPO" --comment "Done in the initial commit." >/dev/null
CHECK_E01="$CHECK_E01- [x] #$N"$'\n'
B='Ticket, WorkTicket, TaskEnvelope, WriterResult, ReviewReport, GateResult, MergeBrief, ScoreRecord with strict decoding and validation; tier-gate evaluation and score math in code.

## Acceptance
- [ ] Valid fixtures pass, invalid cases rejected by table tests
- [ ] CLI validate and gate commands

Part of #EPICNUM (E01). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E01}
N=$(new_issue '[E01.02] Contracts package' "$B" 'E01,tier:2,size:M' 'Phase 0 - One ticket end to end')
gh issue close "$N" --repo "$REPO" --comment "Done in the initial commit." >/dev/null
CHECK_E01="$CHECK_E01- [x] #$N"$'\n'
B='Reflection-based generator writes contracts/schema/*.json; CI fails if generated files are stale.

## Acceptance
- [ ] One schema per contract kind
- [ ] CI check for drift

Part of #EPICNUM (E01). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E01}
N=$(new_issue '[E01.03] Generate JSON Schemas from contracts' "$B" 'E01,tier:2,size:S' 'Phase 0 - One ticket end to end')
CHECK_E01="$CHECK_E01- [ ] #$N"$'\n'
B='Load config/*.yaml and policy/*.yaml (models, pricing, routing, allowlist, tiers, caps, human_review_tiers); validate at startup; refuse to start if invalid.

## Acceptance
- [ ] Invalid config stops startup with a clear error
- [ ] Policy is read-only at runtime

Part of #EPICNUM (E01). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E01}
N=$(new_issue '[E01.04] Config and policy loader' "$B" 'E01,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E01="$CHECK_E01- [ ] #$N"$'\n'
B='config/models.yaml, pricing.yaml, benchmarks.yaml and policy/allowlist.yaml filled from ARCHITECTURE Section 9 with TODO(verify) markers.

## Acceptance
- [ ] Every model in allowlist has a price
- [ ] Tier floors encoded

Part of #EPICNUM (E01). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E01}
N=$(new_issue '[E01.05] Model, price and allowlist config v1' "$B" 'E01,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E01="$CHECK_E01- [ ] #$N"$'\n'
B='gitleaks in CI, gosec, Dependabot for Go modules and Actions, pinned action SHAs.

## Acceptance
- [ ] CI fails on a planted fake secret

Part of #EPICNUM (E01). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E01}
N=$(new_issue '[E01.06] Secret scanning and supply chain for this repo' "$B" 'E01,tier:0,size:S' 'Phase 0 - One ticket end to end')
CHECK_E01="$CHECK_E01- [ ] #$N"$'\n'
gh issue edit "$EPIC_E01" --repo "$REPO" --body "Repo skeleton, contracts every component speaks, config and policy loading."$'\n\n## Issues\n'"$CHECK_E01" >/dev/null
echo "E01: epic #$EPIC_E01 with 6 issues"

# E02
EPIC_E02=$(new_issue '[E02] Model clients (plain HTTPS)' 'Thin clients for Claude, GPT and Grok with budgets, pricing and logging. No AI frameworks.' 'epic,E02' 'Phase 0 - One ticket end to end')
CHECK_E02=""
B='internal/llm: Complete(ctx, Request) Response with messages, tools (JSON schema), max tokens, cache hints, run/step ids. Normalized usage.

## Acceptance
- [ ] Interface documented
- [ ] Fake client for tests

Part of #EPICNUM (E02). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E02}
N=$(new_issue '[E02.01] LLM client interface' "$B" 'E02,tier:2,size:S' 'Phase 0 - One ticket end to end')
CHECK_E02="$CHECK_E02- [ ] #$N"$'\n'
B='Messages API over net/http: tool use, prompt caching, usage parsing, retries with backoff on 429/5xx.

## Acceptance
- [ ] Recorded-response tests cost $0
- [ ] Tool calls round-trip

Part of #EPICNUM (E02). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E02}
N=$(new_issue '[E02.02] Anthropic client' "$B" 'E02,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E02="$CHECK_E02- [ ] #$N"$'\n'
B='Responses/Chat API over net/http with tool calling and usage parsing.

## Acceptance
- [ ] Recorded-response tests
- [ ] Tool calls round-trip

Part of #EPICNUM (E02). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E02}
N=$(new_issue '[E02.03] OpenAI client' "$B" 'E02,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E02="$CHECK_E02- [ ] #$N"$'\n'
B='Grok API (OpenAI-compatible endpoint, TODO(verify)) reusing the OpenAI wire code.

## Acceptance
- [ ] Recorded-response tests

Part of #EPICNUM (E02). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E02}
N=$(new_issue '[E02.04] xAI client' "$B" 'E02,tier:2,size:S' 'Phase 0 - One ticket end to end')
CHECK_E02="$CHECK_E02- [ ] #$N"$'\n'
B='USD per call from pricing.yaml incl. cached tokens; every call logged (provider, model, tokens, usd, latency, status).

## Acceptance
- [ ] Cost matches hand calculation in tests

Part of #EPICNUM (E02). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E02}
N=$(new_issue '[E02.05] Pricing and token accounting' "$B" 'E02,tier:2,size:S' 'Phase 0 - One ticket end to end')
CHECK_E02="$CHECK_E02- [ ] #$N"$'\n'
B='Hierarchical budgets (org/month > repo/month > ticket > step); pre-call estimate; reject when it would exceed; alert at 80%.

## Acceptance
- [ ] $0.05 cap test stops the run
- [ ] Alert event emitted at 80%

Part of #EPICNUM (E02). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E02}
N=$(new_issue '[E02.06] Budget hard stop' "$B" 'E02,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E02="$CHECK_E02- [ ] #$N"$'\n'
B='On 429/5xx/timeout: retry, then next model in fallbacks, never below tier floor, keep cross-family rule.

## Acceptance
- [ ] Outage simulation falls back correctly
- [ ] Property test: never below floor

Part of #EPICNUM (E02). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E02}
N=$(new_issue '[E02.07] Fallback chain' "$B" 'E02,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E02="$CHECK_E02- [ ] #$N"$'\n'
B='Record/replay HTTP fixtures so tests never call real APIs; secrets stripped from recordings.

## Acceptance
- [ ] No API key in any fixture (grep test)

Part of #EPICNUM (E02). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E02}
N=$(new_issue '[E02.08] Response recorder for tests' "$B" 'E02,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E02="$CHECK_E02- [ ] #$N"$'\n'
gh issue edit "$EPIC_E02" --repo "$REPO" --body "Thin clients for Claude, GPT and Grok with budgets, pricing and logging. No AI frameworks."$'\n\n## Issues\n'"$CHECK_E02" >/dev/null
echo "E02: epic #$EPIC_E02 with 8 issues"

# E03
EPIC_E03=$(new_issue '[E03] Sandbox runner' 'Run any template or repo'"'"'s make verbs in Docker and return parsed results.' 'epic,E03' 'Phase 0 - One ticket end to end')
CHECK_E03=""
B='Create rootless container per run from the repo profile image; mount /work only; CPU/mem/time limits; egress allowlist; destroy after run.

## Acceptance
- [ ] No Docker socket inside
- [ ] Limits enforced in tests

Part of #EPICNUM (E03). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E03}
N=$(new_issue '[E03.01] Docker sandbox lifecycle' "$B" 'E03,tier:3,size:M' 'Phase 0 - One ticket end to end')
CHECK_E03="$CHECK_E03- [ ] #$N"$'\n'
B='Run make setup/build/test/test-one/lint with timeouts; capture exit codes and logs; truncate output with pointer to full log.

## Acceptance
- [ ] Works on be-go and be-fastapi templates

Part of #EPICNUM (E03). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E03}
N=$(new_issue '[E03.02] Make verb runner' "$B" 'E03,tier:2,size:S' 'Phase 0 - One ticket end to end')
CHECK_E03="$CHECK_E03- [ ] #$N"$'\n'
B='Parse go test -json and pytest JUnit XML into failing tests + relevant stack lines only.

## Acceptance
- [ ] Only failing tests reach the model

Part of #EPICNUM (E03). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E03}
N=$(new_issue '[E03.03] Test output parsers: Go and pytest' "$B" 'E03,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E03="$CHECK_E03- [ ] #$N"$'\n'
B='Run proof test on parent commit (must fail) and on change (must pass).

## Acceptance
- [ ] Proof result matches WriterResult.proof

Part of #EPICNUM (E03). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E03}
N=$(new_issue '[E03.04] Test proof check' "$B" 'E03,tier:2,size:S' 'Phase 0 - One ticket end to end')
CHECK_E03="$CHECK_E03- [ ] #$N"$'\n'
B='Apply WriterResult.CheckAgainst (scope, line cap, proof, new deps) before any review.

## Acceptance
- [ ] Violations block review

Part of #EPICNUM (E03). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E03}
N=$(new_issue '[E03.05] Deterministic ticket check' "$B" 'E03,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E03="$CHECK_E03- [ ] #$N"$'\n'
gh issue edit "$EPIC_E03" --repo "$REPO" --body "Run any template or repo's make verbs in Docker and return parsed results."$'\n\n## Issues\n'"$CHECK_E03" >/dev/null
echo "E03: epic #$EPIC_E03 with 5 issues"

# E04
EPIC_E04=$(new_issue '[E04] Agents core' 'One small tool-use loop and a policy engine shared by every LLM agent.' 'epic,E04' 'Phase 0 - One ticket end to end')
CHECK_E04=""
B='In-house loop over llm client: send, execute tool calls, feed results, stop on done/budget/max turns.

## Acceptance
- [ ] Loop is under 400 lines
- [ ] Deterministic tests with fake client

Part of #EPICNUM (E04). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E04}
N=$(new_issue '[E04.01] Tool-use loop' "$B" 'E04,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E04="$CHECK_E04- [ ] #$N"$'\n'
B='Typed tools with JSON schemas; every call checked against envelope permissions and scope files; blocked calls logged.

## Acceptance
- [ ] Out-of-scope write is blocked
- [ ] Reviewer cannot write

Part of #EPICNUM (E04). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E04}
N=$(new_issue '[E04.02] Tool registry and policy engine' "$B" 'E04,tier:3,size:M' 'Phase 0 - One ticket end to end')
CHECK_E04="$CHECK_E04- [ ] #$N"$'\n'
B='Build packet from WorkTicket: rules refs at pinned SHA, file summaries, reuse candidates, prior attempts; enforce token budgets (6k/3k/1.5k/1.5k).

## Acceptance
- [ ] Packet under 6k tokens on fixtures

Part of #EPICNUM (E04). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E04}
N=$(new_issue '[E04.03] Context packet builder' "$B" 'E04,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E04="$CHECK_E04- [ ] #$N"$'\n'
B='Prompts/tracks/backend/writer.md + loop: edit files in scope, run tests, commit, return WriterResult.

## Acceptance
- [ ] Produces valid WriterResult on fixture tickets

Part of #EPICNUM (E04). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E04}
N=$(new_issue '[E04.04] Writer agent (backend track)' "$B" 'E04,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E04="$CHECK_E04- [ ] #$N"$'\n'
B='Prompts/tracks/backend/reviewer.md: diff + static analysis in, ReviewReport out; runs only after CI is green.

## Acceptance
- [ ] Cross-family enforced
- [ ] Valid ReviewReport

Part of #EPICNUM (E04). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E04}
N=$(new_issue '[E04.05] Reviewer agent' "$B" 'E04,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E04="$CHECK_E04- [ ] #$N"$'\n'
B='Every model output parsed into a contract; invalid output retried once with the error, then failed.

## Acceptance
- [ ] Invalid JSON never enters pipeline

Part of #EPICNUM (E04). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E04}
N=$(new_issue '[E04.06] Model output validation' "$B" 'E04,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E04="$CHECK_E04- [ ] #$N"$'\n'
B='Test that fails if a call site to internal/llm is not listed in docs/model-calls.md.

## Acceptance
- [ ] Unlisted call site fails CI

Part of #EPICNUM (E04). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E04}
N=$(new_issue '[E04.07] docs/model-calls.md enforcement' "$B" 'E04,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E04="$CHECK_E04- [ ] #$N"$'\n'
gh issue edit "$EPIC_E04" --repo "$REPO" --body "One small tool-use loop and a policy engine shared by every LLM agent."$'\n\n## Issues\n'"$CHECK_E04" >/dev/null
echo "E04: epic #$EPIC_E04 with 7 issues"

# E05
EPIC_E05=$(new_issue '[E05] GitHub layer' 'Separate Writer, Reviewer and Gate identities; PRs, checks and merge brief.' 'epic,E05' 'Phase 0 - One ticket end to end')
CHECK_E05=""
B='argus-writer, argus-reviewer, argus-gate with least permissions (ARCHITECTURE 11.1). Manual setup checklist in docs.

## Acceptance
- [ ] Apps installed on test org only

Part of #EPICNUM (E05). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E05}
N=$(new_issue '[E05.01] Create GitHub Apps (test org)' "$B" 'E05,tier:3,size:S' 'Phase 0 - One ticket end to end')
CHECK_E05="$CHECK_E05- [ ] #$N"$'\n'
B='JWT -> installation token, 1h, cached, never visible to models or logs.

## Acceptance
- [ ] Token never in traces (grep test)

Part of #EPICNUM (E05). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E05}
N=$(new_issue '[E05.02] Installation token minting' "$B" 'E05,tier:3,size:S' 'Phase 0 - One ticket end to end')
CHECK_E05="$CHECK_E05- [ ] #$N"$'\n'
B='Push agent/* branch, open PR with template: scope list, ACs, evidence, cost.

## Acceptance
- [ ] PR authored by argus-writer

Part of #EPICNUM (E05). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E05}
N=$(new_issue '[E05.03] Branch push and PR creation' "$B" 'E05,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E05="$CHECK_E05- [ ] #$N"$'\n'
B='Submit review and ai-review / security-review check runs as argus-reviewer.

## Acceptance
- [ ] Review authored by argus-reviewer

Part of #EPICNUM (E05). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E05}
N=$(new_issue '[E05.04] Reviewer posting' "$B" 'E05,tier:2,size:S' 'Phase 0 - One ticket end to end')
CHECK_E05="$CHECK_E05- [ ] #$N"$'\n'
B='Collect CI results, review, security, human approval; run GateResult.Evaluate; post check run as argus-gate.

## Acceptance
- [ ] Tier 0 and 4 blocked without human approval

Part of #EPICNUM (E05). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E05}
N=$(new_issue '[E05.05] tier-gate check run' "$B" 'E05,tier:3,size:M' 'Phase 0 - One ticket end to end')
CHECK_E05="$CHECK_E05- [ ] #$N"$'\n'
B='Fill MergeBrief from data; cheap model only for one_line and risk_notes; post as PR comment.

## Acceptance
- [ ] Valid MergeBrief on every green PR

Part of #EPICNUM (E05). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E05}
N=$(new_issue '[E05.06] Merge brief' "$B" 'E05,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E05="$CHECK_E05- [ ] #$N"$'\n'
B='Receive check_run / pull_request / review events; verify signatures; dedupe.

## Acceptance
- [ ] Forged webhook rejected

Part of #EPICNUM (E05). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E05}
N=$(new_issue '[E05.07] Read check results and webhooks' "$B" 'E05,tier:3,size:M' 'Phase 0 - One ticket end to end')
CHECK_E05="$CHECK_E05- [ ] #$N"$'\n'
gh issue edit "$EPIC_E05" --repo "$REPO" --body "Separate Writer, Reviewer and Gate identities; PRs, checks and merge brief."$'\n\n## Issues\n'"$CHECK_E05" >/dev/null
echo "E05: epic #$EPIC_E05 with 7 issues"

# E06
EPIC_E06=$(new_issue '[E06] Templates v1 (be-go, be-fastapi)' 'First two templates with the shared verbs and ready profiles.' 'epic,E06' 'Phase 0 - One ticket end to end')
CHECK_E06=""
B='Makefile verbs, AGENTS.base.md, PR template, gitleaks config, Dependabot, .env.example.

## Acceptance
- [ ] Same verbs in every template

Part of #EPICNUM (E06). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E06}
N=$(new_issue '[E06.01] Shared template parts' "$B" 'E06,tier:0,size:S' 'Phase 0 - One ticket end to end')
CHECK_E06="$CHECK_E06- [ ] #$N"$'\n'
B='Go + chi + pgx/sqlc, health endpoint, sample feature with tests, devcontainer, Dockerfile, CI calling central gates, .argus/profile.yaml.

## Acceptance
- [ ] make setup build lint test passes on a fresh copy

Part of #EPICNUM (E06). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E06}
N=$(new_issue '[E06.02] be-go template' "$B" 'E06,tier:0,size:M' 'Phase 0 - One ticket end to end')
CHECK_E06="$CHECK_E06- [ ] #$N"$'\n'
B='Python + FastAPI + uv + SQLAlchemy/Alembic + ruff + mypy + pytest, same structure as be-go.

## Acceptance
- [ ] make setup build lint test passes on a fresh copy

Part of #EPICNUM (E06). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E06}
N=$(new_issue '[E06.03] be-fastapi template' "$B" 'E06,tier:0,size:M' 'Phase 0 - One ticket end to end')
CHECK_E06="$CHECK_E06- [ ] #$N"$'\n'
B='argus-templates CI generates each template and runs all verbs.

## Acceptance
- [ ] Broken template fails CI

Part of #EPICNUM (E06). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E06}
N=$(new_issue '[E06.04] Template self-test CI' "$B" 'E06,tier:0,size:S' 'Phase 0 - One ticket end to end')
CHECK_E06="$CHECK_E06- [ ] #$N"$'\n'
gh issue edit "$EPIC_E06" --repo "$REPO" --body "First two templates with the shared verbs and ready profiles."$'\n\n## Issues\n'"$CHECK_E06" >/dev/null
echo "E06: epic #$EPIC_E06 with 4 issues"

# E07
EPIC_E07=$(new_issue '[E07] Phase 0 CLI and acceptance' 'Tie it together and prove it on real tickets.' 'epic,E07' 'Phase 0 - One ticket end to end')
CHECK_E07=""
B='argus run --repo owner/name --work-ticket file.json: sandbox -> Writer -> checks -> PR -> Reviewer -> tier-gate -> merge brief.

## Acceptance
- [ ] One fixture ticket reaches green gate on test org

Part of #EPICNUM (E07). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E07}
N=$(new_issue '[E07.01] argus run command' "$B" 'E07,tier:2,size:M' 'Phase 0 - One ticket end to end')
CHECK_E07="$CHECK_E07- [ ] #$N"$'\n'
B='10 Tier 1-2 WorkTickets on be-go and be-fastapi sample repos, plus 5 deliberately bad attempts.

## Acceptance
- [ ] Bad attempts blocked by scope, cap or proof

Part of #EPICNUM (E07). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E07}
N=$(new_issue '[E07.02] Benchmark ticket set' "$B" 'E07,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E07="$CHECK_E07- [ ] #$N"$'\n'
B='Automated grep across prompts, traces, logs and fixtures for keys and tokens.

## Acceptance
- [ ] Planted secret detected

Part of #EPICNUM (E07). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E07}
N=$(new_issue '[E07.03] No-secrets test' "$B" 'E07,tier:2,size:S' 'Phase 0 - One ticket end to end')
CHECK_E07="$CHECK_E07- [ ] #$N"$'\n'
B='Run benchmark, record pass rate, cost, time; sign-off doc.

## Acceptance
- [ ] >= 6 of 10 green with no human edits
- [ ] Report in docs/reports/

Part of #EPICNUM (E07). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E07}
N=$(new_issue '[E07.04] Phase 0 acceptance run and report' "$B" 'E07,tier:1,size:S' 'Phase 0 - One ticket end to end')
CHECK_E07="$CHECK_E07- [ ] #$N"$'\n'
gh issue edit "$EPIC_E07" --repo "$REPO" --body "Tie it together and prove it on real tickets."$'\n\n## Issues\n'"$CHECK_E07" >/dev/null
echo "E07: epic #$EPIC_E07 with 4 issues"

# E08
EPIC_E08=$(new_issue '[E08] Pipeline CI gates (build once, freeze)' 'Central reusable workflows in argus-gates, proven by the gate test suite, then frozen.' 'epic,E08' 'Phase 1 - Intake, decomposition, gates')
CHECK_E08=""
B='Reusable workflow reading scope list from PR and tier label; fails on out-of-scope files or over cap.

## Acceptance
- [ ] Gate test fixtures fail as expected

Part of #EPICNUM (E08). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E08}
N=$(new_issue '[E08.01] Scope and line-cap gates' "$B" 'E08,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E08="$CHECK_E08- [ ] #$N"$'\n'
B='Run proof test on parent and head; must fail then pass.

## Acceptance
- [ ] Fixture without fail-before is blocked

Part of #EPICNUM (E08). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E08}
N=$(new_issue '[E08.02] Test proof gate' "$B" 'E08,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E08="$CHECK_E08- [ ] #$N"$'\n'
B='gitleaks, dependency audit, new-dependency detection by tier, block code PRs touching CI paths.

## Acceptance
- [ ] Each fixture blocked

Part of #EPICNUM (E08). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E08}
N=$(new_issue '[E08.03] Secrets, dependency and CI-path gates' "$B" 'E08,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E08="$CHECK_E08- [ ] #$N"$'\n'
B='jscpd (or per-stack tool, TODO(verify)) on changed code; threshold from policy.

## Acceptance
- [ ] Duplicated block fixture blocked

Part of #EPICNUM (E08). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E08}
N=$(new_issue '[E08.04] Duplication gate' "$B" 'E08,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E08="$CHECK_E08- [ ] #$N"$'\n'
B='CI step reports the Reviewer agent verdict (no second model); score job appends ScoreRecord on merge.

## Acceptance
- [ ] No model call from CI
- [ ] Score line valid

Part of #EPICNUM (E08). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E08}
N=$(new_issue '[E08.05] AI review reporter and score job' "$B" 'E08,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E08="$CHECK_E08- [ ] #$N"$'\n'
B='argus-gates/gate-tests: known-bad PR fixtures; all must fail.

## Acceptance
- [ ] Suite green = every fixture blocked

Part of #EPICNUM (E08). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E08}
N=$(new_issue '[E08.06] Gate test suite' "$B" 'E08,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E08="$CHECK_E08- [ ] #$N"$'\n'
B='Version tag, repos call pinned version, rulesets block agent writes to gates and policy.

## Acceptance
- [ ] Writer App push to gates rejected

Part of #EPICNUM (E08). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E08}
N=$(new_issue '[E08.07] Pin and freeze gates' "$B" 'E08,tier:4,size:S' 'Phase 1 - Intake, decomposition, gates')
CHECK_E08="$CHECK_E08- [ ] #$N"$'\n'
gh issue edit "$EPIC_E08" --repo "$REPO" --body "Central reusable workflows in argus-gates, proven by the gate test suite, then frozen."$'\n\n## Issues\n'"$CHECK_E08" >/dev/null
echo "E08: epic #$EPIC_E08 with 7 issues"

# E09
EPIC_E09=$(new_issue '[E09] Orchestrator and workflows' 'Durable runs: decompose, tier, route, control loop, fault check.' 'epic,E09' 'Phase 1 - Intake, decomposition, gates')
CHECK_E09=""
B='Self-hosted Temporal in docker compose; Go worker; run workflow skeleton with signals for plan approval, human approval, merged.

## Acceptance
- [ ] Run survives worker restart

Part of #EPICNUM (E09). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E09}
N=$(new_issue '[E09.01] Temporal setup' "$B" 'E09,tier:3,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E09="$CHECK_E09- [ ] #$N"$'\n'
B='Rules from policy/tiers.yaml first; cheap model only when rules return unknown.

## Acceptance
- [ ] Unit tests per tier
- [ ] Model not called when rules decide

Part of #EPICNUM (E09). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E09}
N=$(new_issue '[E09.02] Tier classifier' "$B" 'E09,tier:2,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E09="$CHECK_E09- [ ] #$N"$'\n'
B='Opus splits request into WorkTickets; code validates threshold rule, scope files exist, no shared files in parallel, topological order.

## Acceptance
- [ ] Invalid split rejected and retried
- [ ] Plan posted to issue

Part of #EPICNUM (E09). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E09}
N=$(new_issue '[E09.03] Decomposer' "$B" 'E09,tier:3,size:L' 'Phase 1 - Intake, decomposition, gates')
CHECK_E09="$CHECK_E09- [ ] #$N"$'\n'
B='Pick writer/reviewer per tier from routing.yaml; cross-family; Tier 0 pinned to best model.

## Acceptance
- [ ] Property test: reviewer family != writer family

Part of #EPICNUM (E09). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E09}
N=$(new_issue '[E09.04] Static router' "$B" 'E09,tier:2,size:S' 'Phase 1 - Intake, decomposition, gates')
CHECK_E09="$CHECK_E09- [ ] #$N"$'\n'
B='Attempts, revise with findings, escalate model and tier after 2 failures, max 1 escalation, then human.

## Acceptance
- [ ] Loop matches ARCHITECTURE 6.5 in tests

Part of #EPICNUM (E09). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E09}
N=$(new_issue '[E09.05] Control loop and two strikes' "$B" 'E09,tier:3,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E09="$CHECK_E09- [ ] #$N"$'\n'
B='Rerun, base-branch run, sandbox repro, cross-family code question; flag ci-suspect with CI diagnosis; /ci-ok command.

## Acceptance
- [ ] Planted CI fault flagged, planted code fault not

Part of #EPICNUM (E09). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E09}
N=$(new_issue '[E09.06] Fault check and ci-suspect' "$B" 'E09,tier:3,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E09="$CHECK_E09- [ ] #$N"$'\n'
B='Writer can return grew_too_big; Decomposer re-splits without penalty.

## Acceptance
- [ ] Re-split tickets pass validation

Part of #EPICNUM (E09). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E09}
N=$(new_issue '[E09.07] grew_too_big re-split' "$B" 'E09,tier:2,size:S' 'Phase 1 - Intake, decomposition, gates')
CHECK_E09="$CHECK_E09- [ ] #$N"$'\n'
gh issue edit "$EPIC_E09" --repo "$REPO" --body "Durable runs: decompose, tier, route, control loop, fault check."$'\n\n## Issues\n'"$CHECK_E09" >/dev/null
echo "E09: epic #$EPIC_E09 with 7 issues"

# E10
EPIC_E10=$(new_issue '[E10] Intake' 'GitHub, Slack and CLI requests become Tickets.' 'epic,E10' 'Phase 1 - Intake, decomposition, gates')
CHECK_E10=""
B='Label agent or /agent comment from allowlisted users creates a Ticket.

## Acceptance
- [ ] Non-allowlisted user ignored

Part of #EPICNUM (E10). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E10}
N=$(new_issue '[E10.01] GitHub issue intake' "$B" 'E10,tier:2,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E10="$CHECK_E10- [ ] #$N"$'\n'
B='/agent slash command, plan and merge brief messages, approve buttons for Tier 0/4 plans.

## Acceptance
- [ ] Signed requests only

Part of #EPICNUM (E10). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E10}
N=$(new_issue '[E10.02] Slack intake and approvals' "$B" 'E10,tier:2,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E10="$CHECK_E10- [ ] #$N"$'\n'
B='/tier N, /approve, /reject, /revise, /retry, /stop, /teach, /repo, /ci-ok.

## Acceptance
- [ ] Each command audited with actor

Part of #EPICNUM (E10). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E10}
N=$(new_issue '[E10.03] Comment commands' "$B" 'E10,tier:2,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E10="$CHECK_E10- [ ] #$N"$'\n'
B='HMAC-signed webhook for external routines.

## Acceptance
- [ ] Bad signature rejected

Part of #EPICNUM (E10). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E10}
N=$(new_issue '[E10.04] Generic webhook (Grok Bot routines)' "$B" 'E10,tier:2,size:S' 'Phase 1 - Intake, decomposition, gates')
CHECK_E10="$CHECK_E10- [ ] #$N"$'\n'
gh issue edit "$EPIC_E10" --repo "$REPO" --body "GitHub, Slack and CLI requests become Tickets."$'\n\n## Issues\n'"$CHECK_E10" >/dev/null
echo "E10: epic #$EPIC_E10 with 4 issues"

# E11
EPIC_E11=$(new_issue '[E11] Tracks and communication' 'All tracks with the hub model and Security veto.' 'epic,E11' 'Phase 1 - Intake, decomposition, gates')
CHECK_E11=""
B='Writer and Reviewer prompts plus tools.yaml for frontend, mobile, design, devops, data, test, docs, security.

## Acceptance
- [ ] Golden eval tasks per track

Part of #EPICNUM (E11). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E11}
N=$(new_issue '[E11.01] Track prompts and tool allowlists' "$B" 'E11,tier:2,size:L' 'Phase 1 - Intake, decomposition, gates')
CHECK_E11="$CHECK_E11- [ ] #$N"$'\n'
B='Allowed direct links only; topic capped at 3 messages, then back to Orchestrator.

## Acceptance
- [ ] 4th message routed to Orchestrator

Part of #EPICNUM (E11). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E11}
N=$(new_issue '[E11.02] Hub messaging with 3-message cap' "$B" 'E11,tier:2,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E11="$CHECK_E11- [ ] #$N"$'\n'
B='Security review on Tier 2+; veto blocks tier-gate until resolved.

## Acceptance
- [ ] Veto blocks merge in tests

Part of #EPICNUM (E11). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E11}
N=$(new_issue '[E11.03] Security track and veto' "$B" 'E11,tier:3,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E11="$CHECK_E11- [ ] #$N"$'\n'
B='Project CI changes written only by best model, CI-only PRs, human review required.

## Acceptance
- [ ] Mixed CI and code PR blocked

Part of #EPICNUM (E11). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E11}
N=$(new_issue '[E11.04] Tier 0 devops writer' "$B" 'E11,tier:3,size:S' 'Phase 1 - Intake, decomposition, gates')
CHECK_E11="$CHECK_E11- [ ] #$N"$'\n'
gh issue edit "$EPIC_E11" --repo "$REPO" --body "All tracks with the hub model and Security veto."$'\n\n## Issues\n'"$CHECK_E11" >/dev/null
echo "E11: epic #$EPIC_E11 with 4 issues"

# E12
EPIC_E12=$(new_issue '[E12] Storage and observability' 'Postgres state, traces and cost reporting.' 'epic,E12' 'Phase 1 - Intake, decomposition, gates')
CHECK_E12=""
B='Migrations for tables in ARCHITECTURE Section 20; sqlc queries; org_id on every table.

## Acceptance
- [ ] Migrations up/down tested

Part of #EPICNUM (E12). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E12}
N=$(new_issue '[E12.01] Postgres schema and sqlc' "$B" 'E12,tier:3,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E12="$CHECK_E12- [ ] #$N"$'\n'
B='Spans per run, ticket, model call, tool call; export to Langfuse via OTLP.

## Acceptance
- [ ] Cost per run visible in Langfuse

Part of #EPICNUM (E12). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E12}
N=$(new_issue '[E12.02] OpenTelemetry tracing' "$B" 'E12,tier:2,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E12="$CHECK_E12- [ ] #$N"$'\n'
B='Prometheus metrics; Grafana dashboards: cost, first-pass rate, escalations, flaky checks.

## Acceptance
- [ ] Dashboard JSON in repo

Part of #EPICNUM (E12). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E12}
N=$(new_issue '[E12.03] Metrics and dashboards' "$B" 'E12,tier:1,size:S' 'Phase 1 - Intake, decomposition, gates')
CHECK_E12="$CHECK_E12- [ ] #$N"$'\n'
B='Slack message with spend by repo, tier and model.

## Acceptance
- [ ] Report matches DB totals

Part of #EPICNUM (E12). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E12}
N=$(new_issue '[E12.04] Daily cost report' "$B" 'E12,tier:1,size:S' 'Phase 1 - Intake, decomposition, gates')
CHECK_E12="$CHECK_E12- [ ] #$N"$'\n'
gh issue edit "$EPIC_E12" --repo "$REPO" --body "Postgres state, traces and cost reporting."$'\n\n## Issues\n'"$CHECK_E12" >/dev/null
echo "E12: epic #$EPIC_E12 with 4 issues"

# E13
EPIC_E13=$(new_issue '[E13] Templates v2 and argus init' 'Node and web templates plus the init command.' 'epic,E13' 'Phase 1 - Intake, decomposition, gates')
CHECK_E13=""
B='Node LTS + TypeScript + NestJS + pnpm + Prisma + Jest.

## Acceptance
- [ ] All verbs pass

Part of #EPICNUM (E13). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E13}
N=$(new_issue '[E13.01] be-node-ts template' "$B" 'E13,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E13="$CHECK_E13- [ ] #$N"$'\n'
B='Next.js App Router + TypeScript + Vitest + Playwright.

## Acceptance
- [ ] All verbs pass

Part of #EPICNUM (E13). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E13}
N=$(new_issue '[E13.02] fe-next template' "$B" 'E13,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E13="$CHECK_E13- [ ] #$N"$'\n'
B='Vue 3 + Vite + Pinia + Vitest + Playwright.

## Acceptance
- [ ] All verbs pass

Part of #EPICNUM (E13). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E13}
N=$(new_issue '[E13.03] fe-vue template' "$B" 'E13,tier:0,size:M' 'Phase 1 - Intake, decomposition, gates')
CHECK_E13="$CHECK_E13- [ ] #$N"$'\n'
B='Compose templates into monorepo or multi-repo, wire components, write profile and bootstrap commit; no model calls.

## Acceptance
- [ ] be-go + fe-next monorepo builds and tests

Part of #EPICNUM (E13). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E13}
N=$(new_issue '[E13.04] argus init command' "$B" 'E13,tier:2,size:L' 'Phase 1 - Intake, decomposition, gates')
CHECK_E13="$CHECK_E13- [ ] #$N"$'\n'
B='Parse JS test output into failing tests only.

## Acceptance
- [ ] Parser tests on samples

Part of #EPICNUM (E13). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E13}
N=$(new_issue '[E13.05] Test parsers: Jest/Vitest/Playwright' "$B" 'E13,tier:1,size:S' 'Phase 1 - Intake, decomposition, gates')
CHECK_E13="$CHECK_E13- [ ] #$N"$'\n'
gh issue edit "$EPIC_E13" --repo "$REPO" --body "Node and web templates plus the init command."$'\n\n## Issues\n'"$CHECK_E13" >/dev/null
echo "E13: epic #$EPIC_E13 with 5 issues"

# E14
EPIC_E14=$(new_issue '[E14] Existing repo onboarding' 'Connect repos that already exist.' 'epic,E14' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E14=""
B='Parse CI files, manifests, devcontainer, CODEOWNERS into a draft RepoProfile by code.

## Acceptance
- [ ] Profiles drafted for 6 sample repos

Part of #EPICNUM (E14). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E14}
N=$(new_issue '[E14.01] Repo discovery' "$B" 'E14,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E14="$CHECK_E14- [ ] #$N"$'\n'
B='Run verbs or detected commands; record baseline and known failures; report what is missing.

## Acceptance
- [ ] Broken repo gets a clear missing list

Part of #EPICNUM (E14). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E14}
N=$(new_issue '[E14.02] Sandbox verification and missing report' "$B" 'E14,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E14="$CHECK_E14- [ ] #$N"$'\n'
B='gitleaks over history; findings reported for human rotation.

## Acceptance
- [ ] Planted historical secret found

Part of #EPICNUM (E14). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E14}
N=$(new_issue '[E14.03] Full-history secret scan' "$B" 'E14,tier:3,size:S' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E14="$CHECK_E14- [ ] #$N"$'\n'
B='docs/overview.md, AGENTS.md, PR template, project CI calling central gates (Tier 0, human review).

## Acceptance
- [ ] 4 of 6 sample repos onboard

Part of #EPICNUM (E14). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E14}
N=$(new_issue '[E14.04] Onboarding PR' "$B" 'E14,tier:0,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E14="$CHECK_E14- [ ] #$N"$'\n'
gh issue edit "$EPIC_E14" --repo "$REPO" --body "Connect repos that already exist."$'\n\n## Issues\n'"$CHECK_E14" >/dev/null
echo "E14: epic #$EPIC_E14 with 4 issues"

# E15
EPIC_E15=$(new_issue '[E15] New project flow' 'From plain-text idea to a running walking skeleton.' 'epic,E15' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E15=""
B='Ask missing questions, write PRD to config repo.

## Acceptance
- [ ] PRD confirmed by requester

Part of #EPICNUM (E15). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E15}
N=$(new_issue '[E15.01] Discovery and PRD' "$B" 'E15,tier:3,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E15="$CHECK_E15- [ ] #$N"$'\n'
B='Opus drafts spec with templates, layout, milestones; Astra and Grok critique; human approval gate.

## Acceptance
- [ ] No repo work before approval

Part of #EPICNUM (E15). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E15}
N=$(new_issue '[E15.02] ProjectSpec drafting and approval' "$B" 'E15,tier:4,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E15="$CHECK_E15- [ ] #$N"$'\n'
B='Generate code, open bootstrap PR, verify profile, post protection settings.

## Acceptance
- [ ] Bootstrap PR green

Part of #EPICNUM (E15). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E15}
N=$(new_issue '[E15.03] Bootstrap with argus init' "$B" 'E15,tier:0,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E15="$CHECK_E15- [ ] #$N"$'\n'
B='Milestones to WorkTickets as GitHub issues; demo report and approval per milestone.

## Acceptance
- [ ] Walking skeleton merged for a sample PRD

Part of #EPICNUM (E15). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E15}
N=$(new_issue '[E15.04] Milestone backlog and demo reports' "$B" 'E15,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E15="$CHECK_E15- [ ] #$N"$'\n'
gh issue edit "$EPIC_E15" --repo "$REPO" --body "From plain-text idea to a running walking skeleton."$'\n\n## Issues\n'"$CHECK_E15" >/dev/null
echo "E15: epic #$EPIC_E15 with 4 issues"

# E16
EPIC_E16=$(new_issue '[E16] Code index and reuse' 'Find existing code before writing new code.' 'epic,E16' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E16=""
B='Symbols per language from templates; refresh changed files on push.

## Acceptance
- [ ] Index covers all template languages

Part of #EPICNUM (E16). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E16}
N=$(new_issue '[E16.01] tree-sitter symbol index' "$B" 'E16,tier:2,size:L' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E16="$CHECK_E16- [ ] #$N"$'\n'
B='pgvector + full-text search over symbols and summaries.

## Acceptance
- [ ] Search finds known helpers in fixtures

Part of #EPICNUM (E16). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E16}
N=$(new_issue '[E16.02] Embeddings and hybrid search' "$B" 'E16,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E16="$CHECK_E16- [ ] #$N"$'\n'
B='Add candidates to context packet; Writer must answer each; similarity check on new public functions.

## Acceptance
- [ ] Near-duplicate helper blocked in review

Part of #EPICNUM (E16). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E16}
N=$(new_issue '[E16.03] Reuse candidates and decisions' "$B" 'E16,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E16="$CHECK_E16- [ ] #$N"$'\n'
B='Cheap-model summaries keyed by file hash; only changed files summarized.

## Acceptance
- [ ] Second run makes no summary calls

Part of #EPICNUM (E16). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E16}
N=$(new_issue '[E16.04] File summary cache' "$B" 'E16,tier:1,size:S' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E16="$CHECK_E16- [ ] #$N"$'\n'
gh issue edit "$EPIC_E16" --repo "$REPO" --body "Find existing code before writing new code."$'\n\n## Issues\n'"$CHECK_E16" >/dev/null
echo "E16: epic #$EPIC_E16 with 4 issues"

# E17
EPIC_E17=$(new_issue '[E17] Scoring and routing' 'Scores decide which model does what.' 'epic,E17' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E17=""
B='ScoreRecord from model gateway usage; review survival matched by code; Adjudicator only for disputed findings.

## Acceptance
- [ ] 100% of merges scored

Part of #EPICNUM (E17). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E17}
N=$(new_issue '[E17.01] Score job with real usage' "$B" 'E17,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E17="$CHECK_E17- [ ] #$N"$'\n'
B='Rolling 30-merge averages per bucket; prior worth 10 merges from benchmarks.yaml.

## Acceptance
- [ ] Stats rebuildable from score files

Part of #EPICNUM (E17). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E17}
N=$(new_issue '[E17.02] Routing stats and benchmark prior' "$B" 'E17,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E17="$CHECK_E17- [ ] #$N"$'\n'
B='Pick by expected score minus cost; 10% exploration at Tiers 1-2; none at 0, 3, 4.

## Acceptance
- [ ] Property tests: allowlist, floor, cross-family

Part of #EPICNUM (E17). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E17}
N=$(new_issue '[E17.03] Score-based router' "$B" 'E17,tier:3,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E17="$CHECK_E17- [ ] #$N"$'\n'
B='Slack report of current routing table and changes with numbers.

## Acceptance
- [ ] Report generated from stats

Part of #EPICNUM (E17). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E17}
N=$(new_issue '[E17.04] Weekly routing report' "$B" 'E17,tier:1,size:S' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E17="$CHECK_E17- [ ] #$N"$'\n'
gh issue edit "$EPIC_E17" --repo "$REPO" --body "Scores decide which model does what."$'\n\n## Issues\n'"$CHECK_E17" >/dev/null
echo "E17: epic #$EPIC_E17 with 4 issues"

# E18
EPIC_E18=$(new_issue '[E18] Mobile templates' 'Flutter, React Native, Android and iOS.' 'epic,E18' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E18=""
B='Flutter stable + dart analyze + flutter test.

## Acceptance
- [ ] All verbs pass in Linux sandbox

Part of #EPICNUM (E18). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E18}
N=$(new_issue '[E18.01] mob-flutter template' "$B" 'E18,tier:0,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E18="$CHECK_E18- [ ] #$N"$'\n'
B='Expo + TypeScript + Jest + RN Testing Library.

## Acceptance
- [ ] All verbs pass

Part of #EPICNUM (E18). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E18}
N=$(new_issue '[E18.02] mob-react-native template' "$B" 'E18,tier:0,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E18="$CHECK_E18- [ ] #$N"$'\n'
B='Gradle, Kotlin default with Java option, ktlint/detekt or Checkstyle, JUnit.

## Acceptance
- [ ] All verbs pass

Part of #EPICNUM (E18). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E18}
N=$(new_issue '[E18.03] mob-android template' "$B" 'E18,tier:0,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E18="$CHECK_E18- [ ] #$N"$'\n'
B='SwiftUI + SPM + SwiftLint + XCTest; CI on macOS runners or Mac build host.

## Acceptance
- [ ] Verbs pass on macOS runner

Part of #EPICNUM (E18). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E18}
N=$(new_issue '[E18.04] mob-ios-swift template and macOS runners' "$B" 'E18,tier:0,size:L' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E18="$CHECK_E18- [ ] #$N"$'\n'
B='Failing tests only.

## Acceptance
- [ ] Parser tests on samples

Part of #EPICNUM (E18). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E18}
N=$(new_issue '[E18.05] Test parsers: Flutter, Gradle, XCTest' "$B" 'E18,tier:1,size:S' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E18="$CHECK_E18- [ ] #$N"$'\n'
gh issue edit "$EPIC_E18" --repo "$REPO" --body "Flutter, React Native, Android and iOS."$'\n\n## Issues\n'"$CHECK_E18" >/dev/null
echo "E18: epic #$EPIC_E18 with 5 issues"

# E19
EPIC_E19=$(new_issue '[E19] Cursor backend' 'Optional Writer backend through Cursor Cloud Agents API.' 'epic,E19' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E19=""
B='Cloud Agents API v1 over HTTPS (TODO(verify) fields); create run, poll status, fetch branch and usage.

## Acceptance
- [ ] Recorded-response tests

Part of #EPICNUM (E19). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E19}
N=$(new_issue '[E19.01] Cursor API client' "$B" 'E19,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E19="$CHECK_E19- [ ] #$N"$'\n'
B='Same contract, same gates, same scoring; behind a flag.

## Acceptance
- [ ] Composer 2.5 ticket reaches green gate

Part of #EPICNUM (E19). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E19}
N=$(new_issue '[E19.02] Map Cursor runs to WriterResult' "$B" 'E19,tier:2,size:M' 'Phase 2 - Onboarding, new projects, routing')
CHECK_E19="$CHECK_E19- [ ] #$N"$'\n'
gh issue edit "$EPIC_E19" --repo "$REPO" --body "Optional Writer backend through Cursor Cloud Agents API."$'\n\n## Issues\n'"$CHECK_E19" >/dev/null
echo "E19: epic #$EPIC_E19 with 2 issues"

# E20
EPIC_E20=$(new_issue '[E20] Multi-repo' 'Work across several repos in an org.' 'epic,E20' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E20=""
B='Edges from manifests, API clients and routes, events; nightly refresh.

## Acceptance
- [ ] Fixture org map correct

Part of #EPICNUM (E20). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E20}
N=$(new_issue '[E20.01] Org map builder' "$B" 'E20,tier:2,size:M' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E20="$CHECK_E20- [ ] #$N"$'\n'
B='Search + org map; ask when confidence < 0.7.

## Acceptance
- [ ] >= 18 of 20 benchmark requests resolved correctly

Part of #EPICNUM (E20). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E20}
N=$(new_issue '[E20.02] Repo resolver' "$B" 'E20,tier:2,size:M' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E20="$CHECK_E20- [ ] #$N"$'\n'
B='Linked PRs, provider before consumer, blocked consumer PRs, integration check.

## Acceptance
- [ ] Two-repo request merged in order

Part of #EPICNUM (E20). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E20}
N=$(new_issue '[E20.03] Change sets and merge order' "$B" 'E20,tier:3,size:L' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E20="$CHECK_E20- [ ] #$N"$'\n'
gh issue edit "$EPIC_E20" --repo "$REPO" --body "Work across several repos in an org."$'\n\n## Issues\n'"$CHECK_E20" >/dev/null
echo "E20: epic #$EPIC_E20 with 3 issues"

# E21
EPIC_E21=$(new_issue '[E21] Learning: skills and memory' 'Turn repeated lessons into reusable skills.' 'epic,E21' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E21=""
B='SKILL.md format, versions in git and DB, retrieval within token cap.

## Acceptance
- [ ] Top-k skills under 3k tokens

Part of #EPICNUM (E21). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E21}
N=$(new_issue '[E21.01] Skill registry' "$B" 'E21,tier:2,size:M' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E21="$CHECK_E21- [ ] #$N"$'\n'
B='Cluster repeated findings by code; model writes skill text only.

## Acceptance
- [ ] Drafts cite source runs

Part of #EPICNUM (E21). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E21}
N=$(new_issue '[E21.02] Learner drafting' "$B" 'E21,tier:2,size:M' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E21="$CHECK_E21- [ ] #$N"$'\n'
B='Replay 5-10 past tickets with and without a skill; promote only on lift; human approval for org scope.

## Acceptance
- [ ] No skill promoted without passing evals

Part of #EPICNUM (E21). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E21}
N=$(new_issue '[E21.03] Replay evaluation and promotion' "$B" 'E21,tier:3,size:L' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E21="$CHECK_E21- [ ] #$N"$'\n'
B='Facts per repo after 2+ occurrences or /teach.

## Acceptance
- [ ] No personal data stored

Part of #EPICNUM (E21). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E21}
N=$(new_issue '[E21.04] Project memory' "$B" 'E21,tier:2,size:S' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E21="$CHECK_E21- [ ] #$N"$'\n'
gh issue edit "$EPIC_E21" --repo "$REPO" --body "Turn repeated lessons into reusable skills."$'\n\n## Issues\n'"$CHECK_E21" >/dev/null
echo "E21: epic #$EPIC_E21 with 4 issues"

# E22
EPIC_E22=$(new_issue '[E22] Security hardening' 'Prove the safety claims.' 'epic,E22' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E22=""
B='Malicious issues, comments and repo files try out-of-scope tool calls.

## Acceptance
- [ ] 100% of out-of-scope calls blocked

Part of #EPICNUM (E22). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E22}
N=$(new_issue '[E22.01] Prompt-injection red-team suite' "$B" 'E22,tier:4,size:L' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E22="$CHECK_E22- [ ] #$N"$'\n'
B='Run sandboxes under gVisor runtime.

## Acceptance
- [ ] Benchmark still passes

Part of #EPICNUM (E22). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E22}
N=$(new_issue '[E22.02] gVisor sandboxes' "$B" 'E22,tier:3,size:M' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E22="$CHECK_E22- [ ] #$N"$'\n'
B='syft SBOM, cosign signatures, distroless images.

## Acceptance
- [ ] Unsigned image refused at deploy

Part of #EPICNUM (E22). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E22}
N=$(new_issue '[E22.03] SBOM and signed images' "$B" 'E22,tier:3,size:S' 'Phase 3 - Multi-repo, learning, hardening')
CHECK_E22="$CHECK_E22- [ ] #$N"$'\n'
gh issue edit "$EPIC_E22" --repo "$REPO" --body "Prove the safety claims."$'\n\n## Issues\n'"$CHECK_E22" >/dev/null
echo "E22: epic #$EPIC_E22 with 3 issues"

# E23
EPIC_E23=$(new_issue '[E23] Pilot rollout' 'Humans roll out to one real repo, then expand.' 'epic,E23' 'Phase 4 - Pilot on a real team'"'"'s repos')
CHECK_E23=""
B='Install Apps, rulesets, central gates on one pilot repo (human task with checklist).

## Acceptance
- [ ] Checklist complete

Part of #EPICNUM (E23). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E23}
N=$(new_issue '[E23.01] Pilot repo selection and setup' "$B" 'E23,tier:4,size:S' 'Phase 4 - Pilot on a real team'"'"'s repos')
CHECK_E23="$CHECK_E23- [ ] #$N"$'\n'
B='First-pass rate, reverts, cost per merged ticket, time to merge.

## Acceptance
- [ ] Report each month

Part of #EPICNUM (E23). See ARCHITECTURE.md and docs/BACKLOG.md.'; B=${B//EPICNUM/$EPIC_E23}
N=$(new_issue '[E23.02] Monthly metrics review' "$B" 'E23,tier:1,size:S' 'Phase 4 - Pilot on a real team'"'"'s repos')
CHECK_E23="$CHECK_E23- [ ] #$N"$'\n'
gh issue edit "$EPIC_E23" --repo "$REPO" --body "Humans roll out to one real repo, then expand."$'\n\n## Issues\n'"$CHECK_E23" >/dev/null
echo "E23: epic #$EPIC_E23 with 2 issues"

echo "Done: 23 epics, 107 issues."
