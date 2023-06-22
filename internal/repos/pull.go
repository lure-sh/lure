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

package repos

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/jmoiron/sqlx"
	"github.com/pelletier/go-toml/v2"
	"go.elara.ws/logger/log"
	"go.elara.ws/lure/distro"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/db"
	"go.elara.ws/lure/internal/shutils"
	"go.elara.ws/lure/internal/shutils/decoder"
	"go.elara.ws/lure/internal/types"
	"go.elara.ws/vercmp"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Pull pulls the provided repositories. If a repo doesn't exist, it will be cloned
// and its packages will be written to the DB. If it does exist, it will be pulled.
// In this case, only changed packages will be processed.
func Pull(ctx context.Context, gdb *sqlx.DB, repos []types.Repo) error {
	for _, repo := range repos {
		repoURL, err := url.Parse(repo.URL)
		if err != nil {
			return err
		}

		log.Info("Pulling repository").Str("name", repo.Name).Send()
		repoDir := filepath.Join(config.RepoDir, repo.Name)

		var repoFS billy.Filesystem
		gitDir := filepath.Join(repoDir, ".git")
		// Only pull repos that contain valid git repos
		if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
			r, err := git.PlainOpen(repoDir)
			if err != nil {
				return err
			}

			w, err := r.Worktree()
			if err != nil {
				return err
			}

			old, err := r.Head()
			if err != nil {
				return err
			}

			err = w.PullContext(ctx, &git.PullOptions{Progress: os.Stderr})
			if errors.Is(err, git.NoErrAlreadyUpToDate) {
				log.Info("Repository up to date").Str("name", repo.Name).Send()
			} else if err != nil {
				return err
			}
			repoFS = w.Filesystem

			// Make sure the DB is created even if the repo is up to date
			if !errors.Is(err, git.NoErrAlreadyUpToDate) || !config.DBPresent {
				new, err := r.Head()
				if err != nil {
					return err
				}

				// If the DB was not present at startup, that means it's
				// empty. In this case, we need to update the DB fully
				// rather than just incrementally.
				if config.DBPresent {
					err = processRepoChanges(ctx, repo, r, w, old, new, gdb)
					if err != nil {
						return err
					}
				} else {
					err = processRepoFull(ctx, repo, repoDir, gdb)
					if err != nil {
						return err
					}
				}
			}
		} else {
			err = os.RemoveAll(repoDir)
			if err != nil {
				return err
			}

			err = os.MkdirAll(repoDir, 0o755)
			if err != nil {
				return err
			}

			_, err = git.PlainCloneContext(ctx, repoDir, false, &git.CloneOptions{
				URL:      repoURL.String(),
				Progress: os.Stderr,
			})
			if err != nil {
				return err
			}

			err = processRepoFull(ctx, repo, repoDir, gdb)
			if err != nil {
				return err
			}

			repoFS = osfs.New(repoDir)
		}

		fl, err := repoFS.Open("lure-repo.toml")
		if err != nil {
			log.Warn("Git repository does not appear to be a valid LURE repo").Str("repo", repo.Name).Send()
			continue
		}

		var repoCfg types.RepoConfig
		err = toml.NewDecoder(fl).Decode(&repoCfg)
		if err != nil {
			return err
		}
		fl.Close()

		currentVer, _, _ := strings.Cut(config.Version, "-")
		if vercmp.Compare(currentVer, repoCfg.Repo.MinVersion) == -1 {
			log.Warn("LURE repo's minumum LURE version is greater than the current version. Try updating LURE if something doesn't work.").Str("repo", repo.Name).Send()
		}
	}

	return nil
}

type actionType uint8

const (
	actionDelete actionType = iota
	actionUpdate
)

type action struct {
	Type actionType
	File string
}

