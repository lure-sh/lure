/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Elara Musayelyan
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

	"github.com/urfave/cli/v2"
	"lure.sh/lure/internal/config"
	"lure.sh/lure/internal/db"
	"lure.sh/lure/internal/types"
	"lure.sh/lure/pkg/build"
	"lure.sh/lure/pkg/distro"
	"lure.sh/lure/pkg/loggerctx"
	"lure.sh/lure/pkg/manager"
	"lure.sh/lure/pkg/repos"
	"go.elara.ws/vercmp"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var upgradeCmd = &cli.Command{
	Name:    "upgrade",
	Usage:   "Upgrade all installed packages",
	Aliases: []string{"up"},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "clean",
			Aliases: []string{"c"},
			Usage:   "Build package from scratch even if there's an already built package available",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		log := loggerctx.From(ctx)

		info, err := distro.ParseOSRelease(ctx)
		if err != nil {
			log.Fatal("Error parsing os-release file").Err(err).Send()
		}

		mgr := manager.Detect()
		if mgr == nil {
			log.Fatal("Unable to detect a supported package manager on the system").Send()
		}

		err = repos.Pull(ctx, config.Config(ctx).Repos)
		if err != nil {
			log.Fatal("Error pulling repos").Err(err).Send()
		}

		updates, err := checkForUpdates(ctx, mgr, info)
		if err != nil {
			log.Fatal("Error checking for updates").Err(err).Send()
		}

		if len(updates) > 0 {
			build.InstallPkgs(ctx, updates, nil, types.BuildOpts{
				Manager:     mgr,
				Clean:       c.Bool("clean"),
				Interactive: c.Bool("interactive"),
			})
		} else {
			log.Info("There is nothing to do.").Send()
		}

		return nil
	},
}

func checkForUpdates(ctx context.Context, mgr manager.Manager, info *distro.OSRelease) ([]db.Package, error) {
	installed, err := mgr.ListInstalled(nil)
	if err != nil {
		return nil, err
	}

	pkgNames := maps.Keys(installed)
	found, _, err := repos.FindPkgs(ctx, pkgNames)
	if err != nil {
		return nil, err
	}

	var out []db.Package
	for pkgName, pkgs := range found {
		if slices.Contains(config.Config(ctx).IgnorePkgUpdates, pkgName) {
			continue
		}

		if len(pkgs) > 1 {
			// Puts the element with the highest version first
			slices.SortFunc(pkgs, func(a, b db.Package) int {
				return vercmp.Compare(a.Version, b.Version)
			})
		}

		// First element is the package we want to install
		pkg := pkgs[0]

		repoVer := pkg.Version
		if pkg.Release != 0 && pkg.Epoch == 0 {
			repoVer = fmt.Sprintf("%s-%d", pkg.Version, pkg.Release)
		} else if pkg.Release != 0 && pkg.Epoch != 0 {
			repoVer = fmt.Sprintf("%d:%s-%d", pkg.Epoch, pkg.Version, pkg.Release)
		}

		c := vercmp.Compare(repoVer, installed[pkgName])
		if c == 0 || c == -1 {
			continue
		} else if c == 1 {
			out = append(out, pkg)
		}
	}
	return out, nil
}
