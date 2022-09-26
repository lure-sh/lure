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
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/internal/shutils/decoder"
	"go.arsenm.dev/lure/manager"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func upgradeCmd(c *cli.Context) error {
	info, err := distro.ParseOSRelease(c.Context)
	if err != nil {
		log.Fatal("Error parsing os-release file").Err(err).Send()
	}

	mgr := manager.Detect()
	if mgr == nil {
		log.Fatal("Unable to detect supported package manager on system").Send()
	}

	updates, err := checkForUpdates(c.Context, mgr, info)
	if err != nil {
		log.Fatal("Error checking for updates").Err(err).Send()
	}

	if len(updates) > 0 {
		installPkgs(c.Context, updates, mgr)
	} else {
		log.Info("There is nothing to do.").Send()
	}

	return nil
}

func checkForUpdates(ctx context.Context, mgr manager.Manager, info *distro.OSRelease) ([]string, error) {
	installed, err := mgr.ListInstalled()
	if err != nil {
		return nil, err
	}

	var out []string
	for name, version := range installed {
		scripts, err := findPkg(name)
		if err != nil {
			continue
		}

		// since we're not using a glob, we can assume a single item
		script := scripts[0]

		fl, err := os.Open(script)
		if err != nil {
			return nil, err
		}

		file, err := syntax.NewParser().Parse(fl, "lure.sh")
		if err != nil {
			return nil, err
		}

		runner, err := interp.New()
		if err != nil {
			return nil, err
		}

		err = runner.Run(ctx, file)
		if err != nil {
			return nil, err
		}

		dec := decoder.New(info, runner)

		var vars BuildVars
		err = dec.DecodeVars(&vars)
		if err != nil {
			return nil, err
		}

		repoVer := vars.Version
		if vars.Release != 0 && vars.Epoch == 0 {
			repoVer = fmt.Sprintf("%s-%d", vars.Version, vars.Release)
		} else if vars.Release != 0 && vars.Epoch != 0 {
			repoVer = fmt.Sprintf("%d:%s-%d", vars.Epoch, vars.Version, vars.Release)
		}

		c := vercmp(repoVer, version)
		if c == 0 || c == -1 {
			continue
		} else if c == 1 {
			out = append(out, name)
		}
	}

	return out, nil
}
