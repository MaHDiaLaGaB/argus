# argus

Argus is a self-hosted AI development pipeline. An orchestrator and track agents turn GitHub issues or plain-text requests into small, reviewed, test-proven pull requests. Deterministic gates check every PR, a model from a different family reviews it, and humans merge.

Named after Argus Panoptes, the hundred-eyed watchman of Greek myth: nothing reaches `master` unwatched.

See `ARCHITECTURE.md` for the design and `docs/BACKLOG.md` for the build plan.

## Status
Phase 0, step 1–2: repo skeleton and contracts.

## Commands
```
make setup     # download modules
make build     # bin/argus
make test      # all tests with the race detector
make lint      # gofmt check, go vet, golangci-lint if installed
make schema    # regenerate contracts/schema/*.json (make test fails if stale)
make run ARGS="validate --kind workticket internal/contracts/testdata/valid/workticket.json"
```

## CLI
```
argus version
argus validate --kind <ticket|workticket|envelope|writer-result|review|gate|merge-brief|score> <file.json>
argus gate gate.json                      # evaluates the tier-gate decision
argus schema --kind ticket                # print one generated JSON Schema
argus schema --out contracts/schema       # write every schema
argus config check [--root .]             # load and validate config/ and policy/
```

## Layout
```
cmd/argus/           CLI
internal/contracts/    contracts, validation, tier-gate and score math, schema generator
contracts/schema/      generated JSON Schemas (one per kind; do not edit)
internal/config/       loader and validation for config/*.yaml and policy/*.yaml
docs/model-calls.md    every place a model is called, with reason and token budget
policy/  config/       human-owned policy, model and price config (added in later steps)
```
