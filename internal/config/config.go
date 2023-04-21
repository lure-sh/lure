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

	"github.com/pelletier/go-toml/v2"
	"go.elara.ws/lure/internal/types"
)

var defaultConfig = types.Config{
	RootCmd:          "sudo",
	PagerStyle:       "native",
	IgnorePkgUpdates: []string{},
	Repos: []types.Repo{
		{
			Name: "default",
			URL:  "https://github.com/Arsen6331/lure-repo.git",
		},
	},
}

// Decode decodes the config file into the given
// pointer
func Decode(cfg *types.Config) error {
	cfgFl, err := os.Open(ConfigPath)
	if err != nil {
		return err
	}
	defer cfgFl.Close()

	// Write defaults to pointer in case some values are not set in the config
	*cfg = defaultConfig
	// Set repos to nil so as to avoid a duplicate default
	cfg.Repos = nil
	return toml.NewDecoder(cfgFl).Decode(cfg)
}
