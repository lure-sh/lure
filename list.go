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
	"fmt"

	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/types"
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

	result, err := db.GetPkgs(gdb, "true")
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

	err = result.Iterate(func(d types.Document) error {
		var pkg db.Package
		err := document.StructScan(d, &pkg)
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
		return nil
	})
	if err != nil {
		log.Fatal("Error iterating over packages").Err(err).Send()
	}

	return nil
}
