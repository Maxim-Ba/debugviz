package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}

	rootCmd := &cobra.Command{Use: "demo-cli"}
	rootCmd.PersistentPreRun = debugviz.CLICommandPreRun

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server",
		Run:   runServe,
	}

	migrateCmd := &cobra.Command{Use: "migrate"}
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Apply migrations",
		Run:   runMigrateUp,
	}
	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Rollback migrations",
		Run:   runMigrateDown,
	}
	migrateCmd.AddCommand(upCmd, downCmd)

	rootCmd.AddCommand(serveCmd, migrateCmd)

	if err := debugviz.RunCLI("demo-cli", func(ctx context.Context) error {
		return rootCmd.ExecuteContext(ctx)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServe(_ *cobra.Command, _ []string) {
	fmt.Println("serve")
}

func runMigrateUp(_ *cobra.Command, _ []string) {
	fmt.Println("migrate up")
}

func runMigrateDown(_ *cobra.Command, _ []string) {
	fmt.Println("migrate down")
}
