package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "test-cli"}

	serveCmd := &cobra.Command{
		Use: "serve",
		Run: runServe,
	}

	migrateCmd := &cobra.Command{Use: "migrate"}
	upCmd := &cobra.Command{
		Use: "up",
		Run: runMigrateUp,
	}
	migrateCmd.AddCommand(upCmd)

	rootCmd.AddCommand(serveCmd, migrateCmd)
	_ = rootCmd.Execute()
}

func runServe(_ *cobra.Command, _ []string) {
	fmt.Println("serve")
}

func runMigrateUp(_ *cobra.Command, _ []string) {
	fmt.Println("migrate up")
}
