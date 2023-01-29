package dl

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mholt/archiver/v4"
	"github.com/schollz/progressbar/v3"
	"go.arsenm.dev/lure/internal/shutils"
)

// FileDownloader downloads files using HTTP
type FileDownloader struct{}

// Name always returns "file"
func (FileDownloader) Name() string {
	return "file"
}

// MatchURL always returns true, as FileDownloader
// is used as a fallback if nothing else matches
func (FileDownloader) MatchURL(string) bool {
	return true
}

// Download downloads a file using HTTP. If the file is
// compressed using a supported format, it will be extracted
func (FileDownloader) Download(opts Options) (Type, string, error) {
	res, err := http.Get(opts.URL)
	if err != nil {
		return 0, "", err
	}

	name := getFilename(res)
	path := filepath.Join(opts.Destination, name)
	fl, err := os.Create(path)
	if err != nil {
		return 0, "", err
	}
	defer fl.Close()

	var bar io.WriteCloser
	if opts.Progress != nil {
		bar = progressbar.NewOptions64(
			res.ContentLength,
			progressbar.OptionSetDescription(name),
			progressbar.OptionSetWriter(opts.Progress),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(10),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() {
				_, _ = io.WriteString(opts.Progress, "\n")
			}),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetRenderBlankState(true),
		)
		defer bar.Close()
	} else {
		bar = shutils.NopRWC{}
	}

	h := sha256.New()

	var w io.Writer
	if opts.SHA256 != nil {
		w = io.MultiWriter(fl, h, bar)
	} else {
		w = io.MultiWriter(fl, bar)
	}

	_, err = io.Copy(w, res.Body)
	if err != nil {
		return 0, "", err
	}
	res.Body.Close()

	if opts.SHA256 != nil {
		sum := h.Sum(nil)
		if !bytes.Equal(sum, opts.SHA256) {
			return 0, "", ErrChecksumMismatch
		}
	}

	if opts.PostprocDisabled {
		return TypeFile, "", nil
	}

	_, err = fl.Seek(0, io.SeekStart)
	if err != nil {
		return 0, "", err
	}

	format, r, err := archiver.Identify(name, fl)
	if err == archiver.ErrNoMatch {
		return TypeFile, "", nil
	} else if err != nil {
		return 0, "", err
	}

	err = extractFile(r, format, name, opts)
	if err != nil {
		return 0, "", err
	}

	err = os.Remove(path)
	return TypeDir, "", err
}

// extractFile extracts an archive or decompresses a file
func extractFile(r io.Reader, format archiver.Format, name string, opts Options) (err error) {
	fname := format.Name()

	switch format := format.(type) {
	case archiver.Extractor:
		err = format.Extract(context.Background(), r, nil, func(ctx context.Context, f archiver.File) error {
			fr, err := f.Open()
			if err != nil {
				return err
			}
			defer fr.Close()
			fi, err := f.Stat()
			if err != nil {
				return err
			}
			fm := fi.Mode()

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
				outFl, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fm.Perm())
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

	return nil
}

var cdHeaderRgx = regexp.MustCompile(`filename="(.+)"`)

// getFilename attempts to parse the Content-Disposition
// HTTP response header and extract a filename. If the
// header does not exist, it will use the last element
// of the path.
func getFilename(res *http.Response) (name string) {
	cd := res.Header.Get("Content-Disposition")
	matches := cdHeaderRgx.FindStringSubmatch(cd)
	if len(matches) > 1 {
		name = matches[1]
	} else {
		name = path.Base(res.Request.URL.Path)
	}
	return name
}
