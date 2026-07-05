package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
	"github.com/spf13/cobra"
)

func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}
	rootCmd := &cobra.Command{Use: "demo-cli"}
	serveCmd := &cobra.Command{
		Use: "serve",
		Run: func(_ *cobra.Command, _ []string) { fmt.Println("serve") },
	}
	rootCmd.AddCommand(serveCmd)
	if err := debugviz.RunCLI("demo-cli", func(ctx context.Context) error {
		return rootCmd.ExecuteContext(ctx)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
