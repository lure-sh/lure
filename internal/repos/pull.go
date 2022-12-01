package repos

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/genjidb/genji"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pelletier/go-toml/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/download"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/shutils"
	"go.arsenm.dev/lure/internal/shutils/decoder"
	"go.arsenm.dev/lure/internal/types"
	"go.arsenm.dev/lure/vercmp"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Pull pulls the provided repositories. If a repo doesn't exist, it will be cloned
// and its packages will be written to the DB. If it does exist, it will be pulled.
// In this case, only changed packages will be processed.
func Pull(ctx context.Context, gdb *genji.DB, repos []types.Repo) error {
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

			if !errors.Is(err, git.NoErrAlreadyUpToDate) {
				new, err := r.Head()
				if err != nil {
					return err
				}

				err = processRepoChanges(ctx, repo, r, old, new, gdb)
				if err != nil {
					return err
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

type ActionType uint8

const (
	ActionDelete ActionType = iota
	ActionUpdate
)

type Action struct {
	Type ActionType
	File string
}

func processRepoChanges(ctx context.Context, repo types.Repo, r *git.Repository, old, new *plumbing.Reference, gdb *genji.DB) error {
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

	var actions []Action
	for _, fp := range patch.FilePatches() {
		from, to := fp.Files()
		if to == nil {
			actions = append(actions, Action{
				Type: ActionDelete,
				File: from.Path(),
			})
		} else if from == nil {
			actions = append(actions, Action{
				Type: ActionUpdate,
				File: to.Path(),
			})
		} else {
			if from.Path() != to.Path() {
				actions = append(actions,
					Action{
						Type: ActionDelete,
						File: from.Path(),
					},
					Action{
						Type: ActionUpdate,
						File: to.Path(),
					},
				)
			} else {
				actions = append(actions, Action{
					Type: ActionUpdate,
					File: to.Path(),
				})
			}
		}
	}

	parser := syntax.NewParser()
	runner, err := interp.New(
		interp.StatHandler(shutils.NopStat),
		interp.ExecHandler(shutils.NopExec),
		interp.OpenHandler(shutils.NopOpen),
		interp.ReadDirHandler(shutils.NopReadDir),
		interp.StdIO(shutils.NopRWC{}, shutils.NopRWC{}, shutils.NopRWC{}),
	)
	if err != nil {
		return err
	}

	for _, action := range actions {
		switch action.Type {
		case ActionDelete:
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
		case ActionUpdate:
			scriptFl, err := newCommit.File(action.File)
			if err != nil {
				return nil
			}

			r, err := scriptFl.Reader()
			if err != nil {
				return nil
			}

			pkg := db.Package{
				Depends:      map[string][]string{},
				BuildDepends: map[string][]string{},
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

func processRepoFull(ctx context.Context, repo types.Repo, repoDir string, gdb *genji.DB) error {
	glob := filepath.Join(repoDir, "/*/lure.sh")
	matches, err := filepath.Glob(glob)
	if err != nil {
		return err
	}

	parser := syntax.NewParser()
	runner, err := interp.New(
		interp.StatHandler(shutils.NopStat),
		interp.ExecHandler(shutils.NopExec),
		interp.OpenHandler(shutils.NopOpen),
		interp.ReadDirHandler(shutils.NopReadDir),
		interp.StdIO(shutils.NopRWC{}, shutils.NopRWC{}, shutils.NopRWC{}),
	)
	if err != nil {
		return err
	}

	for _, match := range matches {
		scriptFl, err := os.Open(match)
		if err != nil {
			return err
		}

		pkg := db.Package{
			Depends:      map[string][]string{},
			BuildDepends: map[string][]string{},
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

func resolveOverrides(runner *interp.Runner, pkg *db.Package) {
	for name, val := range runner.Vars {
		if strings.HasPrefix(name, "deps") {
			override := strings.TrimPrefix(name, "deps")
			override = strings.TrimPrefix(override, "_")

			pkg.Depends[override] = val.List
		} else if strings.HasPrefix(name, "build_deps") {
			override := strings.TrimPrefix(name, "build_deps")
			override = strings.TrimPrefix(override, "_")

			pkg.BuildDepends[override] = val.List
		} else {
			continue
		}
	}
}
