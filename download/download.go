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

package download

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/mholt/archiver/v4"
)

var ErrChecksumMismatch = errors.New("checksums did not match")

type GetOptions struct {
	SourceURL   string
	Destination string
	SHA256Sum   []byte
	// EncloseGit determines if Get will create an enclosing
	// directory for git repos
	EncloseGit bool
}

// Get downloads from a URL
func Get(ctx context.Context, opts GetOptions) error {
	dest, err := filepath.Abs(opts.Destination)
	if err != nil {
		return err
	}
	opts.Destination = dest

	err = os.MkdirAll(opts.Destination, 0o755)
	if err != nil {
		return err
	}

	src, err := url.Parse(opts.SourceURL)
	if err != nil {
		return err
	}
	query := src.Query()

	tag := query.Get("~tag")
	query.Del("~tag")

	branch := src.Query().Get("~branch")
	query.Del("~branch")

	commit := src.Query().Get("~commit")
	query.Del("~commit")

	var refName plumbing.ReferenceName
	if tag != "" {
		refName = plumbing.NewTagReferenceName(tag)
	} else if branch != "" {
		refName = plumbing.NewBranchReferenceName(branch)
	}

	if strings.HasPrefix(src.Scheme, "git+") {
		src.Scheme = strings.TrimPrefix(src.Scheme, "git+")
		src.RawQuery = query.Encode()

		name := path.Base(src.Path)
		name = strings.TrimSuffix(name, ".git")

		dstDir := opts.Destination
		if opts.EncloseGit {
			dstDir = filepath.Join(opts.Destination, name)
		}

		repo, err := git.PlainCloneContext(ctx, dstDir, false, &git.CloneOptions{
			URL:      src.String(),
			Progress: os.Stderr,
		})
		if err != nil {
			return err
		}

		w, err := repo.Worktree()
		if err != nil {
			return err
		}

		checkoutOpts := &git.CheckoutOptions{}
		if refName != "" {
			checkoutOpts.Branch = refName
		} else if commit != "" {
			checkoutOpts.Hash = plumbing.NewHash(commit)
		} else {
			return nil
		}

		return w.Checkout(checkoutOpts)
	} else {
		name := query.Get("~name")
		query.Del("~name")

		archive := query.Get("~archive")
		query.Del("~archive")

		src.RawQuery = query.Encode()

		if name == "" {
			name = path.Base(src.Path)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.String(), nil)
		if err != nil {
			return err
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		hash := sha256.New()

		format, input, err := archiver.Identify(name, res.Body)
		if err == archiver.ErrNoMatch || archive == "false" {
			fl, err := os.Create(filepath.Join(opts.Destination, name))
			if err != nil {
				return err
			}

			w := io.MultiWriter(hash, fl)

			_, err = io.Copy(w, input)
			if err != nil {
				return err
			}

			res.Body.Close()
			fl.Close()

			if opts.SHA256Sum != nil {
				sum := hash.Sum(nil)
				if !bytes.Equal(opts.SHA256Sum, sum) {
					return ErrChecksumMismatch
				}
			}
		} else if err != nil {
			return err
		} else {
			r := io.TeeReader(input, hash)
			fname := format.Name()

			switch format := format.(type) {
			case archiver.Extractor:
				err = format.Extract(ctx, r, nil, func(ctx context.Context, f archiver.File) error {
					fr, err := f.Open()
					if err != nil {
						return err
					}
					defer fr.Close()

					path := filepath.Join(opts.Destination, f.NameInArchive)

					err = os.MkdirAll(filepath.Dir(path), 0o755)
					if err != nil {
						return err
					}

					if f.IsDir() {
						err = os.Mkdir(path, 0o755)
						if err != nil {
							return err
						}
					} else {
						outFl, err := os.Create(path)
						if err != nil {
							return err
						}
						defer outFl.Close()

						_, err = io.Copy(outFl, fr)
						return err
					}
					return nil
				})
				if err != nil {
					return err
				}
			case archiver.Decompressor:
				rc, err := format.OpenReader(r)
				if err != nil {
					return err
				}
				defer rc.Close()

				path := filepath.Join(opts.Destination, name)
				path = strings.TrimSuffix(path, fname)

				outFl, err := os.Create(path)
				if err != nil {
					return err
				}

				_, err = io.Copy(outFl, rc)
				if err != nil {
					return err
				}
			}

			if opts.SHA256Sum != nil {
				sum := hash.Sum(nil)
				if !bytes.Equal(opts.SHA256Sum, sum) {
					return ErrChecksumMismatch
				}
			}
		}
	}

	return nil
}
