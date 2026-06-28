package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

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

	root.AddCommand(&cobra.Command{
		Use:   "scan",
		Short: "Scan Go packages and emit a static graph (M1)",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(os.Stderr, "scan: not implemented yet (issue 1.4)")
			os.Exit(1)
		},
	})

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
