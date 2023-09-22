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
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/urfave/cli/v2"
	"go.elara.ws/lure/internal/log"
	"go.elara.ws/lure/internal/types"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/db"
	"go.elara.ws/lure/pkg/repos"
	"golang.org/x/exp/slices"
)

var addrepoCmd = &cli.Command{
	Name:    "addrepo",
	Usage:   "Add a new repository",
	Aliases: []string{"ar"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Required: true,
			Usage:    "Name of the new repo",
		},
		&cli.StringFlag{
			Name:     "url",
			Aliases:  []string{"u"},
			Required: true,
			Usage:    "URL of the new repo",
		},
	},
	Action: func(c *cli.Context) error {
		name := c.String("name")
		repoURL := c.String("url")

		cfg := config.Config()

		for _, repo := range cfg.Repos {
			if repo.URL == repoURL {
				log.Fatal("Repo already exists").Str("name", repo.Name).Send()
			}
		}

		cfg.Repos = append(cfg.Repos, types.Repo{
			Name: name,
			URL:  repoURL,
		})

		cfgFl, err := os.Create(config.GetPaths().ConfigPath)
		if err != nil {
			log.Fatal("Error opening config file").Err(err).Send()
		}

		err = toml.NewEncoder(cfgFl).Encode(cfg)
		if err != nil {
			log.Fatal("Error encoding config").Err(err).Send()
		}

		err = repos.Pull(c.Context, cfg.Repos)
		if err != nil {
			log.Fatal("Error pulling repos").Err(err).Send()
		}

		return nil
	},
}

var removerepoCmd = &cli.Command{
	Name:    "removerepo",
	Usage:   "Remove an existing repository",
	Aliases: []string{"rr"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Required: true,
			Usage:    "Name of the repo to be deleted",
		},
	},
	Action: func(c *cli.Context) error {
		name := c.String("name")
		cfg := config.Config()

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

		cfgFl, err := os.Create(config.GetPaths().ConfigPath)
		if err != nil {
			log.Fatal("Error opening config file").Err(err).Send()
		}

		err = toml.NewEncoder(cfgFl).Encode(&cfg)
		if err != nil {
			log.Fatal("Error encoding config").Err(err).Send()
		}

		err = os.RemoveAll(filepath.Join(config.GetPaths().RepoDir, name))
		if err != nil {
			log.Fatal("Error removing repo directory").Err(err).Send()
		}

		err = db.DeletePkgs("repository = ?", name)
		if err != nil {
			log.Fatal("Error removing packages from database").Err(err).Send()
		}

		return nil
	},
}

var refreshCmd = &cli.Command{
	Name:    "refresh",
	Usage:   "Pull all repositories that have changed",
	Aliases: []string{"ref"},
	Action: func(c *cli.Context) error {
		err := repos.Pull(c.Context, config.Config().Repos)
		if err != nil {
			log.Fatal("Error pulling repos").Err(err).Send()
		}
		return nil
	},
}
