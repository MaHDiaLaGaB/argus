# AGENTS.md: rules for anyone (human or AI) building argus

1. Read `ARCHITECTURE.md` Section 0 first. Build phase by phase; never start the next phase before the current one is accepted.
2. Language: Go only. Standard library first. A new third-party module needs a written reason in the PR.
3. No AI frameworks (LangChain, LangGraph, LlamaIndex or similar). Model calls are plain HTTPS through `internal/llm`.
4. Code first, AI last. If code can do it, code does it. Every model call site is listed in `docs/model-calls.md`.
5. Contracts in `internal/contracts` are law. Changing a field, enum or rule needs human approval and a schema version bump.
6. Tests first. Every package ships with table tests. `make lint test` must pass before a PR.
7. Same verbs everywhere: `make setup build test test-one lint fmt run`.
8. Branch is `master`. Never hardcode a branch name for connected repos; read it from the repo profile.
9. No secrets in code, tests, fixtures, logs or prompts. No personal tokens for bots. GitHub Apps only.
10. Policy (`policy/`) and the central gate workflows are human-only. Never edit them in a code PR.
11. Small PRs: one outcome, named files, inside the tier line cap.
12. Model IDs and prices live in `config/`, never in code. Unverified values are marked `TODO(verify)`.
