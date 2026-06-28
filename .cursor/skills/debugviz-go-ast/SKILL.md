---
name: debugviz-go-ast
description: >-
  Implements DebugViz Go static analysis and codegen using go/packages, go/ast,
  go/types, go/format. Use for scanner adapters, debugviz scan, instrument,
  wire, golden tests, or EntryDiscoverer in go/pkg/.
---

# DebugViz — Go AST & codegen

## When to apply

- Issues 1.x (scanner), 5.x (instrument), 9.x (wire)
- Files under `go/pkg/scanner/`, `go/pkg/codegen/`, `go/cmd/debugviz/`

## Package map

```
go/pkg/scanner/           — package/file graph, orchestration
go/pkg/scanner/adapters/  — EntryDiscoverer per framework
go/pkg/codegen/instrument/ — StartSpan injection in function bodies
go/pkg/codegen/wire/      — Configure + HTTPMiddleware/interceptors in main
go/pkg/protocol/          — Graph, EntryPoint, Span types (match schemas/)
```

## Scanner (debugviz scan)

1. Load packages with `golang.org/x/tools/go/packages` (mode: Types, Syntax, Imports).
2. Build nodes: `package`, `file`, `function`, `entry_point`, `middleware`.
3. Build edges: `imports`, `calls`, `entry_handles`, `middleware_chain`.
4. Run adapters based on `--framework` (`auto` = detect from imports).
5. Merge manual `entries:` from `debugviz.yaml` as fallback.

### EntryDiscoverer interface

```go
type EntryDiscoverer interface {
    Name() string
    Discover(pkgs []*packages.Package) ([]EntryPoint, error)
}
```

Adapters: chi, gin, echo, stdlib, grpc (1.5), cli (1.6), worker (1.7).

## Instrument (debugviz instrument)

- Insert at start of function body:
  ```go
  ctx, __dv_end := debugviz.StartSpan(ctx, "Package.Func")
  defer __dv_end()
  ```
- Only if first param is `context.Context` (unless `allow_no_context: true`).
- Run `go/format.Node` after rewrite.
- **Idempotent:** detect existing `__dv_end` / debugviz span prefix; skip if present.

## Wire (debugviz wire)

- Parse `wire.main` from yaml; rewrite that file (and router setup if needed).
- Inject `ConfigureFromEnv()` or `Configure(...)` at start of `main`.
- HTTP: wrap handler passed to `http.ListenAndServe` with `HTTPMiddleware`.
- Respect `//debugviz:wire skip` and `//debugviz:app` annotations.
- Prefer `--dry-run` in development; golden tests for before/after main.go.

## Testing

- **Golden files:** `testdata/*.go` input → `*.golden.go` expected output.
- **Scan golden:** `demo/http` graph JSON snapshot.
- Run: `go test ./go/pkg/...`
- CI: `go generate` + diff check where applicable.

## Quality bar

- No panics on malformed user code — return actionable errors.
- Preserve comments and imports order where go/format allows.
- Call graph edges: set `confidence`: static | interface | unknown.

## Do not

- Use reflection for route discovery (post-MVP only).
- Break `go build` without `-tags debugviz` (instrumented code behind build tags where needed).
