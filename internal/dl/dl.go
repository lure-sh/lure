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

// Package dl contains abstractions for downloadingfiles and directories
// from various sources.
package dl

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/purell"
	"github.com/vmihailenco/msgpack/v5"
	"go.elara.ws/lure/internal/dlcache"
	"go.elara.ws/lure/pkg/loggerctx"
	"golang.org/x/exp/slices"
)

const manifestFileName = ".lure_cache_manifest"

// ErrChecksumMismatch occurs when the checksum of a downloaded file
// does not match the expected checksum provided in the Options struct.
var (
	ErrChecksumMismatch = errors.New("dl: checksums did not match")
	ErrNoSuchHashAlgo   = errors.New("dl: invalid hashing algorithm")
)

// Downloaders contains all the downloaders in the order in which
// they should be checked
var Downloaders = []Downloader{
	GitDownloader{},
	TorrentDownloader{},
	FileDownloader{},
}

// Type represents the type of download (file or directory)
type Type uint8

const (
	TypeFile Type = iota
	TypeDir
)

func (t Type) String() string {
	switch t {
	case TypeFile:
		return "file"
	case TypeDir:
		return "dir"
	}
	return "<unknown>"
}

// Options contains the options for downloading
// files and directories
type Options struct {
	Hash             []byte
	HashAlgorithm    string
	Name             string
	URL              string
	Destination      string
	CacheDisabled    bool
	PostprocDisabled bool
	Progress         io.Writer
	LocalDir         string
}

func (opts Options) NewHash() (hash.Hash, error) {
	if opts.HashAlgorithm == "" {
		opts.HashAlgorithm = "sha256"
	}
	switch opts.HashAlgorithm {
	case "sha256":
		return sha256.New(), nil
	case "sha224":
		return sha256.New224(), nil
	case "sha512":
		return sha512.New(), nil
	case "sha384":
		return sha512.New384(), nil
	case "sha1":
		return sha1.New(), nil
	case "md5":
		return md5.New(), nil
	}
	return nil, fmt.Errorf("%w: %s", ErrNoSuchHashAlgo, opts.HashAlgorithm)
}

// Manifest holds information about the type and name
// of a downloaded file or directory. It is stored inside
// each cache directory for later use.
type Manifest struct {
	Type Type
	Name string
}

type Downloader interface {
	// Name returns the name of the downloader
	Name() string
	// MatchURL checks if the given URL matches
	// the downloader.
	MatchURL(string) bool
	// Download downloads the object at the URL
	// provided in the options, to the destination
	// given in the options. It returns a type,
	// a name for the downloaded object (this may be empty),
	// and an error.
	Download(Options) (Type, string, error)
}

// UpdatingDownloader extends the Downloader interface
// with an Update method for protocols such as git, which
// allow for incremental updates without changing the URL.
type UpdatingDownloader interface {
	Downloader
	// Update checks for and performs any
	// available updates for the object
	// described in the options. It returns
	// true if an update was performed, or
	// false if no update was required.
	Update(Options) (bool, error)
}

