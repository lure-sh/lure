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
	"os"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/repos"
)

func fixCmd(c *cli.Context) error {
	gdb.Close()

	log.Info("Removing cache directory").Send()

	err := os.RemoveAll(config.CacheDir)
	if err != nil {
		log.Fatal("Unable to remove cache directory").Err(err).Send()
	}

	log.Info("Rebuilding cache").Send()

	err = os.MkdirAll(config.CacheDir, 0o755)
	if err != nil {
		log.Fatal("Unable to create new cache directory").Err(err).Send()
	}

	// Make sure the DB is rebuilt when repos are pulled
	gdb, err = db.Open(config.DBPath)
	if err != nil {
		log.Fatal("Error initializing database").Err(err).Send()
	}
	config.DBPresent = false

	err = repos.Pull(c.Context, gdb, cfg.Repos)
	if err != nil {
		log.Fatal("Error pulling repos").Err(err).Send()
	}

	log.Info("Done").Send()

	return nil
}
