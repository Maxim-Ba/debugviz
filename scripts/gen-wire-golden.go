package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Maxim-Ba/debugviz/go/pkg/codegen/wire"
)

func main() {
	cases := []struct {
		in, out string
		cfg     wire.Config
		isMain  bool
	}{
		{"stdlib_http.input.go", "stdlib_http.golden.go", wire.Config{ServiceName: "my-app", Wire: wire.WireConfig{HTTP: wire.HTTPWireConfig{ListenAndServe: true}}}, true},
		{"chi_router.input.go", "chi_router.golden.go", wire.Config{ServiceName: "demo-http", Wire: wire.WireConfig{HTTP: wire.HTTPWireConfig{}}}, false},
		{"grpc_main.input.go", "grpc_main.golden.go", wire.Config{ServiceName: "demo-grpc", Wire: wire.WireConfig{GRPC: wire.GRPCWireConfig{NewServer: true}}}, true},
		{"cli_cobra.input.go", "cli_cobra.golden.go", wire.Config{ServiceName: "demo-cli", Wire: wire.WireConfig{CLI: wire.CLIWireConfig{CobraExecute: true}}}, true},
		{"worker_main.input.go", "worker_main.golden.go", wire.Config{ServiceName: "demo-worker", Wire: wire.WireConfig{Worker: wire.WorkerWireConfig{Targets: []wire.WorkerJobTarget{{Name: "OrderConsumer.Process"}}}}}, true},
		{"app_annotation.input.go", "app_annotation.golden.go", wire.Config{Wire: wire.WireConfig{HTTP: wire.HTTPWireConfig{ListenAndServe: true}}}, true},
		{"configure_only.input.go", "configure_only.golden.go", wire.Config{}, true},
		{"grpc_existing_opts.input.go", "grpc_existing_opts.golden.go", wire.Config{Wire: wire.WireConfig{GRPC: wire.GRPCWireConfig{NewServer: true}}}, true},
		{"http_skip.input.go", "http_skip.golden.go", wire.Config{Wire: wire.WireConfig{HTTP: wire.HTTPWireConfig{ListenAndServe: true}}}, true},
	}
	dir := filepath.Join("go", "pkg", "codegen", "wire", "testdata")
	for _, c := range cases {
		in, err := os.ReadFile(filepath.Join(dir, c.in))
		if err != nil {
			panic(err)
		}
		out, _, err := wire.WireFile(in, c.in, c.cfg, c.isMain)
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(dir, c.out), out, 0o644); err != nil {
			panic(err)
		}
		fmt.Println("wrote", c.out)
	}
	in, err := os.ReadFile(filepath.Join(dir, "stdlib_http.golden.go"))
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "idempotent.input.go"), in, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote idempotent.input.go")
}
