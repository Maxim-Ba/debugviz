package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//debugviz:app name=demo-cli
//go:generate go run ../../go/cmd/debugviz wire --config debugviz.yaml --write .

func main() {
	rootCmd := &cobra.Command{Use: "demo-cli"}

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

	if err := rootCmd.Execute(); err != nil {
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
