.PHONY: dev test lint scan fmt wasm-build

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
	go run ./go/cmd/debugviz scan ./demo/http -o /tmp/graph.json

fmt:
	gofmt -w .
	pnpm format
	cargo fmt --manifest-path rust/layout/Cargo.toml

wasm-build:
	cd rust/layout && wasm-pack build --target web
