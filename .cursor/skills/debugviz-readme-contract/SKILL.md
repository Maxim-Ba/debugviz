---
name: debugviz-readme-contract
description: >-
  Enforces DebugViz README-driven development: README is the MVP contract,
  BACKLOG traceability, two integration paths (wire vs manual). Use when
  implementing features, changing CLI/API, adding endpoints, or before closing
  any backlog issue.
---

# DebugViz — README contract

## When to apply

- Starting or finishing any issue from docs/BACKLOG.md
- Changing CLI flags, runtime API, schemas, or UI behavior
- User asks "does this match the spec?"

## Workflow

1. Read the matching section in [README.md](../../README.md) and acceptance criteria in [docs/MVP_PLAN.md](../../docs/MVP_PLAN.md).
2. Identify issue ID (e.g. 1.2, 4.2, 9.3) and README section from BACKLOG traceability table.
3. Implement only what the README promises for that issue — no scope creep.
4. After implementation, verify with the README "Ожидаемый результат" / Quick Start command.
5. Update [docs/BACKLOG.md](../../docs/BACKLOG.md): issue status `[x]` or `[~]`, DoD if applicable.

## Non-negotiables

| Topic | Contract |
|-------|----------|
| Module path | `github.com/Maxim-Ba/debugviz` |
| Entry model | `entry_point` with `kind`: http, grpc, cli, worker |
| Recommended DX | `debugviz.yaml` + `go generate` (wire + instrument) |
| Manual DX | `Configure` + `HTTPMiddleware` / interceptors / `RunCLI` / `RunJob` |
| Target audience | Any Go app — cv-backend is HTTP **example**, not the only target |

## Two integration paths

- **Recommended (M6+):** developer writes yaml + `//go:generate`; `debugviz wire` injects Configure + entry hooks; `instrument` injects inner spans.
- **Advanced (M2):** explicit calls in `main` — escape hatch for custom middleware order.

Do not remove Manual path from docs when adding wire. Wire calls the same runtime API.

## Definition of Done (check before claiming MVP progress)

- [ ] README promise demonstrated (command + expected UI/server behavior)
- [ ] BACKLOG issue status updated
- [ ] Schemas/types synced if graph or trace shape changed
- [ ] No chi-only assumptions unless issue is explicitly HTTP-adapter scoped

## Anti-patterns

- Implementing gin/gRPC without updating README limitations or issue scope
- Adding API not documented in README without updating README first
- Hardcoding cv-backend paths as the only integration story
