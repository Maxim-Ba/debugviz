# DebugViz — Бэклог MVP (Universal Go)

> Оценка: **11–13 недель** part-time. Обновляйте колонку **Статус**: `[ ]` — не начато, `[~]` — в работе, `[x]` — готово.

## Milestones

| Milestone | Недели | Задачи | Статус |
|-----------|--------|--------|--------|
| M0 Foundation | 1 | 0.1–0.3 | [x] |
| M1 Static Graph | 2 | 1.1–1.4, 2.1, 3.1–3.3 | [ ] |
| M2 Live Trace | 2 | 2.2–2.3, 3.2B, 3.4, 4.1–4.4 | [ ] |
| M3 Codegen | 2 | 5.1–5.4 | [ ] |
| M5 Universal Entry Points | 2 | 1.5–1.7, 3.6, 4.5–4.8, 5.5, 8.1–8.3 | [ ] |
| M6 Auto Wire | 1.5 | 9.1–9.6 | [ ] |
| M4 Polish | 1 | 6.1–6.3, 7.1–7.4 | [ ] |

---

## Epic 0: Foundation (M0)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 0.1 | Инициализация monorepo и tooling | P0 | 2d | [x] |
| 0.2 | JSON-схемы (entry_point, trace v2) | P0 | 1d | [x] |
| 0.3 | Demo suite: demo/http (chi REST) | P0 | 2d | [x] |

## Epic 1: Static Scanner (M1 + M5)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 1.1 | Сканер графа пакетов/файлов | P0 | 3d | [x] |
| 1.2 | HTTP entry discovery (chi/gin/echo/stdlib adapters) | P0 | 4d | [x] |
| 1.3 | Call graph (intra/inter-package) | P1 | 4d | [x] |
| 1.4 | CLI `debugviz scan` (`--framework auto`) | P0 | 1d | [x] |
| 1.5 | gRPC entry discovery | P1 | 3d | [x] |
| 1.6 | CLI entry discovery (cobra/urfave) | P2 | 2d | [ ] |
| 1.7 | Worker entry discovery (best-effort) | P2 | 2d | [ ] |

## Epic 2: Debug Server (M1–M2)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 2.1 | HTTP API + загрузка статического графа | P0 | 2d | [ ] |
| 2.2 | Приём trace + WebSocket | P0 | 3d | [ ] |
| 2.3 | Маппинг span → node (+ entry_point) | P0 | 2d | [ ] |

## Epic 3: 3D Frontend (M1–M2 + M5)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 3.1 | Bootstrap Three.js сцены | P0 | 3d | [ ] |
| 3.2A | Graph layout в JS (MVP) | P1 | 2d | [ ] |
| 3.2B | Graph layout WASM (Rust) | P1 | 2d | [ ] |
| 3.3 | Отрисовка рёбер + labels (icons по entry kind) | P1 | 2d | [ ] |
| 3.4 | Live trace visualization (any entry kind) | P0 | 4d | [ ] |
| 3.5 | UI panels (trace history, filters) | P2 | 2d | [ ] |
| 3.6 | Entry point picker (filter by kind) | P1 | 2d | [ ] |

## Epic 4: Go Runtime Library (M2 + M5)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 4.1 | Core span API | P0 | 2d | [ ] |
| 4.2 | Universal HTTP middleware (`HTTPMiddleware`) | P0 | 2d | [ ] |
| 4.3 | HTTP exporter | P0 | 1d | [ ] |
| 4.4 | Manual span helpers | P1 | 1d | [ ] |
| 4.5 | Router thin wrappers (chi/gin/echo) | P2 | 1d | [ ] |
| 4.6 | gRPC interceptors (unary + stream) | P1 | 2d | [ ] |
| 4.7 | CLI root span hook (`RunCLI`) | P2 | 1d | [ ] |
| 4.8 | Worker job span hook (`RunJob`) | P2 | 1d | [ ] |

## Epic 5: Codegen Instrumentation (M3)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 5.1 | AST-анализатор функций | P0 | 3d | [ ] |
| 5.2 | Code injector | P0 | 4d | [ ] |
| 5.3 | CLI `debugviz instrument` | P0 | 2d | [ ] |
| 5.4 | Интеграция go generate | P1 | 1d | [ ] |
| 5.5 | Codegen worker handlers без ctx (opt-in) | P2 | 2d | [ ] |

## Epic 6: cv-backend Integration (M4)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 6.1 | Integration guide (HTTP example) | P0 | 2d | [ ] |
| 6.2 | Pre-built graph для cv-backend | P2 | 0.5d | [ ] |
| 6.3 | E2E smoke test | P1 | 2d | [ ] |

## Epic 7: Polish & Launch (M4)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 7.1 | Docker one-liner | P0 | 1d | [ ] |
| 7.2 | README spec v2 + auto wire UX | P0 | 2d | [x] |
| 7.3 | Benchmark suite | P2 | 1d | [ ] |
| 7.4 | Landing / portfolio page | P2 | 1d | [ ] |

