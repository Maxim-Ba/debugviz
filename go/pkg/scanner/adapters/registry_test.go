package adapters_test

import (
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters"
)

func TestSelectDiscoverersAutoUsesCLIForDemoCLI(t *testing.T) {
	pkgs := loadPackages(t, "./demo/cli/...")
	discoverers, err := adapters.SelectDiscoverers(adapters.FrameworkAuto, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	names := discovererNames(discoverers)
	if !contains(names, "cli") {
		t.Fatalf("auto mode: want cli discoverer, got %v", names)
	}
}

func TestSelectDiscoverersAutoUsesGRPCForDemoGRPC(t *testing.T) {
	pkgs := loadPackages(t, "./demo/grpc/...")
	discoverers, err := adapters.SelectDiscoverers(adapters.FrameworkAuto, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	names := discovererNames(discoverers)
	if !contains(names, "grpc") {
		t.Fatalf("auto mode: want grpc discoverer, got %v", names)
	}
}

func TestSelectDiscoverersAutoUsesChiForDemoHTTP(t *testing.T) {
	pkgs := loadPackages(t, "./demo/http/...")
	discoverers, err := adapters.SelectDiscoverers(adapters.FrameworkAuto, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	names := discovererNames(discoverers)
	if !contains(names, "chi") {
		t.Fatalf("auto mode: want chi discoverer, got %v", names)
	}
	if !contains(names, "stdlib") {
		t.Fatalf("auto mode: want stdlib discoverer, got %v", names)
	}
}

func TestNormalizeFrameworkRejectsUnknown(t *testing.T) {
	if _, err := adapters.NormalizeFramework("invalid"); err == nil {
		t.Fatal("expected error for unknown framework")
	}
}

func discovererNames(discoverers []adapters.EntryDiscoverer) []string {
	names := make([]string, 0, len(discoverers))
	for _, d := range discoverers {
		names = append(names, d.Name())
	}
	return names
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