// Download downloads a file or directory using the specified options.
// It first gets the appropriate downloader for the URL, then checks
// if caching is enabled. If caching is enabled, it attempts to get
// the cache directory for the URL and update it if necessary.
// If the source is found in the cache, it links it to the destination
// using hard links. If the source is not found in the cache,
// it downloads the source to a new cache directory and links it
// to the destination.
func Download(ctx context.Context, opts Options) (err error) {
	log := loggerctx.From(ctx)
	normalized, err := normalizeURL(opts.URL)
	if err != nil {
		return err
	}
	opts.URL = normalized

	d := getDownloader(opts.URL)

	if opts.CacheDisabled {
		_, _, err = d.Download(opts)
		return err
	}

	var t Type
	cacheDir, ok := dlcache.Get(ctx, opts.URL)
	if ok {
		var updated bool
		if d, ok := d.(UpdatingDownloader); ok {
			log.Info("Source can be updated, updating if required").Str("source", opts.Name).Str("downloader", d.Name()).Send()

			updated, err = d.Update(Options{
				Hash:          opts.Hash,
				HashAlgorithm: opts.HashAlgorithm,
				Name:          opts.Name,
				URL:           opts.URL,
				Destination:   cacheDir,
				Progress:      opts.Progress,
				LocalDir:      opts.LocalDir,
			})
			if err != nil {
				return err
			}
		}

		m, err := getManifest(cacheDir)
		if err == nil {
			t = m.Type

			dest := filepath.Join(opts.Destination, m.Name)
			ok, err := handleCache(cacheDir, dest, m.Name, t)
			if err != nil {
				return err
			}

			if ok && !updated {
				log.Info("Source found in cache, linked to destination").Str("source", opts.Name).Stringer("type", t).Send()
				return nil
			} else if ok {
				return nil
			}
		} else {
			// If we cannot read the manifest,
			// this cache entry is invalid and
			// the source must be re-downloaded.
			err = os.RemoveAll(cacheDir)
			if err != nil {
				return err
			}
		}
	}

	log.Info("Downloading source").Str("source", opts.Name).Str("downloader", d.Name()).Send()

	cacheDir, err = dlcache.New(ctx, opts.URL)
	if err != nil {
		return err
	}

	t, name, err := d.Download(Options{
		Hash:          opts.Hash,
		HashAlgorithm: opts.HashAlgorithm,
		Name:          opts.Name,
		URL:           opts.URL,
		Destination:   cacheDir,
		Progress:      opts.Progress,
		LocalDir:      opts.LocalDir,
	})
	if err != nil {
		return err
	}

	err = writeManifest(cacheDir, Manifest{t, name})
	if err != nil {
		return err
	}

	dest := filepath.Join(opts.Destination, name)
	_, err = handleCache(cacheDir, dest, name, t)
	return err
}

// writeManifest writes the manifest to the specified cache directory.
func writeManifest(cacheDir string, m Manifest) error {
	fl, err := os.Create(filepath.Join(cacheDir, manifestFileName))
	if err != nil {
		return err
	}
	defer fl.Close()
	return msgpack.NewEncoder(fl).Encode(m)
}

// getManifest reads the manifest from the specified cache directory.
func getManifest(cacheDir string) (m Manifest, err error) {
	fl, err := os.Open(filepath.Join(cacheDir, manifestFileName))
	if err != nil {
		return Manifest{}, err
	}
	defer fl.Close()

	err = msgpack.NewDecoder(fl).Decode(&m)
	return
}

// handleCache links the cache directory or a file within it to the destination
func handleCache(cacheDir, dest, name string, t Type) (bool, error) {
	switch t {
	case TypeFile:
		cd, err := os.Open(cacheDir)
		if err != nil {
			return false, err
		}

		names, err := cd.Readdirnames(0)
		if err == io.EOF {
			break
		} else if err != nil {
			return false, err
		}

		cd.Close()

		if slices.Contains(names, name) {
			err = os.Link(filepath.Join(cacheDir, name), dest)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	case TypeDir:
		err := linkDir(cacheDir, dest)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// linkDir recursively walks through a directory, creating
// hard links for each file from the src directory to the
// dest directory. If it encounters a directory, it will
// create a directory with the same name and permissions
// in the dest directory, because hard links cannot be
// created for directories.
func linkDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == manifestFileName {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		newPath := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(newPath, info.Mode())
		}

		return os.Link(path, newPath)
	})
}

func getDownloader(u string) Downloader {
	for _, d := range Downloaders {
		if d.MatchURL(u) {
			return d
		}
	}
	return nil
}

// normalizeURL normalizes a URL string, so that insignificant
// differences don't change the hash.
func normalizeURL(u string) (string, error) {
	const normalizationFlags = purell.FlagRemoveTrailingSlash |
		purell.FlagRemoveDefaultPort |
		purell.FlagLowercaseHost |
		purell.FlagLowercaseScheme |
		purell.FlagRemoveDuplicateSlashes |
		purell.FlagRemoveFragment |
		purell.FlagRemoveUnnecessaryHostDots |
		purell.FlagSortQuery |
		purell.FlagDecodeHexHost |
		purell.FlagDecodeOctalHost |
		purell.FlagDecodeUnnecessaryEscapes |
		purell.FlagRemoveEmptyPortSeparator

	u, err := purell.NormalizeURLString(u, normalizationFlags)
	if err != nil {
		return "", err
	}

	// Fix magnet URLs after normalization
	u = strings.Replace(u, "magnet://", "magnet:", 1)
	return u, nil
}
