package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/logger"
)

var log = logger.NewPretty(os.Stderr)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		// Exit the program after a maximum of 200ms
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}()

	app := &cli.App{
		Name:  "lure",
		Usage: "Linux User REpository",
		Commands: []*cli.Command{
			{
				Name:    "install",
				Usage:   "Install a new package",
				Aliases: []string{"in"},
				Action:  installCmd,
			},
			{
				Name:    "remove",
				Usage:   "Remove an installed package",
				Aliases: []string{"rm"},
				Action:  removeCmd,
			},
			{
				Name:    "upgrade",
				Usage:   "Upgrade all installed packages",
				Aliases: []string{"up"},
				Action:  upgradeCmd,
			},
			{
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "script",
						Aliases: []string{"s"},
						Value:   "lure.sh",
						Usage:   "Path to the build script",
					},
				},
				Name:   "build",
				Usage:  "Build a local package",
				Action: buildCmd,
			},
			{
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Required: true,
						Usage:    "Name of the new repo",
					},
					&cli.StringFlag{
						Name:     "url",
						Aliases:  []string{"u"},
						Required: true,
						Usage:    "URL of the new repo",
					},
				},
				Name:    "addrepo",
				Usage:   "Add a new repository",
				Aliases: []string{"ar"},
				Action:  addrepoCmd,
			},
			{
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Required: true,
						Usage:    "Name of the repo to be deleted",
					},
				},
				Name:    "removerepo",
				Usage:   "Remove an existing repository",
				Aliases: []string{"rr"},
				Action:  removerepoCmd,
			},
			{
				Name:    "refresh",
				Usage:   "Pull all repositories that have changed",
				Aliases: []string{"ref"},
				Action:  refreshCmd,
			},
		},
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Error("Error while running app").Err(err).Send()
	}
}
