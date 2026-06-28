---
name: debugviz-issue-workflow
description: >-
  Runs a DebugViz backlog issue end-to-end: read MVP_PLAN acceptance criteria,
  minimal implementation, verify README promise, update BACKLOG status. Use when
  user says "implement issue X.Y", "start M0/M1", or "close backlog task".
---

# DebugViz — issue workflow

## When to apply

- User references issue ID (0.1, 1.2, 9.3, M2, Epic 4, etc.)
- Starting implementation phase after docs-only phase
- Closing a milestone or marking BACKLOG `[x]`

## Steps

### 1. Load issue context

```
docs/BACKLOG.md     — priority, estimate, status, README link
docs/MVP_PLAN.md    — full acceptance criteria
README.md           — user-facing contract to satisfy
```

Read the **debugviz-readme-contract** skill if API or UX is involved.

### 2. Plan minimal scope

- One issue = one PR-sized change when possible.
- List files to create/modify; avoid touching unrelated epics.
- If issue depends on unfinished work (e.g. 9.3 needs 4.2), stop and implement dependency first or stub with clear TODO.

### 3. Implement

- Follow `.cursor/rules/debugviz-go.mdc` or monorepo rule for stack.
- Add tests required by MVP_PLAN acceptance criteria.
- For codegen/scanner: golden tests mandatory.

### 4. Verify (README promise)

| Issue type | Verify with |
|------------|-------------|
| 0.x foundation | `make dev` / `docker compose up` / CI commands |
| 1.x scan | `debugviz scan ./demo/http -o /tmp/g.json` |
| 2.x server | curl POST /api/graph, WS connect |
| 4.x runtime | curl/grpcurl + spans in server |
| 3.x UI | open :3000, visual or Playwright (6.3) |
| 5.x instrument | `debugviz instrument --dry-run` then `--write`, `go test` |
| 9.x wire | `debugviz wire --dry-run`, golden diff, demo without manual debugviz in main |

### 5. Update backlog

In [docs/BACKLOG.md](../../docs/BACKLOG.md):

- Set issue row to `[x]` when fully done, `[~]` when partial.
- Update Definition of Done rows if criteria met.
- Do **not** mark 7.2 unless README itself changed.

### 6. Report to user

Include:

- Issue ID and what acceptance criteria passed
- Exact commands run
- What's left for follow-up issues

## Milestone order (do not skip critical path)

```
M0 → M1 → M2 → M3
         ↘ M5 (parallel after M2 HTTP stable)
M3 + M2 → M6 (wire)
→ M4 polish
```

Critical path (MVP_PLAN): `0.2 → 1.1 → 1.2 → 1.4 → 2.1 → 3.1 → 4.1 → 4.2 → 3.4 → 5.2 → 9.3 → 6.1`

## P2 deferral

Issues 1.6, 1.7, 4.7, 4.8, 8.2, 8.3, 5.5 may slip after HTTP demo works — document deferral in BACKLOG comment or user message, do not block M2/M4 on them.

## Commits

Create git commits only when user explicitly asks.
