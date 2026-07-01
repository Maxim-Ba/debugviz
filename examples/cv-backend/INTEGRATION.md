# cv-backend — пример HTTP-интеграции DebugViz

[cv-backend](https://github.com/Maxim-Ba/cv-backend) — **референсный HTTP-проект** для документации DebugViz. Это не единственный supported target: тот же поток применим к любому Go-приложению с chi/gin/echo/stdlib.

**Слои cv-backend:** `router → services → repository` (chi v5, PostgreSQL).

---

## Что получите

| Уровень | Результат |
|---------|-----------|
| Статический граф | 3D-визуализация ~77 HTTP entry points, call edges между слоями |
| Live trace | Подсветка path `GET /api/tech/1` от handler через service до repository |
| Codegen (опционально) | Авто-spans в `internal/services/**` и `internal/repository/**` |

---

## Предварительные требования

- Go 1.23+
- DebugViz CLI: `go install github.com/Maxim-Ba/debugviz/go/cmd/debugviz@latest`
- Debug stack: `docker compose up` в репозитории [debugviz](https://github.com/Maxim-Ba/debugviz) (UI :3000, server :4000)
- Клон cv-backend рядом или в любом каталоге

---

## Шаг 1 — Сканирование и загрузка графа

Из корня cv-backend:

```bash
debugviz scan ./... \
  --output graph.json \
  --framework auto
```

**Ожидаемый результат:** `graph.json` v2 с `entry_point` узлами (`kind: http`), ≥ 20 REST routes, call path из ≥ 4 файлов для типичного handler (например `GET /api/tech/{techID}`: `techhndlr.go` → `technology.go` (service) → `technology.go` (repository)).

Загрузка в debug-server:

```bash
curl -X POST http://localhost:4000/api/graph \
  -H "Content-Type: application/json" \
  --data-binary @graph.json

curl http://localhost:4000/api/graph/meta
# {"entry_points": 77, ...}
```

### Pre-built graph

В этом репозитории уже лежит снимок графа: [`graph.json`](graph.json). Используйте его для UI без локального сканирования:

```bash
curl -X POST http://localhost:4000/api/graph \
  -H "Content-Type: application/json" \
  --data-binary @examples/cv-backend/graph.json
```

Пересоздать снимок (из корня debugviz, при установленном cv-backend):

```bash
make scan-cv-backend
```

---

## Шаг 2 — Runtime: entry hook (HTTP middleware)

Два пути — как в [README](../../README.md#два-пути-интеграции).

### Рекомендуемый (M6+): `debugviz wire`

```yaml
# debugviz.yaml (в корне cv-backend)
server_url: http://localhost:4000
service_name: cv-backend

wire:
  main: ./cmd/main.go
  http:
    listen_and_serve: true

include:
  - "internal/services/**"
  - "internal/repository/**"
```

```go
// cmd/main.go
//debugviz:app name=cv-backend
//go:generate debugviz wire --config debugviz.yaml --write
//go:generate debugviz instrument --config debugviz.yaml --write ./...
```

```bash
go generate ./...
go build -tags debugviz ./cmd/...
```

После `wire` в `main` появятся `ConfigureFromEnv` и обёртка `HTTPMiddleware` вокруг `ListenAndServe` — без ручного кода.

### Ручной (Advanced, M2)

**Зависимость:**

```bash
go get github.com/Maxim-Ba/debugviz/go/lib/debugviz
```

**`cmd/main.go`** — инициализация и обёртка handler:

```go
import (
    "github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

func main() {
    if err := debugviz.ConfigureFromEnv(); err != nil {
        log.Fatalf("debugviz: %v", err)
    }

    // ... initApplication → router ...

    handler := debugviz.HTTPMiddleware(router.R, debugviz.HTTPMiddlewareConfig{
        ServiceName: "cv-backend",
    })

    server := &http.Server{
        Addr:    cfg.ServerAddr,
        Handler: handler,
    }
    // ListenAndServe / Shutdown как раньше
}
```

**Альтернатива (chi middleware):** в `internal/router/router.go` после создания `chi.Mux`:

```go
r.Use(debugviz.ChiMiddleware(debugviz.HTTPMiddlewareConfig{
    ServiceName: "cv-backend",
}))
```

> Chi-middleware ставьте **после** CORS/logging, **до** route handlers — иначе root span не увидит финальный path.

**Сборка с runtime:**

```bash
DEBUGVIZ_ENABLED=true \
DEBUGVIZ_SERVER_URL=http://localhost:4000 \
DEBUGVIZ_SERVICE_NAME=cv-backend \
go run -tags debugviz ./cmd/...
```

---

## Шаг 3 — Codegen: inner spans (M3)

`debugviz.yaml` (см. выше) + `go generate` вставляет `StartSpan` в функции с `context.Context` в `services` и `repository`.

Проверка без записи:

```bash
debugviz instrument --config debugviz.yaml --dry-run ./...
```

Идемпотентная запись:

```bash
debugviz instrument --config debugviz.yaml --write ./...
```

Пример сгенерированного кода в `internal/services/technology.go`:

```go
func (s *TechService) GetWithTags(id int64) (dto.TechnologyWithTagsDTO, error) {
    ctx, __dv_end := debugviz.StartSpan(context.Background(), "services.TechService.GetWithTags")
    defer __dv_end()
    // ...
}
```

> Для handler-функций без `ctx` в сигнатуре (`TechGet(w, r)`) inner span не вставляется — root span даёт middleware, глубина — через instrument в service/repo.

---

## Шаг 4 — Запуск вместе с debug stack

**Вариант A:** cv-backend отдельно, debugviz через docker compose:

```bash
# Терминал 1 — debugviz
cd /path/to/debugviz
docker compose up

# Терминал 2 — cv-backend (порт по умолчанию :3333)
cd /path/to/cv-backend
DEBUGVIZ_ENABLED=true \
DEBUGVIZ_SERVER_URL=http://localhost:4000 \
DEBUGVIZ_SERVICE_NAME=cv-backend \
go run -tags debugviz ./cmd/...
```

**Вариант B:** только статика — загрузите pre-built `graph.json` (шаг 1) и откройте UI без запуска cv-backend.

---

## Шаг 5 — Проверка live trace

```bash
# Загрузить граф (если ещё не загружен)
curl -X POST http://localhost:4000/api/graph \
  -H "Content-Type: application/json" \
  --data-binary @graph.json

# Запрос с явным trace id
curl -H "X-Trace-ID: cv-demo-1" http://localhost:3333/api/tech/1
```

Откройте [http://localhost:3000](http://localhost:3000).

**Ожидаемый path в UI:**

```
entry:http:GET:/api/tech/{techID}
  → func:internal/router/techhndlr.go:TechGet
  → func:internal/services/technology.go:GetWithTags
  → func:internal/repository/technology.go:GetWithTags
```

Подсветка появляется < 1 сек после запроса (WebSocket → UI).

---

## Environment variables

| Переменная | Default | Описание |
|------------|---------|----------|
| `DEBUGVIZ_ENABLED` | `false` | Включить экспорт spans |
| `DEBUGVIZ_SERVER_URL` | `http://localhost:4000` | URL debug-server |
| `DEBUGVIZ_SERVICE_NAME` | — | Имя сервиса (`cv-backend`) |
| `DEBUGVIZ_SAMPLE_RATE` | `1.0` | Доля traces |
| `DEBUGVIZ_BATCH_SIZE` | `50` | Spans в batch POST |

---

## Troubleshooting

| Симптом | Решение |
|---------|---------|
| UI пустой, нет графа | `POST /api/graph` + проверить `GET /api/graph/meta` |
| Нет live подсветки | `DEBUGVIZ_ENABLED=true`, debug-server доступен, граф загружен |
| Root span есть, inner нет | Запустить `instrument --write`, сборка с `-tags debugviz` |
| `node_id` null в trace | Пути в span `file` должны совпадать с путями в `graph.json` (относительные от module root) |
| chi route не в графе | `debugviz scan --framework chi` или `auto`; проверить `r.Get/Post` в router |

---

## См. также

- [README — Пример HTTP-интеграции](../../README.md#пример-http-интеграция-cv-backend)
- [README — Сценарий 2 (live trace)](../../README.md#сценарий-2--live-trace-m2--m5)
- [README — Сценарий 3 (codegen)](../../README.md#сценарий-3--codegen-m3--m6)
- [demo/http](../../demo/http/) — минимальный рабочий пример в monorepo
