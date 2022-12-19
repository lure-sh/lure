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

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/repos"
	"go.arsenm.dev/lure/manager"
	"go.arsenm.dev/lure/vercmp"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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

	err = repos.Pull(c.Context, gdb, cfg.Repos)
	if err != nil {
		log.Fatal("Error pulling repos").Err(err).Send()
	}

	updates, err := checkForUpdates(c.Context, mgr, info)
	if err != nil {
		log.Fatal("Error checking for updates").Err(err).Send()
	}

	if len(updates) > 0 {
		installPkgs(c.Context, updates, nil, mgr)
	} else {
		log.Info("There is nothing to do.").Send()
	}

	return nil
}

func checkForUpdates(ctx context.Context, mgr manager.Manager, info *distro.OSRelease) ([]db.Package, error) {
	installed, err := mgr.ListInstalled(nil)
	if err != nil {
		return nil, err
	}

	pkgNames := maps.Keys(installed)
	found, _, err := repos.FindPkgs(gdb, pkgNames)
	if err != nil {
		return nil, err
	}

	var out []db.Package
	for pkgName, pkgs := range found {
		if slices.Contains(cfg.IgnorePkgUpdates, pkgName) {
			continue
		}

		if len(pkgs) > 1 {
			// Puts the element with the highest version first
			slices.SortFunc(pkgs, func(a, b db.Package) bool {
				return vercmp.Compare(a.Version, b.Version) == 1
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
