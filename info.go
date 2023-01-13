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
	"os"

	"go.arsenm.dev/logger/log"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/internal/cliutils"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/overrides"
	"go.arsenm.dev/lure/internal/repos"
	"gopkg.in/yaml.v3"
)

func infoCmd(c *cli.Context) error {
	args := c.Args()
	if args.Len() < 1 {
		log.Fatalf("Command info expected at least 1 argument, got %d", args.Len()).Send()
	}

	err := repos.Pull(c.Context, gdb, cfg.Repos)
	if err != nil {
		log.Fatal("Error pulling repositories").Err(err).Send()
	}

	found, _, err := repos.FindPkgs(gdb, args.Slice())
	if err != nil {
		log.Fatal("Error finding packages").Err(err).Send()
	}

	if len(found) == 0 {
		os.Exit(1)
	}

	pkgs := cliutils.FlattenPkgs(found, "show", translator)

	var names []string
	all := c.Bool("all")

	if !all {
		info, err := distro.ParseOSRelease(c.Context)
		if err != nil {
			log.Fatal("Error parsing os-release file").Err(err).Send()
		}
		names, err = overrides.Resolve(
			info,
			overrides.DefaultOpts.
				WithLanguages([]string{config.SystemLang()}),
		)
		if err != nil {
			log.Fatal("Error resolving overrides").Err(err).Send()
		}
	}

	for _, pkg := range pkgs {
		if !all {
			err = yaml.NewEncoder(os.Stdout).Encode(overrides.ResolvePackage(&pkg, names))
			if err != nil {
				log.Fatal("Error encoding script variables").Err(err).Send()
			}
		} else {
			err = yaml.NewEncoder(os.Stdout).Encode(pkg)
			if err != nil {
				log.Fatal("Error encoding script variables").Err(err).Send()
			}
		}

		fmt.Println("---")
	}

	return nil
}
