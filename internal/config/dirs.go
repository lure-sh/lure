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

package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"go.elara.ws/logger/log"
)

var (
	ConfigDir  string
	ConfigPath string
	CacheDir   string
	RepoDir    string
	PkgsDir    string
	DBPath     string
)

// DBPresent is true if the database
// was present when LURE was started
var DBPresent bool

func init() {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Unable to detect user config directory").Err(err).Send()
	}

	ConfigDir = filepath.Join(cfgDir, "lure")

	err = os.MkdirAll(ConfigDir, 0o755)
	if err != nil {
		log.Fatal("Unable to create LURE config directory").Err(err).Send()
	}

	ConfigPath = filepath.Join(ConfigDir, "lure.toml")

	if _, err := os.Stat(ConfigPath); err != nil {
		cfgFl, err := os.Create(ConfigPath)
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

	CacheDir = filepath.Join(cacheDir, "lure")
	RepoDir = filepath.Join(CacheDir, "repo")
	PkgsDir = filepath.Join(CacheDir, "pkgs")

	err = os.MkdirAll(RepoDir, 0o755)
	if err != nil {
		log.Fatal("Unable to create repo cache directory").Err(err).Send()
	}

	err = os.MkdirAll(PkgsDir, 0o755)
	if err != nil {
		log.Fatal("Unable to create package cache directory").Err(err).Send()
	}

	DBPath = filepath.Join(CacheDir, "db")

	fi, err := os.Stat(DBPath)
	DBPresent = err == nil && !fi.IsDir()
}
