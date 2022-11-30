/*
 * LURE - Linux User REpository
 * Copyright (C) 2022 Arsen Musayelyan
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

	"go.arsenm.dev/logger/log"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/lure/internal/repos"
	"go.arsenm.dev/lure/manager"
)

func installCmd(c *cli.Context) error {
	args := c.Args()
	if args.Len() < 1 {
		log.Fatalf("Command install expected at least 1 argument, got %d", args.Len()).Send()
	}

	mgr := manager.Detect()
	if mgr == nil {
		log.Fatal("Unable to detect supported package manager on system").Send()
	}

	installPkgs(c.Context, args.Slice(), mgr, true)

	return nil
}

func installPkgs(ctx context.Context, pkgs []string, mgr manager.Manager, pull bool) {
	if pull {
		err := repos.Pull(ctx, gdb, cfg.Repos)
		if err != nil {
			log.Fatal("Error pulling repositories").Err(err).Send()
		}
	}

	scripts, notFound := findPkgs(pkgs)

	if len(notFound) > 0 {
		err := mgr.Install(nil, notFound...)
		if err != nil {
			log.Fatal("Error installing native packages").Err(err).Send()
		}
	}

	installScripts(ctx, mgr, scripts)
}

func installScripts(ctx context.Context, mgr manager.Manager, scripts []string) {
	for _, script := range scripts {
		builtPkgs, _, err := buildPackage(ctx, script, mgr)
		if err != nil {
			log.Fatal("Error building package").Err(err).Send()
		}

		err = mgr.InstallLocal(nil, builtPkgs...)
		if err != nil {
			log.Fatal("Error installing package").Err(err).Send()
		}
	}
}

func removeCmd(c *cli.Context) error {
	args := c.Args()
	if args.Len() < 1 {
		log.Fatalf("Command remove expected at least 1 argument, got %d", args.Len()).Send()
	}

	mgr := manager.Detect()
	if mgr == nil {
		log.Fatal("Unable to detect supported package manager on system").Send()
	}

	err := mgr.Remove(nil, c.Args().Slice()...)
	if err != nil {
		log.Fatal("Error removing packages").Err(err).Send()
	}

	return nil
}
