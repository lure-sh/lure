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

	"github.com/pelletier/go-toml/v2"
	"go.elara.ws/lure/internal/log"
	"go.elara.ws/lure/internal/types"
)

var defaultConfig = &types.Config{
	RootCmd:          "sudo",
	PagerStyle:       "native",
	IgnorePkgUpdates: []string{},
	Repos: []types.Repo{
		{
			Name: "default",
			URL:  "https://github.com/Elara6331/lure-repo.git",
		},
	},
}

var config *types.Config

// Config returns a LURE configuration struct.
// The first time it's called, it'll load the config from a file.
// Subsequent calls will just return the same value.
func Config() *types.Config {
	if config == nil {
		cfgFl, err := os.Open(GetPaths().ConfigPath)
		if err != nil {
			log.Warn("Error opening config file, using defaults").Err(err).Send()
			return defaultConfig
		}
		defer cfgFl.Close()

		// Copy the default configuration into config
		defCopy := *defaultConfig
		config = &defCopy
		config.Repos = nil

		err = toml.NewDecoder(cfgFl).Decode(config)
		if err != nil {
			log.Warn("Error decoding config file, using defaults").Err(err).Send()
			// Set config back to nil so that we try again next time
			config = nil
			return defaultConfig
		}
	}

	return config
}
