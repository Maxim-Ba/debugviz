package adapters_test

import (
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters"
)

func TestCobraDiscoverSubcommands(t *testing.T) {
	pattern := "./go/pkg/scanner/adapters/testdata/clidemo/..."
	pkgs := loadPackages(t, pattern)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewCLI().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	commands := cliCommandSet(entries)
	for _, want := range []string{"serve", "migrate up"} {
		if _, ok := commands[want]; !ok {
			t.Fatalf("missing CLI command %q, got %v", want, commands)
		}
	}

	for _, entry := range entries {
		if entry.Kind != protocol.EntryKindCLI {
			t.Fatalf("entry kind = %q, want cli", entry.Kind)
		}
		if entry.Command == "" {
			t.Fatalf("entry missing command: %+v", entry)
		}
		if !entry.HasHandler {
			t.Fatalf("expected handler for command %q", entry.Command)
		}
	}
}

func TestUrfaveCLIDiscoverSubcommands(t *testing.T) {
	pattern := "./go/pkg/scanner/adapters/testdata/urfavecli/..."
	pkgs := loadPackages(t, pattern)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewCLI().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	commands := cliCommandSet(entries)
	for _, want := range []string{"deploy", "sync pull"} {
		if _, ok := commands[want]; !ok {
			t.Fatalf("missing CLI command %q, got %v", want, commands)
		}
	}
}

func cliCommandSet(entries []adapters.EntryPoint) map[string]struct{} {
	out := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		out[entry.Command] = struct{}{}
	}
	return out
}
