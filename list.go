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

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/lure/distro"
)

func listCmd(c *cli.Context) error {
	info, err := distro.ParseOSRelease(c.Context)
	if err != nil {
		log.Fatal("Error parsing os-release").Err(err).Send()
	}

	pkgs, err := findPkg("*")
	if err != nil {
		log.Fatal("Error finding packages").Err(err).Send()
	}

	for _, script := range pkgs {
		vars, err := getBuildVars(c.Context, script, info)
		if err != nil {
			log.Fatal("Error getting build variables").Err(err).Send()
		}

		fmt.Println(vars.Name, vars.Version)
	}

	return nil
}
