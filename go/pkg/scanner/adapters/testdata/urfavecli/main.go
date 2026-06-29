package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "deploy",
				Action: runDeploy,
			},
			{
				Name: "sync",
				Subcommands: []*cli.Command{
					{
						Name:   "pull",
						Action: runSyncPull,
					},
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runDeploy(_ *cli.Context) error {
	fmt.Println("deploy")
	return nil
}

func runSyncPull(_ *cli.Context) error {
	fmt.Println("sync pull")
	return nil
}
