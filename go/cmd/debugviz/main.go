package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Maxim-Ba/debugviz/go/pkg/scanner"
)

var version = "dev"

var (
	scanOutput       string
	scanIncludeTests bool
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
		Long:  "Build a package/file dependency graph from Go source without running the application.",
		RunE:  runScan,
	}
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Write graph JSON to file (default: stdout)")
	scanCmd.Flags().BoolVar(&scanIncludeTests, "include-tests", false, "Include *_test.go files")
	root.AddCommand(scanCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runScan(_ *cobra.Command, args []string) error {
	graph, err := scanner.Scan(args, scanner.Options{IncludeTests: scanIncludeTests})
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("encode graph: %w", err)
	}
	data = append(data, '\n')

	if scanOutput == "" {
		if _, err := os.Stdout.Write(data); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
		return nil
	}

	if err := os.WriteFile(scanOutput, data, 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	return nil
}
