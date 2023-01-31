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
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/repos"
	"go.arsenm.dev/lure/manager"
)

func listCmd(c *cli.Context) error {
	err := repos.Pull(c.Context, gdb, cfg.Repos)
	if err != nil {
		log.Fatal("Error pulling repositories").Err(err).Send()
	}

	where := "true"
	args := []any(nil)
	if c.NArg() > 0 {
		where = "name LIKE ? OR json_array_contains(provides, ?)"
		args = []any{c.Args().First(), c.Args().First()}
	}

	result, err := db.GetPkgs(gdb, where, args...)
	if err != nil {
		log.Fatal("Error getting packages").Err(err).Send()
	}
	defer result.Close()

	var installed map[string]string
	if c.Bool("installed") {
		mgr := manager.Detect()
		if mgr == nil {
			log.Fatal("Unable to detect supported package manager on system").Send()
		}

		installed, err = mgr.ListInstalled(&manager.Opts{AsRoot: false})
		if err != nil {
			log.Fatal("Error listing installed packages").Err(err).Send()
		}
	}

	for result.Next() {
		var pkg db.Package
		err := result.StructScan(&pkg)
		if err != nil {
			return err
		}

		version := pkg.Version
		if c.Bool("installed") {
			instVersion, ok := installed[pkg.Name]
			if !ok {
				return nil
			} else {
				version = instVersion
			}
		}

		fmt.Printf("%s/%s %s\n", pkg.Repository, pkg.Name, version)
	}

	if err != nil {
		log.Fatal("Error iterating over packages").Err(err).Send()
	}

	return nil
}
