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
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/manager"
)

var (
	cacheDir string
	cfgPath  string
	config   Config
)

type Config struct {
	RootCmd string `toml:"rootCmd"`
	Repos   []Repo `toml:"repo"`
}

type Repo struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}

var defaultConfig = Config{
	RootCmd: "sudo",
	Repos: []Repo{
		{
			Name: "default",
			URL:  "https://github.com/Arsen6331/lure-repo.git",
		},
	},
}

func init() {
	cfg, cache, err := makeDirs()
	if err != nil {
		log.Fatal("Error creating directories").Err(err).Send()
	}
	cacheDir = cache

	cfgPath = filepath.Join(cfg, "lure.toml")

	cfgFl, err := os.Open(cfgPath)
	if err != nil {
		log.Fatal("Error opening config file").Err(err).Send()
	}
	defer cfgFl.Close()

	err = toml.NewDecoder(cfgFl).Decode(&config)
	if err != nil {
		log.Fatal("Error decoding config file").Err(err).Send()
	}

	manager.DefaultRootCmd = config.RootCmd
}

func makeDirs() (string, string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", "", err
	}

	baseCfgPath := filepath.Join(cfgDir, "lure")

	err = os.MkdirAll(baseCfgPath, 0o755)
	if err != nil {
		return "", "", err
	}

	cfgPath := filepath.Join(baseCfgPath, "lure.toml")

	if _, err := os.Stat(cfgPath); err != nil {
		cfgFl, err := os.Create(cfgPath)
		if err != nil {
			return "", "", err
		}

		err = toml.NewEncoder(cfgFl).Encode(&defaultConfig)
		if err != nil {
			return "", "", err
		}

		cfgFl.Close()
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", "", err
	}

	baseCachePath := filepath.Join(cacheDir, "lure")

	err = os.MkdirAll(filepath.Join(baseCachePath, "repo"), 0o755)
	if err != nil {
		return "", "", err
	}

	err = os.MkdirAll(filepath.Join(baseCachePath, "pkgs"), 0o755)
	if err != nil {
		return "", "", err
	}

	return baseCfgPath, baseCachePath, nil
}
