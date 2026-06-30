package adapters_test

import (
	"path/filepath"
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters"
)

func TestWorkerDiscoverHandlerMethods(t *testing.T) {
	pattern := "./go/pkg/scanner/adapters/testdata/workerdemo/..."
	pkgs := loadPackages(t, pattern)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewWorker().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	jobs := workerJobSet(entries)
	for _, want := range []string{"OrderConsumer.Process", "OrderConsumer.Handle"} {
		if _, ok := jobs[want]; !ok {
			t.Fatalf("missing worker job %q, got %v", want, jobs)
		}
	}

	for _, entry := range entries {
		if entry.Kind != protocol.EntryKindWorker {
			t.Fatalf("entry kind = %q, want worker", entry.Kind)
		}
		if entry.Job == "" {
			t.Fatalf("entry missing job: %+v", entry)
		}
		if !entry.HasHandler {
			t.Fatalf("expected handler for job %q", entry.Job)
		}
	}
}

func TestWorkerDiscoverCronJobs(t *testing.T) {
	pattern := "./go/pkg/scanner/adapters/testdata/workercron/..."
	pkgs := loadPackages(t, pattern)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewWorker().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	jobs := workerJobSet(entries)
	if _, ok := jobs["cron:@every 1h"]; !ok {
		t.Fatalf("missing cron job, got %v", jobs)
	}

	for _, entry := range entries {
		if entry.Queue != "cron" {
			t.Fatalf("cron entry queue = %q, want cron", entry.Queue)
		}
	}
}

func TestManualWorkerEntriesFromYAML(t *testing.T) {
	root := repoRoot(t)
	configDir := filepath.Join(root, "go", "pkg", "scanner", "adapters", "testdata", "manual_worker")
	pkgs := loadPackages(t, "./demo/worker/...")
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.LoadManualEntries(configDir, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Kind != protocol.EntryKindWorker {
		t.Fatalf("kind = %q, want worker", entry.Kind)
	}
	if entry.Job != "ProcessOrder" || entry.Queue != "orders" {
		t.Fatalf("unexpected manual worker entry: %+v", entry)
	}
	if !entry.HasHandler {
		t.Fatal("expected handler for manual worker entry")
	}
}

func workerJobSet(entries []adapters.EntryPoint) map[string]struct{} {
	out := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		out[entry.Job] = struct{}{}
	}
	return out
}
