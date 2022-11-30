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
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-git/go-git/v5"
	"github.com/pelletier/go-toml/v2"
	"github.com/urfave/cli/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/download"
	"go.arsenm.dev/lure/internal/config"
	"golang.org/x/exp/slices"
)

type PkgNotFoundError struct {
	pkgName string
}

func (p PkgNotFoundError) Error() string {
	return "package '" + p.pkgName + "' could not be found in any repository"
}

type RepoConfig struct {
	Repo struct {
		MinVersion string `toml:"minVersion"`
	}
}

func addrepoCmd(c *cli.Context) error {
	name := c.String("name")
	repoURL := c.String("url")

	for _, repo := range cfg.Repos {
		if repo.URL == repoURL {
			log.Fatal("Repo already exists").Str("name", repo.Name).Send()
		}
	}

	cfg.Repos = append(cfg.Repos, config.Repo{
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

	return nil
}

func refreshCmd(c *cli.Context) error {
	err := pullRepos(c.Context)
	if err != nil {
		log.Fatal("Error pulling repos").Err(err).Send()
	}
	return nil
}

func findPkg(pkg string) ([]string, error) {
	var out []string
	for _, repo := range cfg.Repos {
		repoDir := filepath.Join(config.RepoDir, repo.Name)
		err := os.MkdirAll(repoDir, 0o755)
		if err != nil {
			return nil, err
		}

		glob := filepath.Join(repoDir, pkg, "lure.sh")
		matches, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 {
			continue
		}

		out = append(out, matches...)
	}

	if len(out) == 0 {
		return nil, PkgNotFoundError{pkgName: pkg}
	}

	return out, nil
}

func pkgPrompt(options []string) ([]string, error) {
	names := make([]string, len(options))
	for i, option := range options {
		pkgDir := filepath.Dir(option)
		repoDir := filepath.Dir(pkgDir)
		names[i] = filepath.Base(repoDir) + "/" + filepath.Base(pkgDir)
	}

	prompt := &survey.MultiSelect{
		Options: names,
		Message: "Choose which package(s) to install",
	}

	var choices []int
	err := survey.AskOne(prompt, &choices)
	if err != nil {
		return nil, err
	}

	out := make([]string, len(choices))
	for i, choiceIndex := range choices {
		out[i] = options[choiceIndex]
	}

	return out, nil
}

func findPkgs(pkgs []string) (scripts, notFound []string) {
	for _, pkg := range pkgs {
		found, err := findPkg(pkg)
		if _, ok := err.(PkgNotFoundError); ok {
			notFound = append(notFound, pkg)
			continue
		}

		if len(found) == 1 {
			scripts = append(scripts, found...)
		} else {
			choices, err := pkgPrompt(found)
			if err != nil {
				log.Fatal("Error prompting for package choices").Err(err).Send()
			}

			scripts = append(scripts, choices...)
		}
	}
	return
}

func pullRepos(ctx context.Context) error {
	for _, repo := range cfg.Repos {
		repoURL, err := url.Parse(repo.URL)
		if err != nil {
			return err
		}

		log.Info("Pulling repository").Str("name", repo.Name).Send()
		repoDir := filepath.Join(config.RepoDir, repo.Name)

		gitDir := filepath.Join(repoDir, ".git")
		if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
			r, err := git.PlainOpen(repoDir)
			if err != nil {
				return err
			}

			w, err := r.Worktree()
			if err != nil {
				return err
			}

			err = w.PullContext(ctx, &git.PullOptions{Progress: os.Stderr})
			if err == git.NoErrAlreadyUpToDate {
				log.Info("Repository up to date").Str("name", repo.Name).Send()
				continue
			} else if err != nil {
				return err
			}

			fl, err := w.Filesystem.Open("lure-repo.toml")
			if err != nil {
				log.Warn("Git repository does not appear to be a valid LURE repo").Str("repo", repo.Name).Send()
				continue
			}

			var repoCfg RepoConfig
			err = toml.NewDecoder(fl).Decode(&repoCfg)
			if err != nil {
				return err
			}
			fl.Close()

			currentVer, _, _ := strings.Cut(version, "-")
			if vercmp(currentVer, repoCfg.Repo.MinVersion) == -1 {
				log.Warn("LURE repo's minumum LURE version is greater than the current version. Try updating LURE if something doesn't work.").Str("repo", repo.Name).Send()
			}

			continue
		}

		err = os.RemoveAll(repoDir)
		if err != nil {
			return err
		}

		err = os.MkdirAll(repoDir, 0o755)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(repoURL.Scheme, "git+") {
			repoURL.Scheme = "git+" + repoURL.Scheme
		}

		err = download.Get(ctx, download.GetOptions{
			SourceURL:   repoURL.String(),
			Destination: repoDir,
		})
		if err != nil {
			return err
		}

		fl, err := os.Open(filepath.Join(repoDir, "lure-repo.toml"))
		if err != nil {
			log.Warn("Git repository does not appear to be a valid LURE repo").Str("repo", repo.Name).Send()
		}

		var repoCfg RepoConfig
		err = toml.NewDecoder(fl).Decode(&repoCfg)
		if err != nil {
			return err
		}
		fl.Close()

		currentVer, _, _ := strings.Cut(version, "-")
		if vercmp(currentVer, repoCfg.Repo.MinVersion) == -1 {
			log.Warn("LURE repo's minumum LURE version is greater than the current version. Try updating LURE if something doesn't work.").Str("repo", repo.Name).Send()
		}
	}

	return nil
}
