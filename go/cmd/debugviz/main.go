package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"github.com/Maxim-Ba/debugviz/go/pkg/scanner"
)

var version = "dev"

var (
	scanOutput       string
	scanFormat       string
	scanIncludeTests bool
	scanFramework    string
)

func main() {
	root := &cobra.Command{
		Use:   "debugviz",
		Short: "DebugViz — static graph and live trace for Go applications",
	}

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print debugviz version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("debugviz", version)
		},
	})

	scanCmd := &cobra.Command{
		Use:   "scan [patterns...]",
		Short: "Scan Go packages and emit a static graph (M1)",
		Long:  "Build a package/file/function/entry graph from Go source without running the application.",
		RunE:  runScan,
	}
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Write graph to file (default: stdout)")
	scanCmd.Flags().StringVar(&scanFormat, "format", "json", "Output format: json, dot")
	scanCmd.Flags().BoolVar(&scanIncludeTests, "include-tests", false, "Include *_test.go files")
	scanCmd.Flags().StringVar(&scanFramework, "framework", "auto", "Entry discoverer: auto, chi, gin, echo, stdlib, grpc, cli")
	root.AddCommand(scanCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runScan(cmd *cobra.Command, args []string) error {
	graph, err := scanner.Scan(args, scanner.Options{
		IncludeTests: scanIncludeTests,
		Framework:    scanFramework,
	})
	if err != nil {
		return err
	}

	data, err := encodeGraph(graph, scanFormat)
	if err != nil {
		return err
	}

	if scanOutput == "" {
		if _, err := cmd.OutOrStdout().Write(data); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
		return nil
	}

	if err := os.WriteFile(scanOutput, data, 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	return nil
}

func encodeGraph(graph *protocol.Graph, format string) ([]byte, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(graph, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode graph: %w", err)
		}
		return append(data, '\n'), nil
	case "dot":
		return []byte(scanner.FormatDOT(graph)), nil
	default:
		return nil, fmt.Errorf("unknown format %q: want json or dot", format)
	}
}
