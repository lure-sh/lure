/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Arsen Musayelyan
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
	"go.elara.ws/logger"
	"go.elara.ws/logger/log"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/db"
	"go.elara.ws/lure/manager"
	"go.elara.ws/translate"
)

//go:generate scripts/gen-version.sh

//go:embed translations
var translationFS embed.FS

var translator translate.Translator

func init() {
	logger := logger.NewCLI(os.Stderr)

	t, err := translate.NewFromFS(translationFS)
	if err != nil {
		logger.Fatal("Error creating new translator").Err(err).Send()
	}
	translator = t

	log.Logger = translate.NewLogger(logger, t, config.Language)
}

func main() {
	if os.Geteuid() == 0 {
		log.Fatal("Running LURE as root is forbidden as it may cause catastrophic damage to your system").Send()
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		// Exit the program after a maximum of 200ms
		time.Sleep(200 * time.Millisecond)
		gdb.Close()
		os.Exit(0)
	}()

	app := &cli.App{
		Name:  "lure",
		Usage: "Linux User REpository",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "pm-args",
				Aliases: []string{"P"},
				Usage:   "Arguments to be passed on to the package manager",
			},
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Value:   isatty.IsTerminal(os.Stdin.Fd()),
				Usage:   "Enable interactive questions and prompts",
			},
		},
		Commands: []*cli.Command{
			{
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "clean",
						Aliases: []string{"c"},
						Usage:   "Build package from scratch even if there's an already built package available",
					},
				},
				Name:         "install",
				Usage:        "Install a new package",
				Aliases:      []string{"in"},
				Action:       installCmd,
				BashComplete: completionInstall,
			},
			{
				Name:    "remove",
				Usage:   "Remove an installed package",
				Aliases: []string{"rm"},
				Action:  removeCmd,
			},
			{
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "clean",
						Aliases: []string{"c"},
						Usage:   "Build package from scratch even if there's an already built package available",
					},
				},
				Name:    "upgrade",
				Usage:   "Upgrade all installed packages",
				Aliases: []string{"up"},
				Action:  upgradeCmd,
			},
			{
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "all",
						Aliases: []string{"a"},
						Usage:   "Show all information, not just for the current distro",
					},
				},
				Name:   "info",
				Usage:  "Print information about a package",
				Action: infoCmd,
			},
			{
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "installed",
						Aliases: []string{"I"},
					},
				},
				Name:    "list",
				Usage:   "List LURE repo packages",
				Aliases: []string{"ls"},
				Action:  listCmd,
			},
			{
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "script",
						Aliases: []string{"s"},
						Value:   "lure.sh",
						Usage:   "Path to the build script",
					},
					&cli.BoolFlag{
						Name:    "clean",
						Aliases: []string{"c"},
						Usage:   "Build package from scratch even if there's an already built package available",
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
				Name:    "listrepo",
				Usage:   "List all of the repos you have",
				Aliases: []string{"lr"},
				Action:  listrepoCmd,
			},
			{
				Name:    "refresh",
				Usage:   "Pull all repositories that have changed",
				Aliases: []string{"ref"},
				Action:  refreshCmd,
			},
			{
				Name:   "fix",
				Usage:  "Attempt to fix problems with LURE",
				Action: fixCmd,
			},
			{
				Name:   "version",
				Usage:  "Display the current LURE version and exit",
				Action: displayVersion,
			},
		},
		Before: func(c *cli.Context) error {
			args := strings.Split(c.String("pm-args"), " ")
			if len(args) == 1 && args[0] == "" {
				args = nil
			}

			manager.Args = append(manager.Args, args...)
			return nil
		},
		After: func(ctx *cli.Context) error {
			return gdb.Close()
		},
		EnableBashCompletion: true,
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Error("Error while running app").Err(err).Send()
	}
}

func displayVersion(c *cli.Context) error {
	print(config.Version)
	return nil
}

func completionInstall(c *cli.Context) {
	result, err := db.GetPkgs(gdb, "true")
	if err != nil {
		log.Fatal("Error getting packages").Err(err).Send()
	}
	defer result.Close()

	for result.Next() {
		var pkg db.Package
		err = result.StructScan(&pkg)
		if err != nil {
			log.Fatal("Error iterating over packages").Err(err).Send()
		}

		fmt.Println(pkg.Name)
	}
}