## Epic 8: Multi-runtime demos (M5)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 8.1 | demo/grpc (unary service) | P1 | 1d | [ ] |
| 8.2 | demo/cli (cobra subcommands) | P2 | 1d | [ ] |
| 8.3 | demo/worker (fake queue consumer) | P2 | 1d | [ ] |

## Epic 9: Auto Wire (M6)

| ID | Задача | Приоритет | Оценка | Статус |
|----|--------|-----------|--------|--------|
| 9.1 | CLI `debugviz wire` | P1 | 2d | [ ] |
| 9.2 | Wire config + `ConfigureFromEnv` | P1 | 1d | [ ] |
| 9.3 | HTTP auto-wiring (ListenAndServe, middleware) | P1 | 3d | [ ] |
| 9.4 | gRPC / CLI / worker auto-wiring | P2 | 3d | [ ] |
| 9.5 | Аннотации `//debugviz:app` | P2 | 1d | [ ] |
| 9.6 | Golden tests + CI для wire | P1 | 2d | [ ] |

---

## Definition of Done (MVP)

| Критерий | Статус |
|----------|--------|
| `docker compose up` — demo/http end-to-end | [ ] |
| `debugviz scan --framework auto` → entry points в UI | [ ] |
| Live HTTP curl → path подсвечен в 3D | [ ] |
| Live gRPC grpcurl → path подсвечен в 3D | [ ] |
| CLI command → path visible (P2, post-launch OK) | [ ] |
| `debugviz instrument --write` + `-tags debugviz` — auto inner spans | [ ] |
| `debugviz wire --write` — demo/http без ручного wiring (M6) | [ ] |
| README spec v2 + INTEGRATION.md + CI green | [~] |
| Screen recording 60 sec для portfolio | [ ] |

---

## README ↔ Issues (traceability)

Целевая спецификация: [README.md](../README.md) (v2 Universal Go).

| Issue | README section |
|-------|----------------|
| 0.1 | [Разработка monorepo](../README.md#разработка-monorepo) |
| 0.2 | [Span format](../README.md#span-format), [EntryPoint model](../README.md#модель-entrypoint) |
| 0.3 | [Quick Start](../README.md#quick-start), demo/http |
| 1.1–1.3 | [Сценарий 1](../README.md#сценарий-1--статический-граф-m1) |
| 1.2 | [Сканирование](../README.md#шаг-1--сканирование), [Выбор integration](../README.md#выбор-integration) |
| 1.4 | [Флаги scan](../README.md#шаг-1--сканирование) |
| 1.5 | [gRPC discovery](../README.md#выбор-integration) |
| 1.6 | [CLI discovery](../README.md#выбор-integration) |
| 1.7 | [Worker discovery](../README.md#выбор-integration) |
| 2.1 | [HTTP API](../README.md#http-api) |
| 2.2–2.3 | [Сценарий 2 — UI result](../README.md#ожидаемый-результат-в-ui) |
| 3.1–3.3 | [Сценарий 1 — UI](../README.md#шаг-3--просмотр-в-ui) |
| 3.4 | [Сценарий 2](../README.md#сценарий-2--live-trace-m2--m5) |
| 3.5 | [UI panels P2](../README.md#ui-httplocalhost3000) |
| 3.6 | [Entry point picker](../README.md#ui-httplocalhost3000) |
| 4.1 | [Runtime API](../README.md#runtime-api) |
| 4.2 | [HTTP middleware](../README.md#http--universal-nethttp-middleware) |
| 4.3 | [Exporter behavior](../README.md#exporter-behavior) |
| 4.4 | [Manual spans](../README.md#manual-spans-все-типы-приложений) |
| 4.5 | [HTTP — chi wrapper](../README.md#http--universal-nethttp-middleware) |
| 4.6 | [gRPC interceptors](../README.md#grpc--interceptors) |
| 4.7 | [CLI RunCLI](../README.md#cli--root-span) |
| 4.8 | [Worker RunJob](../README.md#worker--job-span) |
| 5.1–5.4 | [Сценарий 3](../README.md#сценарий-3--codegen-m3) |
| 5.5 | [allow_no_context](../README.md#конфиг-debugvizyaml) |
| 6.1 | [Пример cv-backend](../README.md#пример-http-интеграция-cv-backend) |
| 6.2 | [Quick Start](../README.md#quick-start) |
| 6.3 | E2E (не в README) |
| 7.1 | [Quick Start](../README.md#quick-start) |
| 7.2 | README v2 (done) |
| 7.3 | [Benchmark](../README.md#benchmark) |
| 7.4 | post-MVP |
| 8.1 | [Quick Start gRPC](../README.md#quick-start) |
| 8.2 | [Quick Start CLI](../README.md#quick-start) |
| 8.3 | [Quick Start worker](../README.md#quick-start) |
| 9.1–9.6 | [Два пути интеграции](../README.md#два-пути-интеграции), [Сценарий 3](../README.md#сценарий-3--codegen-m3--m6) |

---

Подробное описание задач: [MVP_PLAN.md](./MVP_PLAN.md)
