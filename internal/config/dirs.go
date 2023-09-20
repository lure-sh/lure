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

package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"go.elara.ws/lure/internal/log"
)

type Paths struct {
	ConfigDir  string
	ConfigPath string
	CacheDir   string
	RepoDir    string
	PkgsDir    string
	DBPath     string
}

var paths *Paths

func GetPaths() *Paths {
	if paths == nil {
		paths = &Paths{}

		cfgDir, err := os.UserConfigDir()
		if err != nil {
			log.Fatal("Unable to detect user config directory").Err(err).Send()
		}

		paths.ConfigDir = filepath.Join(cfgDir, "lure")

		err = os.MkdirAll(paths.ConfigDir, 0o755)
		if err != nil {
			log.Fatal("Unable to create LURE config directory").Err(err).Send()
		}

		paths.ConfigPath = filepath.Join(paths.ConfigDir, "lure.toml")

		if _, err := os.Stat(paths.ConfigPath); err != nil {
			cfgFl, err := os.Create(paths.ConfigPath)
			if err != nil {
				log.Fatal("Unable to create LURE config file").Err(err).Send()
			}

			err = toml.NewEncoder(cfgFl).Encode(&defaultConfig)
			if err != nil {
				log.Fatal("Error encoding default configuration").Err(err).Send()
			}

			cfgFl.Close()
		}

		cacheDir, err := os.UserCacheDir()
		if err != nil {
			log.Fatal("Unable to detect cache directory").Err(err).Send()
		}

		paths.CacheDir = filepath.Join(cacheDir, "lure")
		paths.RepoDir = filepath.Join(paths.CacheDir, "repo")
		paths.PkgsDir = filepath.Join(paths.CacheDir, "pkgs")

		err = os.MkdirAll(paths.RepoDir, 0o755)
		if err != nil {
			log.Fatal("Unable to create repo cache directory").Err(err).Send()
		}

		err = os.MkdirAll(paths.PkgsDir, 0o755)
		if err != nil {
			log.Fatal("Unable to create package cache directory").Err(err).Send()
		}

		paths.DBPath = filepath.Join(paths.CacheDir, "db")
	}
	return paths
}
