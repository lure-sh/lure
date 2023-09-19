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
	"fmt"

	"github.com/urfave/cli/v2"
	"go.elara.ws/logger/log"
	"go.elara.ws/lure/internal/build"
	"go.elara.ws/lure/internal/cliutils"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/db"
	"go.elara.ws/lure/internal/repos"
	"go.elara.ws/lure/internal/types"
	"go.elara.ws/lure/manager"
)

var installCmd = &cli.Command{
	Name:    "install",
	Usage:   "Install a new package",
	Aliases: []string{"in"},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "clean",
			Aliases: []string{"c"},
			Usage:   "Build package from scratch even if there's an already built package available",
		},
	},
	Action: func(c *cli.Context) error {
		args := c.Args()
		if args.Len() < 1 {
			log.Fatalf("Command install expected at least 1 argument, got %d", args.Len()).Send()
		}

		mgr := manager.Detect()
		if mgr == nil {
			log.Fatal("Unable to detect a supported package manager on the system").Send()
		}

		err := repos.Pull(c.Context, config.Config().Repos)
		if err != nil {
			log.Fatal("Error pulling repositories").Err(err).Send()
		}

		found, notFound, err := repos.FindPkgs(args.Slice())
		if err != nil {
			log.Fatal("Error finding packages").Err(err).Send()
		}

		pkgs := cliutils.FlattenPkgs(found, "install", c.Bool("interactive"))
		build.InstallPkgs(c.Context, pkgs, notFound, types.BuildOpts{
			Manager:     mgr,
			Clean:       c.Bool("clean"),
			Interactive: c.Bool("interactive"),
		})
		return nil
	},
	BashComplete: func(c *cli.Context) {
		result, err := db.GetPkgs("true")
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
	},
}

var removeCmd = &cli.Command{
	Name:    "remove",
	Usage:   "Remove an installed package",
	Aliases: []string{"rm"},
	Action: func(c *cli.Context) error {
		args := c.Args()
		if args.Len() < 1 {
			log.Fatalf("Command remove expected at least 1 argument, got %d", args.Len()).Send()
		}

		mgr := manager.Detect()
		if mgr == nil {
			log.Fatal("Unable to detect a supported package manager on the system").Send()
		}

		err := mgr.Remove(nil, c.Args().Slice()...)
		if err != nil {
			log.Fatal("Error removing packages").Err(err).Send()
		}

		return nil
	},
}
