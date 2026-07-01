package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	if err := os.Chdir(root); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestVersionIsSet(t *testing.T) {
	if version == "" {
		t.Fatal("version must not be empty")
	}
}

func TestScanCommandJSON(t *testing.T) {
	resetCommandFlags()
	outFile := filepath.Join(t.TempDir(), "graph.json")
	cmd := newRootCommand()
	cmd.SetArgs([]string{
		"scan", "./demo/http",
		"--output", outFile,
		"--framework", "auto",
		"--format", "json",
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	var graph protocol.Graph
	if err := json.Unmarshal(data, &graph); err != nil {
		t.Fatal(err)
	}
	if graph.Version != protocol.GraphVersion {
		t.Fatalf("version = %q, want %q", graph.Version, protocol.GraphVersion)
	}
	if graph.RootModule != "github.com/Maxim-Ba/debugviz" {
		t.Fatalf("root_module = %q", graph.RootModule)
	}

	var entryPoints int
	for _, node := range graph.Nodes {
		if node.Type == protocol.NodeTypeEntryPoint {
			entryPoints++
		}
	}
	if entryPoints < 6 {
		t.Fatalf("entry_points = %d, want at least 6", entryPoints)
	}
}

func TestScanCommandDOT(t *testing.T) {
	resetCommandFlags()
	cmd := newRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"scan", "./demo/http", "--format", "dot", "--framework", "auto"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"digraph debugviz {", "entry:http:GET:/health"} {
		if !strings.Contains(got, want) {
			t.Fatalf("dot output missing %q\n%s", want, got)
		}
	}
}

func TestScanCommandUnknownFormat(t *testing.T) {
	resetCommandFlags()
	cmd := newRootCommand()
	cmd.SetArgs([]string{"scan", "./demo/http", "--format", "yaml"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestScanCommandUnknownFramework(t *testing.T) {
	resetCommandFlags()
	cmd := newRootCommand()
	cmd.SetArgs([]string{"scan", "./demo/http", "--framework", "invalid"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown framework")
	}
}

func TestInstrumentCommandDryRun(t *testing.T) {
	resetCommandFlags()
	cmd := newRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"instrument",
		"--config", "go/cmd/debugviz/testdata/instrument/debugviz.yaml",
		"--dry-run",
		"./go/cmd/debugviz/testdata/instrument/...",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "instrument go/cmd/debugviz/testdata/instrument/sample.go") {
		t.Fatalf("dry-run output missing instrumented files:\n%s", out.String())
	}
}

func TestInstrumentCommandRequiresMode(t *testing.T) {
	resetCommandFlags()
	cmd := newRootCommand()
	cmd.SetArgs([]string{"instrument", "./demo/http"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error without --dry-run or --write")
	}
}

func resetCommandFlags() {
	scanOutput = ""
	scanFormat = "json"
	scanIncludeTests = false
	scanFramework = "auto"
	instrumentConfig = ""
	instrumentDryRun = false
	instrumentWrite = false
	instrumentIncludeTests = false
}
