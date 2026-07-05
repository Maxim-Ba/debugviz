package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "demo-cli"}
	serveCmd := &cobra.Command{
		Use: "serve",
		Run: func(_ *cobra.Command, _ []string) { fmt.Println("serve") },
	}
	rootCmd.AddCommand(serveCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