func processRepoChanges(ctx context.Context, repo types.Repo, r *git.Repository, w *git.Worktree, old, new *plumbing.Reference, gdb *sqlx.DB) error {
	oldCommit, err := r.CommitObject(old.Hash())
	if err != nil {
		return err
	}

	newCommit, err := r.CommitObject(new.Hash())
	if err != nil {
		return err
	}

	patch, err := oldCommit.Patch(newCommit)
	if err != nil {
		return err
	}

	var actions []action
	for _, fp := range patch.FilePatches() {
		from, to := fp.Files()

		if !isValid(from, to) {
			continue
		}

		if to == nil {
			actions = append(actions, action{
				Type: actionDelete,
				File: from.Path(),
			})
		} else if from == nil {
			actions = append(actions, action{
				Type: actionUpdate,
				File: to.Path(),
			})
		} else {
			if from.Path() != to.Path() {
				actions = append(actions,
					action{
						Type: actionDelete,
						File: from.Path(),
					},
					action{
						Type: actionUpdate,
						File: to.Path(),
					},
				)
			} else {
				actions = append(actions, action{
					Type: actionUpdate,
					File: to.Path(),
				})
			}
		}
	}

	repoDir := w.Filesystem.Root()
	parser := syntax.NewParser()

	for _, action := range actions {
		env := append(os.Environ(), "scriptdir="+filepath.Dir(filepath.Join(repoDir, action.File)))
		runner, err := interp.New(
			interp.Env(expand.ListEnviron(env...)),
			interp.ExecHandler(shutils.NopExec),
			interp.ReadDirHandler(shutils.RestrictedReadDir(repoDir)),
			interp.StatHandler(shutils.RestrictedStat(repoDir)),
			interp.OpenHandler(shutils.RestrictedOpen(repoDir)),
			interp.StdIO(shutils.NopRWC{}, shutils.NopRWC{}, shutils.NopRWC{}),
		)
		if err != nil {
			return err
		}

		switch action.Type {
		case actionDelete:
			if filepath.Base(action.File) != "lure.sh" {
				continue
			}

			scriptFl, err := oldCommit.File(action.File)
			if err != nil {
				return nil
			}

			r, err := scriptFl.Reader()
			if err != nil {
				return nil
			}

			var pkg db.Package
			err = parseScript(ctx, parser, runner, r, &pkg)
			if err != nil {
				return err
			}

			err = db.DeletePkgs(gdb, "name = ? AND repository = ?", pkg.Name, repo.Name)
			if err != nil {
				return err
			}
		case actionUpdate:
			if filepath.Base(action.File) != "lure.sh" {
				action.File = filepath.Join(filepath.Dir(action.File), "lure.sh")
			}

			scriptFl, err := newCommit.File(action.File)
			if err != nil {
				return nil
			}

			r, err := scriptFl.Reader()
			if err != nil {
				return nil
			}

			pkg := db.Package{
				Description:  db.NewJSON(map[string]string{}),
				Homepage:     db.NewJSON(map[string]string{}),
				Maintainer:   db.NewJSON(map[string]string{}),
				Depends:      db.NewJSON(map[string][]string{}),
				BuildDepends: db.NewJSON(map[string][]string{}),
				Repository:   repo.Name,
			}

			err = parseScript(ctx, parser, runner, r, &pkg)
			if err != nil {
				return err
			}

			resolveOverrides(runner, &pkg)

			err = db.InsertPackage(gdb, pkg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// isValid makes sure the path of the file being updated is valid.
// It checks to make sure the file is not within a nested directory
// and that it is called lure.sh.
func isValid(from, to diff.File) bool {
	var path string
	if from != nil {
		path = from.Path()
	}
	if to != nil {
		path = to.Path()
	}

	match, _ := filepath.Match("*/*.sh", path)
	return match
}

func processRepoFull(ctx context.Context, repo types.Repo, repoDir string, gdb *sqlx.DB) error {
	glob := filepath.Join(repoDir, "/*/lure.sh")
	matches, err := filepath.Glob(glob)
	if err != nil {
		return err
	}

	parser := syntax.NewParser()

	for _, match := range matches {
		env := append(os.Environ(), "scriptdir="+filepath.Dir(match))
		runner, err := interp.New(
			interp.Env(expand.ListEnviron(env...)),
			interp.ExecHandler(shutils.NopExec),
			interp.ReadDirHandler(shutils.RestrictedReadDir(repoDir)),
			interp.StatHandler(shutils.RestrictedStat(repoDir)),
			interp.OpenHandler(shutils.RestrictedOpen(repoDir)),
			interp.StdIO(shutils.NopRWC{}, shutils.NopRWC{}, shutils.NopRWC{}),
		)
		if err != nil {
			return err
		}

		scriptFl, err := os.Open(match)
		if err != nil {
			return err
		}

		pkg := db.Package{
			Description:  db.NewJSON(map[string]string{}),
			Homepage:     db.NewJSON(map[string]string{}),
			Maintainer:   db.NewJSON(map[string]string{}),
			Depends:      db.NewJSON(map[string][]string{}),
			BuildDepends: db.NewJSON(map[string][]string{}),
			Repository:   repo.Name,
		}

		err = parseScript(ctx, parser, runner, scriptFl, &pkg)
		if err != nil {
			return err
		}

		resolveOverrides(runner, &pkg)

		err = db.InsertPackage(gdb, pkg)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseScript(ctx context.Context, parser *syntax.Parser, runner *interp.Runner, r io.ReadCloser, pkg *db.Package) error {
	defer r.Close()
	fl, err := parser.Parse(r, "lure.sh")
	if err != nil {
		return err
	}

	runner.Reset()
	err = runner.Run(ctx, fl)
	if err != nil {
		return err
	}

	d := decoder.New(&distro.OSRelease{}, runner)
	d.Overrides = false
	d.LikeDistros = false
	return d.DecodeVars(pkg)
}

var overridable = map[string]string{
	"deps":       "Depends",
	"build_deps": "BuildDepends",
	"desc":       "Description",
	"homepage":   "Homepage",
	"maintainer": "Maintainer",
}

func resolveOverrides(runner *interp.Runner, pkg *db.Package) {
	pkgVal := reflect.ValueOf(pkg).Elem()
	for name, val := range runner.Vars {
		for prefix, field := range overridable {
			if strings.HasPrefix(name, prefix) {
				override := strings.TrimPrefix(name, prefix)
				override = strings.TrimPrefix(override, "_")

				field := pkgVal.FieldByName(field)
				varVal := field.FieldByName("Val")
				varType := varVal.Type()

				switch varType.Elem().String() {
				case "[]string":
					varVal.SetMapIndex(reflect.ValueOf(override), reflect.ValueOf(val.List))
				case "string":
					varVal.SetMapIndex(reflect.ValueOf(override), reflect.ValueOf(val.Str))
				}
				break
			}
		}
	}
}
