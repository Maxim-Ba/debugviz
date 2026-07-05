package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Maxim-Ba/debugviz/go/pkg/codegen/instrument"
	"github.com/Maxim-Ba/debugviz/go/pkg/codegen/wire"
	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"github.com/Maxim-Ba/debugviz/go/pkg/scanner"
)

var version = "dev"

var (
	scanOutput       string
	scanFormat       string
	scanIncludeTests bool
	scanFramework    string

	instrumentConfig       string
	instrumentDryRun       bool
	instrumentWrite        bool
	instrumentIncludeTests bool

	wireConfig       string
	wireDryRun       bool
	wireWrite        bool
	wireIncludeTests bool
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
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

	instrumentCmd := &cobra.Command{
		Use:   "instrument [patterns...]",
		Short: "Inject inner debugviz spans into Go source (M3)",
		Long:  "Insert debugviz.StartSpan calls into function bodies according to debugviz.yaml rules.",
		RunE:  runInstrument,
	}
	instrumentCmd.Flags().StringVar(&instrumentConfig, "config", "", "Path to debugviz.yaml (default: ./debugviz.yaml)")
	instrumentCmd.Flags().BoolVar(&instrumentDryRun, "dry-run", false, "Print files that would be changed without writing")
	instrumentCmd.Flags().BoolVar(&instrumentWrite, "write", false, "Write instrumented source files")
	instrumentCmd.Flags().BoolVar(&instrumentIncludeTests, "include-tests", false, "Include *_test.go files")
	root.AddCommand(instrumentCmd)

	wireCmd := &cobra.Command{
		Use:   "wire [patterns...]",
		Short: "Inject entry hooks and Configure into main (M6)",
		Long:  "Insert debugviz ConfigureFromEnv and framework entry hooks according to debugviz.yaml wire rules.",
		RunE:  runWire,
	}
	wireCmd.Flags().StringVar(&wireConfig, "config", "", "Path to debugviz.yaml (default: ./debugviz.yaml)")
	wireCmd.Flags().BoolVar(&wireDryRun, "dry-run", false, "Print diff of files that would be changed without writing")
	wireCmd.Flags().BoolVar(&wireWrite, "write", false, "Write wired source files")
	wireCmd.Flags().BoolVar(&wireIncludeTests, "include-tests", false, "Include *_test.go files")
	root.AddCommand(wireCmd)

	return root
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

func runInstrument(cmd *cobra.Command, args []string) error {
	if !instrumentDryRun && !instrumentWrite {
		return fmt.Errorf("specify --dry-run or --write")
	}

	results, err := instrument.Run(instrument.Options{
		Patterns:     args,
		ConfigPath:   instrumentConfig,
		IncludeTests: instrumentIncludeTests,
		DryRun:       instrumentDryRun,
		Write:        instrumentWrite,
	})
	if err != nil {
		return err
	}

	if instrumentDryRun {
		if len(results) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no files to instrument")
			return nil
		}
		for _, result := range results {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "instrument %s (%d functions)\n", result.Path, result.Funcs)
		}
		return nil
	}

	var funcs int
	for _, result := range results {
		funcs += result.Funcs
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "instrumented %d files (%d functions)\n", len(results), funcs)
	return nil
}

func runWire(cmd *cobra.Command, args []string) error {
	if !wireDryRun && !wireWrite {
		return fmt.Errorf("specify --dry-run or --write")
	}

	results, err := wire.Run(wire.Options{
		Patterns:     args,
		ConfigPath:   wireConfig,
		IncludeTests: wireIncludeTests,
		DryRun:       wireDryRun,
		Write:        wireWrite,
	})
	if err != nil {
		return err
	}

	if wireDryRun {
		if len(results) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no files to wire")
			return nil
		}
		for i, result := range results {
			if err := wire.WriteDiff(cmd.OutOrStdout(), result); err != nil {
				return err
			}
			if i < len(results)-1 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}
		}
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wired %d files\n", len(results))
	return nil
}
