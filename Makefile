.PHONY: dev test lint scan scan-cv-backend fmt wasm-build demo-ui demo-stack upload-graph open-ui wait-server demo-ping smoke-epic6

CV_BACKEND_DIR ?= ../Go_Training/cv-backend

GRAPH_FILE ?= graph.json
UI_URL ?= http://localhost:3000
SERVER_URL ?= http://localhost:4000
DEMO_URL ?= http://localhost:8080

# Browser opener (Linux/WSL: xdg-open, macOS: open, Windows: start)
ifeq ($(OS),Windows_NT)
OPEN = cmd.exe /c start
else
UNAME_S := $(shell uname -s 2>/dev/null)
ifeq ($(UNAME_S),Darwin)
OPEN = open
else
OPEN = xdg-open
endif
endif

dev:
	pnpm dev

test:
	go test ./...
	pnpm test
	cd rust/layout && wasm-pack test --node

lint:
	golangci-lint run ./...
	pnpm lint
	cargo fmt --check --manifest-path rust/layout/Cargo.toml

scan:
	go run ./go/cmd/debugviz scan ./demo/http -o $(GRAPH_FILE) --framework auto

# Regenerate pre-built cv-backend graph (issue 6.2). Requires cv-backend checkout.
scan-cv-backend:
	@test -d "$(CV_BACKEND_DIR)" || (echo "cv-backend not found at $(CV_BACKEND_DIR); set CV_BACKEND_DIR=..."; exit 1)
	go build -o bin/debugviz$(if $(filter Windows_NT,$(OS)),.exe,) ./go/cmd/debugviz
	cd "$(CV_BACKEND_DIR)" && "$(CURDIR)/bin/debugviz$(if $(filter Windows_NT,$(OS)),.exe,)" scan ./... -o "$(CURDIR)/examples/cv-backend/graph.json" --framework auto

smoke-epic6:
	pnpm --filter @debugviz/server smoke:epic6

fmt:
	gofmt -w .
	pnpm format
	cargo fmt --manifest-path rust/layout/Cargo.toml

wasm-build:
	cd rust/layout && wasm-pack build --target web

# Wait until debug-server responds (docker compose or pnpm dev).
wait-server:
	@echo "Waiting for debug-server at $(SERVER_URL)..."
	@i=0; while [ $$i -lt 60 ]; do \
		curl -sf "$(SERVER_URL)/health" >/dev/null && exit 0; \
		i=$$((i+1)); sleep 1; \
	done; \
	echo "timeout: debug-server not ready at $(SERVER_URL)"; exit 1

# Scan demo/http and POST graph.json to debug-server.
upload-graph: scan wait-server
	@echo "Uploading $(GRAPH_FILE) -> $(SERVER_URL)/api/graph"
	@curl -sf -X POST "$(SERVER_URL)/api/graph" \
		-H "Content-Type: application/json" \
		--data-binary @$(GRAPH_FILE)
	@echo ""
	@echo "Graph meta:"
	@curl -sf "$(SERVER_URL)/api/graph/meta"; echo ""

open-ui:
	@echo "Open $(UI_URL) in browser"
	@$(OPEN) "$(UI_URL)" 2>/dev/null || true

# Docker: server + web + demo/http
demo-stack:
	docker compose up -d --build
	@$(MAKE) wait-server

# Manual browser check: stack + static graph + open UI
demo-ui: demo-stack upload-graph open-ui
	@echo ""
	@echo "3D graph: $(UI_URL)"
	@echo "Live trace: make demo-ping"

# Send one HTTP request to demo/http (path highlight in UI when tracing is enabled).
demo-ping:
	@echo "GET $(DEMO_URL)/api/items/1"
	@curl -sf "$(DEMO_URL)/api/items/1" | head -c 200; echo ""
