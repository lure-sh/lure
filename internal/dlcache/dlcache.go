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

package dlcache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"go.elara.ws/lure/internal/config"
)

// BasePath returns the base path of the download cache
func BasePath(ctx context.Context) string {
	return filepath.Join(config.GetPaths(ctx).CacheDir, "dl")
}

// New creates a new directory with the given ID in the cache.
// If a directory with the same ID already exists,
// it will be deleted before creating a new one.
func New(ctx context.Context, id string) (string, error) {
	h, err := hashID(id)
	if err != nil {
		return "", err
	}
	itemPath := filepath.Join(BasePath(ctx), h)

	fi, err := os.Stat(itemPath)
	if err == nil || (fi != nil && !fi.IsDir()) {
		err = os.RemoveAll(itemPath)
		if err != nil {
			return "", err
		}
	}

	err = os.MkdirAll(itemPath, 0o755)
	if err != nil {
		return "", err
	}

	return itemPath, nil
}

// Get checks if an entry with the given ID
// already exists in the cache, and if so,
// returns the directory and true. If it
// does not exist, it returns an empty string
// and false.
func Get(ctx context.Context, id string) (string, bool) {
	h, err := hashID(id)
	if err != nil {
		return "", false
	}
	itemPath := filepath.Join(BasePath(ctx), h)

	_, err = os.Stat(itemPath)
	if err != nil {
		return "", false
	}

	return itemPath, true
}

// hashID hashes the input ID with SHA1
// and returns the hex string of the hashed
// ID.
func hashID(id string) (string, error) {
	h := sha1.New()
	_, err := io.WriteString(h, id)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
