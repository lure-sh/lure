package repos_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/repos"
	"go.arsenm.dev/lure/internal/types"
)

func setCfgDirs(t *testing.T) {
	t.Helper()

	var err error
	config.CacheDir, err = os.MkdirTemp("/tmp", "lure-pull-test.*")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	config.RepoDir = filepath.Join(config.CacheDir, "repo")
	config.PkgsDir = filepath.Join(config.CacheDir, "pkgs")

	err = os.MkdirAll(config.RepoDir, 0o755)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	err = os.MkdirAll(config.PkgsDir, 0o755)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	config.DBPath = filepath.Join(config.CacheDir, "db")
}

func removeCacheDir(t *testing.T) {
	t.Helper()

	err := os.RemoveAll(config.CacheDir)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
}

func TestPull(t *testing.T) {
	gdb, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	defer gdb.Close()

	err = db.Init(gdb)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	setCfgDirs(t)
	defer removeCacheDir(t)

	ctx := context.Background()

	err = repos.Pull(ctx, gdb, []types.Repo{
		{
			Name: "default",
			URL:  "https://github.com/Arsen6331/lure-repo.git",
		},
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	result, err := db.GetPkgs(gdb, "name LIKE 'itd%'")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	var pkgAmt int
	for result.Next() {
		var dbPkg db.Package
		err = result.StructScan(&dbPkg)
		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}
		pkgAmt++
	}

	if pkgAmt < 2 {
		t.Errorf("Expected 2 packages to match, got %d", pkgAmt)
	}
}
