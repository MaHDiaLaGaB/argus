# argus backlog: epics and issues

Generated from one definition; `scripts/create-github-issues.sh` creates the same items in GitHub.
Tier = the pipeline tier the work itself would get. Size: S (< 1 day), M (1-3 days), L (3-5 days).

**23 epics, 107 issues, 3 done.**

| Epic | Phase | Issues |
|---|---|---|
| E01 Foundation and contracts | 0 | 6 |
| E02 Model clients (plain HTTPS) | 0 | 8 |
| E03 Sandbox runner | 0 | 5 |
| E04 Agents core | 0 | 7 |
| E05 GitHub layer | 0 | 7 |
| E06 Templates v1 (be-go, be-fastapi) | 0 | 4 |
| E07 Phase 0 CLI and acceptance | 0 | 4 |
| E08 Pipeline CI gates (build once, freeze) | 1 | 7 |
| E09 Orchestrator and workflows | 1 | 7 |
| E10 Intake | 1 | 4 |
| E11 Tracks and communication | 1 | 4 |
| E12 Storage and observability | 1 | 4 |
| E13 Templates v2 and argus init | 1 | 5 |
| E14 Existing repo onboarding | 2 | 4 |
| E15 New project flow | 2 | 4 |
| E16 Code index and reuse | 2 | 4 |
| E17 Scoring and routing | 2 | 4 |
| E18 Mobile templates | 2 | 5 |
| E19 Cursor backend | 2 | 2 |
| E20 Multi-repo | 3 | 3 |
| E21 Learning: skills and memory | 3 | 4 |
| E22 Security hardening | 3 | 3 |
| E23 Pilot rollout | 4 | 2 |

## Phase 0 - One ticket end to end

Go skeleton, contracts, model clients, sandbox, Writer + Reviewer, GitHub layer, first templates. Accept: 6 of 10 benchmark tickets reach a green tier-gate with no human edits.

### E01: Foundation and contracts

