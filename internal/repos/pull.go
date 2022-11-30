package repos

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/pelletier/go-toml/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/download"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/types"
	"go.arsenm.dev/lure/vercmp"
)

func Pull(ctx context.Context, repos []types.Repo) error {
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

			err = w.PullContext(ctx, &git.PullOptions{Progress: os.Stderr})
			if err == git.NoErrAlreadyUpToDate {
				log.Info("Repository up to date").Str("name", repo.Name).Send()
			} else if err != nil {
				return err
			}

			repoFS = w.Filesystem
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
