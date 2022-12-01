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

	"github.com/AlecAivazis/survey/v2"
	"github.com/pelletier/go-toml/v2"
	"github.com/urfave/cli/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/repos"
	"go.arsenm.dev/lure/internal/types"
	"golang.org/x/exp/slices"
)

func addrepoCmd(c *cli.Context) error {
	name := c.String("name")
	repoURL := c.String("url")

	for _, repo := range cfg.Repos {
		if repo.URL == repoURL {
			log.Fatal("Repo already exists").Str("name", repo.Name).Send()
		}
	}

	cfg.Repos = append(cfg.Repos, types.Repo{
		Name: name,
		URL:  repoURL,
	})

	cfgFl, err := os.Create(config.ConfigPath)
	if err != nil {
		log.Fatal("Error opening config file").Err(err).Send()
	}

	err = toml.NewEncoder(cfgFl).Encode(&cfg)
	if err != nil {
		log.Fatal("Error encoding config").Err(err).Send()
	}

	err = repos.Pull(c.Context, gdb, cfg.Repos)
	if err != nil {
		log.Fatal("Error pulling repos").Err(err).Send()
	}

	return nil
}

func removerepoCmd(c *cli.Context) error {
	name := c.String("name")

	found := false
	index := 0
	for i, repo := range cfg.Repos {
		if repo.Name == name {
			index = i
			found = true
		}
	}
	if !found {
		log.Fatal("Repo does not exist").Str("name", name).Send()
	}

	cfg.Repos = slices.Delete(cfg.Repos, index, index+1)

	cfgFl, err := os.Create(config.ConfigPath)
	if err != nil {
		log.Fatal("Error opening config file").Err(err).Send()
	}

	err = toml.NewEncoder(cfgFl).Encode(&cfg)
	if err != nil {
		log.Fatal("Error encoding config").Err(err).Send()
	}

	err = os.RemoveAll(filepath.Join(config.RepoDir, name))
	if err != nil {
		log.Fatal("Error removing repo directory").Err(err).Send()
	}

	err = db.DeletePkgs(gdb, "repository = ?", name)
	if err != nil {
		log.Fatal("Error removing packages from database").Err(err).Send()
	}

	return nil
}

func refreshCmd(c *cli.Context) error {
	err := repos.Pull(c.Context, gdb, cfg.Repos)
	if err != nil {
		log.Fatal("Error pulling repos").Err(err).Send()
	}
	return nil
}
