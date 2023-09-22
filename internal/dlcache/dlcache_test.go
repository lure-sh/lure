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

package dlcache_test

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.elara.ws/lure/internal/dlcache"
	"go.elara.ws/lure/internal/config"
)

func init() {
	dir, err := os.MkdirTemp("/tmp", "lure-dlcache-test.*")
	if err != nil {
		panic(err)
	}
	config.GetPaths().RepoDir = dir
}

func TestNew(t *testing.T) {
	const id = "https://example.com"
	dir, err := dlcache.New(id)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	exp := filepath.Join(dlcache.BasePath(), sha1sum(id))
	if dir != exp {
		t.Errorf("Expected %s, got %s", exp, dir)
	}

	fi, err := os.Stat(dir)
	if err != nil {
		t.Errorf("stat: expected no error, got %s", err)
	}

	if !fi.IsDir() {
		t.Errorf("Expected cache item to be a directory")
	}

	dir2, ok := dlcache.Get(id)
	if !ok {
		t.Errorf("Expected Get() to return valid value")
	}
	if dir2 != dir {
		t.Errorf("Expected %s from Get(), got %s", dir, dir2)
	}
}

func sha1sum(id string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, id)
	return hex.EncodeToString(h.Sum(nil))
}