Repo skeleton, contracts every component speaks, config and policy loading.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E01.01 | **Go repo skeleton**: go.mod, Makefile verbs (setup build test test-one lint fmt run), AGENTS.md, README, CI on master with golangci-lint, govulncheck. | 1 | S | done | make lint test build pass in CI; Branch master protected |
| E01.02 | **Contracts package**: Ticket, WorkTicket, TaskEnvelope, WriterResult, ReviewReport, GateResult, MergeBrief, ScoreRecord with strict decoding and validation; tier-gate evaluation and score math in code. | 2 | M | done | Valid fixtures pass, invalid cases rejected by table tests; CLI validate and gate commands |
| E01.03 | **Generate JSON Schemas from contracts**: Reflection-based generator writes contracts/schema/*.json; CI fails if generated files are stale. | 2 | S | done | One schema per contract kind; CI check for drift |
| E01.04 | **Config and policy loader**: Load config/*.yaml and policy/*.yaml (models, pricing, routing, allowlist, tiers, caps, human_review_tiers); validate at startup; refuse to start if invalid. | 2 | M | todo | Invalid config stops startup with a clear error; Policy is read-only at runtime |
| E01.05 | **Model, price and allowlist config v1**: config/models.yaml, pricing.yaml, benchmarks.yaml and policy/allowlist.yaml filled from ARCHITECTURE Section 9 with TODO(verify) markers. | 1 | S | todo | Every model in allowlist has a price; Tier floors encoded |
| E01.06 | **Secret scanning and supply chain for this repo**: gitleaks in CI, gosec, Dependabot for Go modules and Actions, pinned action SHAs. | 0 | S | todo | CI fails on a planted fake secret |

### E02: Model clients (plain HTTPS)

Thin clients for Claude, GPT and Grok with budgets, pricing and logging. No AI frameworks.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E02.01 | **LLM client interface**: internal/llm: Complete(ctx, Request) Response with messages, tools (JSON schema), max tokens, cache hints, run/step ids. Normalized usage. | 2 | S | todo | Interface documented; Fake client for tests |
| E02.02 | **Anthropic client**: Messages API over net/http: tool use, prompt caching, usage parsing, retries with backoff on 429/5xx. | 2 | M | todo | Recorded-response tests cost $0; Tool calls round-trip |
| E02.03 | **OpenAI client**: Responses/Chat API over net/http with tool calling and usage parsing. | 2 | M | todo | Recorded-response tests; Tool calls round-trip |
| E02.04 | **xAI client**: Grok API (OpenAI-compatible endpoint, TODO(verify)) reusing the OpenAI wire code. | 2 | S | todo | Recorded-response tests |
| E02.05 | **Pricing and token accounting**: USD per call from pricing.yaml incl. cached tokens; every call logged (provider, model, tokens, usd, latency, status). | 2 | S | todo | Cost matches hand calculation in tests |
| E02.06 | **Budget hard stop**: Hierarchical budgets (org/month > repo/month > ticket > step); pre-call estimate; reject when it would exceed; alert at 80%. | 2 | M | todo | $0.05 cap test stops the run; Alert event emitted at 80% |
| E02.07 | **Fallback chain**: On 429/5xx/timeout: retry, then next model in fallbacks, never below tier floor, keep cross-family rule. | 2 | M | todo | Outage simulation falls back correctly; Property test: never below floor |
| E02.08 | **Response recorder for tests**: Record/replay HTTP fixtures so tests never call real APIs; secrets stripped from recordings. | 1 | S | todo | No API key in any fixture (grep test) |

### E03: Sandbox runner

Run any template or repo's make verbs in Docker and return parsed results.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E03.01 | **Docker sandbox lifecycle**: Create rootless container per run from the repo profile image; mount /work only; CPU/mem/time limits; egress allowlist; destroy after run. | 3 | M | todo | No Docker socket inside; Limits enforced in tests |
| E03.02 | **Make verb runner**: Run make setup/build/test/test-one/lint with timeouts; capture exit codes and logs; truncate output with pointer to full log. | 2 | S | todo | Works on be-go and be-fastapi templates |
| E03.03 | **Test output parsers: Go and pytest**: Parse go test -json and pytest JUnit XML into failing tests + relevant stack lines only. | 1 | S | todo | Only failing tests reach the model |
| E03.04 | **Test proof check**: Run proof test on parent commit (must fail) and on change (must pass). | 2 | S | todo | Proof result matches WriterResult.proof |
| E03.05 | **Deterministic ticket check**: Apply WriterResult.CheckAgainst (scope, line cap, proof, new deps) before any review. | 1 | S | todo | Violations block review |

### E04: Agents core

One small tool-use loop and a policy engine shared by every LLM agent.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E04.01 | **Tool-use loop**: In-house loop over llm client: send, execute tool calls, feed results, stop on done/budget/max turns. | 2 | M | todo | Loop is under 400 lines; Deterministic tests with fake client |
| E04.02 | **Tool registry and policy engine**: Typed tools with JSON schemas; every call checked against envelope permissions and scope files; blocked calls logged. | 3 | M | todo | Out-of-scope write is blocked; Reviewer cannot write |
| E04.03 | **Context packet builder**: Build packet from WorkTicket: rules refs at pinned SHA, file summaries, reuse candidates, prior attempts; enforce token budgets (6k/3k/1.5k/1.5k). | 2 | M | todo | Packet under 6k tokens on fixtures |
| E04.04 | **Writer agent (backend track)**: Prompts/tracks/backend/writer.md + loop: edit files in scope, run tests, commit, return WriterResult. | 2 | M | todo | Produces valid WriterResult on fixture tickets |
| E04.05 | **Reviewer agent**: Prompts/tracks/backend/reviewer.md: diff + static analysis in, ReviewReport out; runs only after CI is green. | 2 | M | todo | Cross-family enforced; Valid ReviewReport |
| E04.06 | **Model output validation**: Every model output parsed into a contract; invalid output retried once with the error, then failed. | 1 | S | todo | Invalid JSON never enters pipeline |
| E04.07 | **docs/model-calls.md enforcement**: Test that fails if a call site to internal/llm is not listed in docs/model-calls.md. | 1 | S | todo | Unlisted call site fails CI |

### E05: GitHub layer

Separate Writer, Reviewer and Gate identities; PRs, checks and merge brief.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E05.01 | **Create GitHub Apps (test org)**: argus-writer, argus-reviewer, argus-gate with least permissions (ARCHITECTURE 11.1). Manual setup checklist in docs. | 3 | S | todo | Apps installed on test org only |
| E05.02 | **Installation token minting**: JWT -> installation token, 1h, cached, never visible to models or logs. | 3 | S | todo | Token never in traces (grep test) |
| E05.03 | **Branch push and PR creation**: Push agent/* branch, open PR with template: scope list, ACs, evidence, cost. | 2 | M | todo | PR authored by argus-writer |
| E05.04 | **Reviewer posting**: Submit review and ai-review / security-review check runs as argus-reviewer. | 2 | S | todo | Review authored by argus-reviewer |
| E05.05 | **tier-gate check run**: Collect CI results, review, security, human approval; run GateResult.Evaluate; post check run as argus-gate. | 3 | M | todo | Tier 0 and 4 blocked without human approval |
| E05.06 | **Merge brief**: Fill MergeBrief from data; cheap model only for one_line and risk_notes; post as PR comment. | 1 | S | todo | Valid MergeBrief on every green PR |
| E05.07 | **Read check results and webhooks**: Receive check_run / pull_request / review events; verify signatures; dedupe. | 3 | M | todo | Forged webhook rejected |

### E06: Templates v1 (be-go, be-fastapi)

First two templates with the shared verbs and ready profiles.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E06.01 | **Shared template parts**: Makefile verbs, AGENTS.base.md, PR template, gitleaks config, Dependabot, .env.example. | 0 | S | todo | Same verbs in every template |
| E06.02 | **be-go template**: Go + chi + pgx/sqlc, health endpoint, sample feature with tests, devcontainer, Dockerfile, CI calling central gates, .argus/profile.yaml. | 0 | M | todo | make setup build lint test passes on a fresh copy |
| E06.03 | **be-fastapi template**: Python + FastAPI + uv + SQLAlchemy/Alembic + ruff + mypy + pytest, same structure as be-go. | 0 | M | todo | make setup build lint test passes on a fresh copy |
| E06.04 | **Template self-test CI**: argus-templates CI generates each template and runs all verbs. | 0 | S | todo | Broken template fails CI |

### E07: Phase 0 CLI and acceptance

Tie it together and prove it on real tickets.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E07.01 | **argus run command**: argus run --repo owner/name --work-ticket file.json: sandbox -> Writer -> checks -> PR -> Reviewer -> tier-gate -> merge brief. | 2 | M | todo | One fixture ticket reaches green gate on test org |
| E07.02 | **Benchmark ticket set**: 10 Tier 1-2 WorkTickets on be-go and be-fastapi sample repos, plus 5 deliberately bad attempts. | 1 | S | todo | Bad attempts blocked by scope, cap or proof |
| E07.03 | **No-secrets test**: Automated grep across prompts, traces, logs and fixtures for keys and tokens. | 2 | S | todo | Planted secret detected |
| E07.04 | **Phase 0 acceptance run and report**: Run benchmark, record pass rate, cost, time; sign-off doc. | 1 | S | todo | >= 6 of 10 green with no human edits; Report in docs/reports/ |

## Phase 1 - Intake, decomposition, gates

Frozen pipeline CI gates, Temporal orchestrator, intake, all tracks, more templates.

### E08: Pipeline CI gates (build once, freeze)

Central reusable workflows in argus-gates, proven by the gate test suite, then frozen.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E08.01 | **Scope and line-cap gates**: Reusable workflow reading scope list from PR and tier label; fails on out-of-scope files or over cap. | 0 | M | todo | Gate test fixtures fail as expected |
| E08.02 | **Test proof gate**: Run proof test on parent and head; must fail then pass. | 0 | M | todo | Fixture without fail-before is blocked |
| E08.03 | **Secrets, dependency and CI-path gates**: gitleaks, dependency audit, new-dependency detection by tier, block code PRs touching CI paths. | 0 | M | todo | Each fixture blocked |
| E08.04 | **Duplication gate**: jscpd (or per-stack tool, TODO(verify)) on changed code; threshold from policy. | 0 | M | todo | Duplicated block fixture blocked |
| E08.05 | **AI review reporter and score job**: CI step reports the Reviewer agent verdict (no second model); score job appends ScoreRecord on merge. | 0 | M | todo | No model call from CI; Score line valid |
| E08.06 | **Gate test suite**: argus-gates/gate-tests: known-bad PR fixtures; all must fail. | 0 | M | todo | Suite green = every fixture blocked |
| E08.07 | **Pin and freeze gates**: Version tag, repos call pinned version, rulesets block agent writes to gates and policy. | 4 | S | todo | Writer App push to gates rejected |

### E09: Orchestrator and workflows

Durable runs: decompose, tier, route, control loop, fault check.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E09.01 | **Temporal setup**: Self-hosted Temporal in docker compose; Go worker; run workflow skeleton with signals for plan approval, human approval, merged. | 3 | M | todo | Run survives worker restart |
| E09.02 | **Tier classifier**: Rules from policy/tiers.yaml first; cheap model only when rules return unknown. | 2 | M | todo | Unit tests per tier; Model not called when rules decide |
| E09.03 | **Decomposer**: Opus splits request into WorkTickets; code validates threshold rule, scope files exist, no shared files in parallel, topological order. | 3 | L | todo | Invalid split rejected and retried; Plan posted to issue |
| E09.04 | **Static router**: Pick writer/reviewer per tier from routing.yaml; cross-family; Tier 0 pinned to best model. | 2 | S | todo | Property test: reviewer family != writer family |
| E09.05 | **Control loop and two strikes**: Attempts, revise with findings, escalate model and tier after 2 failures, max 1 escalation, then human. | 3 | M | todo | Loop matches ARCHITECTURE 6.5 in tests |
| E09.06 | **Fault check and ci-suspect**: Rerun, base-branch run, sandbox repro, cross-family code question; flag ci-suspect with CI diagnosis; /ci-ok command. | 3 | M | todo | Planted CI fault flagged, planted code fault not |
| E09.07 | **grew_too_big re-split**: Writer can return grew_too_big; Decomposer re-splits without penalty. | 2 | S | todo | Re-split tickets pass validation |

### E10: Intake

GitHub, Slack and CLI requests become Tickets.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E10.01 | **GitHub issue intake**: Label agent or /agent comment from allowlisted users creates a Ticket. | 2 | M | todo | Non-allowlisted user ignored |
| E10.02 | **Slack intake and approvals**: /agent slash command, plan and merge brief messages, approve buttons for Tier 0/4 plans. | 2 | M | todo | Signed requests only |
| E10.03 | **Comment commands**: /tier N, /approve, /reject, /revise, /retry, /stop, /teach, /repo, /ci-ok. | 2 | M | todo | Each command audited with actor |
| E10.04 | **Generic webhook (Grok Bot routines)**: HMAC-signed webhook for external routines. | 2 | S | todo | Bad signature rejected |

### E11: Tracks and communication

All tracks with the hub model and Security veto.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E11.01 | **Track prompts and tool allowlists**: Writer and Reviewer prompts plus tools.yaml for frontend, mobile, design, devops, data, test, docs, security. | 2 | L | todo | Golden eval tasks per track |
| E11.02 | **Hub messaging with 3-message cap**: Allowed direct links only; topic capped at 3 messages, then back to Orchestrator. | 2 | M | todo | 4th message routed to Orchestrator |
| E11.03 | **Security track and veto**: Security review on Tier 2+; veto blocks tier-gate until resolved. | 3 | M | todo | Veto blocks merge in tests |
| E11.04 | **Tier 0 devops writer**: Project CI changes written only by best model, CI-only PRs, human review required. | 3 | S | todo | Mixed CI and code PR blocked |

### E12: Storage and observability

Postgres state, traces and cost reporting.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E12.01 | **Postgres schema and sqlc**: Migrations for tables in ARCHITECTURE Section 20; sqlc queries; org_id on every table. | 3 | M | todo | Migrations up/down tested |
| E12.02 | **OpenTelemetry tracing**: Spans per run, ticket, model call, tool call; export to Langfuse via OTLP. | 2 | M | todo | Cost per run visible in Langfuse |
| E12.03 | **Metrics and dashboards**: Prometheus metrics; Grafana dashboards: cost, first-pass rate, escalations, flaky checks. | 1 | S | todo | Dashboard JSON in repo |
| E12.04 | **Daily cost report**: Slack message with spend by repo, tier and model. | 1 | S | todo | Report matches DB totals |

### E13: Templates v2 and argus init

Node and web templates plus the init command.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E13.01 | **be-node-ts template**: Node LTS + TypeScript + NestJS + pnpm + Prisma + Jest. | 0 | M | todo | All verbs pass |
| E13.02 | **fe-next template**: Next.js App Router + TypeScript + Vitest + Playwright. | 0 | M | todo | All verbs pass |
| E13.03 | **fe-vue template**: Vue 3 + Vite + Pinia + Vitest + Playwright. | 0 | M | todo | All verbs pass |
| E13.04 | **argus init command**: Compose templates into monorepo or multi-repo, wire components, write profile and bootstrap commit; no model calls. | 2 | L | todo | be-go + fe-next monorepo builds and tests |
| E13.05 | **Test parsers: Jest/Vitest/Playwright**: Parse JS test output into failing tests only. | 1 | S | todo | Parser tests on samples |

## Phase 2 - Onboarding, new projects, routing

Existing repos, new projects with argus init, code index and reuse, scoring and routing, mobile templates, Cursor backend.

### E14: Existing repo onboarding

Connect repos that already exist.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E14.01 | **Repo discovery**: Parse CI files, manifests, devcontainer, CODEOWNERS into a draft RepoProfile by code. | 2 | M | todo | Profiles drafted for 6 sample repos |
| E14.02 | **Sandbox verification and missing report**: Run verbs or detected commands; record baseline and known failures; report what is missing. | 2 | M | todo | Broken repo gets a clear missing list |
| E14.03 | **Full-history secret scan**: gitleaks over history; findings reported for human rotation. | 3 | S | todo | Planted historical secret found |
| E14.04 | **Onboarding PR**: docs/overview.md, AGENTS.md, PR template, project CI calling central gates (Tier 0, human review). | 0 | M | todo | 4 of 6 sample repos onboard |

### E15: New project flow

From plain-text idea to a running walking skeleton.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E15.01 | **Discovery and PRD**: Ask missing questions, write PRD to config repo. | 3 | M | todo | PRD confirmed by requester |
| E15.02 | **ProjectSpec drafting and approval**: Opus drafts spec with templates, layout, milestones; Astra and Grok critique; human approval gate. | 4 | M | todo | No repo work before approval |
| E15.03 | **Bootstrap with argus init**: Generate code, open bootstrap PR, verify profile, post protection settings. | 0 | M | todo | Bootstrap PR green |
| E15.04 | **Milestone backlog and demo reports**: Milestones to WorkTickets as GitHub issues; demo report and approval per milestone. | 2 | M | todo | Walking skeleton merged for a sample PRD |

### E16: Code index and reuse

Find existing code before writing new code.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E16.01 | **tree-sitter symbol index**: Symbols per language from templates; refresh changed files on push. | 2 | L | todo | Index covers all template languages |
| E16.02 | **Embeddings and hybrid search**: pgvector + full-text search over symbols and summaries. | 2 | M | todo | Search finds known helpers in fixtures |
| E16.03 | **Reuse candidates and decisions**: Add candidates to context packet; Writer must answer each; similarity check on new public functions. | 2 | M | todo | Near-duplicate helper blocked in review |
| E16.04 | **File summary cache**: Cheap-model summaries keyed by file hash; only changed files summarized. | 1 | S | todo | Second run makes no summary calls |

### E17: Scoring and routing

Scores decide which model does what.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E17.01 | **Score job with real usage**: ScoreRecord from model gateway usage; review survival matched by code; Adjudicator only for disputed findings. | 2 | M | todo | 100% of merges scored |
| E17.02 | **Routing stats and benchmark prior**: Rolling 30-merge averages per bucket; prior worth 10 merges from benchmarks.yaml. | 2 | M | todo | Stats rebuildable from score files |
| E17.03 | **Score-based router**: Pick by expected score minus cost; 10% exploration at Tiers 1-2; none at 0, 3, 4. | 3 | M | todo | Property tests: allowlist, floor, cross-family |
| E17.04 | **Weekly routing report**: Slack report of current routing table and changes with numbers. | 1 | S | todo | Report generated from stats |

### E18: Mobile templates

Flutter, React Native, Android and iOS.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E18.01 | **mob-flutter template**: Flutter stable + dart analyze + flutter test. | 0 | M | todo | All verbs pass in Linux sandbox |
| E18.02 | **mob-react-native template**: Expo + TypeScript + Jest + RN Testing Library. | 0 | M | todo | All verbs pass |
| E18.03 | **mob-android template**: Gradle, Kotlin default with Java option, ktlint/detekt or Checkstyle, JUnit. | 0 | M | todo | All verbs pass |
| E18.04 | **mob-ios-swift template and macOS runners**: SwiftUI + SPM + SwiftLint + XCTest; CI on macOS runners or Mac build host. | 0 | L | todo | Verbs pass on macOS runner |
| E18.05 | **Test parsers: Flutter, Gradle, XCTest**: Failing tests only. | 1 | S | todo | Parser tests on samples |

### E19: Cursor backend

Optional Writer backend through Cursor Cloud Agents API.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E19.01 | **Cursor API client**: Cloud Agents API v1 over HTTPS (TODO(verify) fields); create run, poll status, fetch branch and usage. | 2 | M | todo | Recorded-response tests |
| E19.02 | **Map Cursor runs to WriterResult**: Same contract, same gates, same scoring; behind a flag. | 2 | M | todo | Composer 2.5 ticket reaches green gate |

## Phase 3 - Multi-repo, learning, hardening

Org map and change sets, skills and memory, security hardening.

### E20: Multi-repo

Work across several repos in an org.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E20.01 | **Org map builder**: Edges from manifests, API clients and routes, events; nightly refresh. | 2 | M | todo | Fixture org map correct |
| E20.02 | **Repo resolver**: Search + org map; ask when confidence < 0.7. | 2 | M | todo | >= 18 of 20 benchmark requests resolved correctly |
| E20.03 | **Change sets and merge order**: Linked PRs, provider before consumer, blocked consumer PRs, integration check. | 3 | L | todo | Two-repo request merged in order |

### E21: Learning: skills and memory

Turn repeated lessons into reusable skills.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E21.01 | **Skill registry**: SKILL.md format, versions in git and DB, retrieval within token cap. | 2 | M | todo | Top-k skills under 3k tokens |
| E21.02 | **Learner drafting**: Cluster repeated findings by code; model writes skill text only. | 2 | M | todo | Drafts cite source runs |
| E21.03 | **Replay evaluation and promotion**: Replay 5-10 past tickets with and without a skill; promote only on lift; human approval for org scope. | 3 | L | todo | No skill promoted without passing evals |
| E21.04 | **Project memory**: Facts per repo after 2+ occurrences or /teach. | 2 | S | todo | No personal data stored |

### E22: Security hardening

Prove the safety claims.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E22.01 | **Prompt-injection red-team suite**: Malicious issues, comments and repo files try out-of-scope tool calls. | 4 | L | todo | 100% of out-of-scope calls blocked |
| E22.02 | **gVisor sandboxes**: Run sandboxes under gVisor runtime. | 3 | M | todo | Benchmark still passes |
| E22.03 | **SBOM and signed images**: syft SBOM, cosign signatures, distroless images. | 3 | S | todo | Unsigned image refused at deploy |

## Phase 4 - Pilot on a real team's repos

Humans roll out to one pilot repo, then expand.

### E23: Pilot rollout

Humans roll out to one real repo, then expand.

| # | Issue | Tier | Size | Status | Acceptance |
|---|---|---|---|---|---|
| E23.01 | **Pilot repo selection and setup**: Install Apps, rulesets, central gates on one pilot repo (human task with checklist). | 4 | S | todo | Checklist complete |
| E23.02 | **Monthly metrics review**: First-pass rate, reverts, cost per merged ticket, time to merge. | 1 | S | todo | Report each month |
