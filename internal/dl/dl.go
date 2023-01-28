package dl

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/dlcache"
)

var Downloaders = []Downloader{
	FileDownloader{},
}

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

type Options struct {
	ID          string
	Name        string
	URL         string
	Destination string
	Progress    io.Writer
}

type Downloader interface {
	Name() string
	Type() Type
	MatchURL(string) bool
	Download(Options) error
}

type UpdatingDownloader interface {
	Downloader
	Update(Options) error
}

func Download(ctx context.Context, opts Options) error {
	d := getDownloader(opts.URL)
	cacheDir, ok := dlcache.Get(opts.ID)
	if ok {
		ok, err := handleCache(cacheDir, opts.Destination, d.Type())
		if err != nil {
			return err
		}

		if ok {
			log.Info("Source found in cache, linked to destination").Str("source", opts.Name).Stringer("type", d.Type()).Send()
			return nil
		}
	}

	log.Info("Downloading source").Str("source", opts.Name).Str("downloader", d.Name()).Send()

	cacheDir, err := dlcache.New(opts.ID)
	if err != nil {
		return err
	}

	err = d.Download(Options{
		Name:        opts.Name,
		URL:         opts.URL,
		Destination: cacheDir,
		Progress:    opts.Progress,
	})
	if err != nil {
		return err
	}

	_, err = handleCache(cacheDir, opts.Destination, d.Type())
	return err
}

func handleCache(cacheDir, dest string, t Type) (bool, error) {
	switch t {
	case TypeFile:
		cd, err := os.Open(cacheDir)
		if err != nil {
			return false, err
		}

		names, err := cd.Readdirnames(1)
		if err != nil && err != io.EOF {
			return false, err
		}

		// If the cache dir contains no files,
		// assume there is no cache entry
		if len(names) == 0 {
			break
		}

		err = os.Link(filepath.Join(cacheDir, names[0]), filepath.Join(dest, filepath.Base(names[0])))
		if err != nil {
			return false, err
		}
		return true, nil
	case TypeDir:
		err := os.Link(cacheDir, dest)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func getDownloader(u string) Downloader {
	for _, d := range Downloaders {
		if d.MatchURL(u) {
			return d
		}
	}
	return nil
}
