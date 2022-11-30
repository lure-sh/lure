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
	"go.arsenm.dev/logger/log"
	"os"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/lure/distro"
	"gopkg.in/yaml.v3"
)

func infoCmd(c *cli.Context) error {
	args := c.Args()
	if args.Len() < 1 {
		log.Fatalf("Command info expected at least 1 argument, got %d", args.Len()).Send()
	}

	info, err := distro.ParseOSRelease(c.Context)
	if err != nil {
		log.Fatal("Error parsing os-release").Err(err).Send()
	}

	found, err := findPkg(args.First())
	if err != nil {
		log.Fatal("Error finding package").Err(err).Send()
	}

	// if multiple are matched, only use the first one
	script := found[0]

	vars, err := getBuildVars(c.Context, script, info)
	if err != nil {
		log.Fatal("Error getting build variables").Err(err).Send()
	}

	err = yaml.NewEncoder(os.Stdout).Encode(vars)
	if err != nil {
		log.Fatal("Error encoding script variables").Err(err).Send()
	}

	return nil
}
