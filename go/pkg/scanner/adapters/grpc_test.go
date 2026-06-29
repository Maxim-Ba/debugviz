package adapters_test

import (
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters"
)

func TestGRPCDiscoverUnaryMethods(t *testing.T) {
	pattern := "./go/pkg/scanner/adapters/testdata/grpcdemo/..."
	pkgs := loadPackages(t, pattern)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewGRPC().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	methods := grpcMethodSet(entries)
	for _, want := range []string{
		"demo.v1.UserService/GetUser",
		"demo.v1.UserService/CreateUser",
	} {
		if _, ok := methods[want]; !ok {
			t.Fatalf("missing gRPC method %q, got %v", want, methods)
		}
	}

	for _, entry := range entries {
		if entry.Kind != protocol.EntryKindGRPC {
			t.Fatalf("entry kind = %q, want grpc", entry.Kind)
		}
		if entry.Service == "" || entry.Method == "" {
			t.Fatalf("entry missing service/method metadata: %+v", entry)
		}
		if !entry.HasHandler {
			t.Fatalf("expected handler for %s/%s", entry.Service, entry.Method)
		}
	}
}

func grpcMethodSet(entries []adapters.EntryPoint) map[string]struct{} {
	out := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		out[entry.Service+"/"+entry.Method] = struct{}{}
	}
	return out
}
