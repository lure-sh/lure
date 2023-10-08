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

package repos_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"lure.sh/lure/internal/config"
	"lure.sh/lure/internal/db"
	"lure.sh/lure/internal/types"
	"lure.sh/lure/pkg/repos"
)

func setCfgDirs(t *testing.T) {
	t.Helper()

	paths := config.GetPaths()

	var err error
	paths.CacheDir, err = os.MkdirTemp("/tmp", "lure-pull-test.*")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	paths.RepoDir = filepath.Join(paths.CacheDir, "repo")
	paths.PkgsDir = filepath.Join(paths.CacheDir, "pkgs")

	err = os.MkdirAll(paths.RepoDir, 0o755)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	err = os.MkdirAll(paths.PkgsDir, 0o755)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	paths.DBPath = filepath.Join(paths.CacheDir, "db")
}

func removeCacheDir(t *testing.T) {
	t.Helper()

	err := os.RemoveAll(config.GetPaths().CacheDir)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
}

func TestPull(t *testing.T) {
	_, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
	defer db.Close()

	setCfgDirs(t)
	defer removeCacheDir(t)

	ctx := context.Background()

	err = repos.Pull(ctx, []types.Repo{
		{
			Name: "default",
			URL:  "https://github.com/Arsen6331/lure-repo.git",
		},
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	result, err := db.GetPkgs("name LIKE 'itd%'")
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
