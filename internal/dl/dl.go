package dl

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/vmihailenco/msgpack/v5"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/dlcache"
)

const manifestFileName = ".lure_cache_manifest"

var ErrChecksumMismatch = errors.New("dl: checksums did not match")

var Downloaders = []Downloader{
	GitDownloader{},
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
	SHA256           []byte
	Name             string
	URL              string
	Destination      string
	CacheDisabled    bool
	PostprocDisabled bool
	Progress         io.Writer
}

type Manifest struct {
	Type Type
	Name string
}

type Downloader interface {
	Name() string
	MatchURL(string) bool
	Download(Options) (Type, string, error)
}

type UpdatingDownloader interface {
	Downloader
	Update(Options) (bool, error)
}

func Download(ctx context.Context, opts Options) (err error) {
	d := getDownloader(opts.URL)

	if opts.CacheDisabled {
		_, _, err = d.Download(opts)
		return err
	}

	var t Type
	cacheDir, ok := dlcache.Get(opts.URL)
	if ok {
		var updated bool
		if d, ok := d.(UpdatingDownloader); ok {
			log.Info("Source can be updated, updating if required").Str("source", opts.Name).Str("downloader", d.Name()).Send()

			updated, err = d.Update(Options{
				Name:        opts.Name,
				URL:         opts.URL,
				Destination: cacheDir,
				Progress:    opts.Progress,
			})
			if err != nil {
				return err
			}
		}

		m, err := getManifest(cacheDir)
		if err != nil {
			return err
		}
		t = m.Type

		dest := filepath.Join(opts.Destination, m.Name)
		ok, err := handleCache(cacheDir, dest, t)
		if err != nil {
			return err
		}

		if ok && !updated {
			log.Info("Source found in cache, linked to destination").Str("source", opts.Name).Stringer("type", t).Send()
			return nil
		} else if ok {
			return nil
		}
	}

	log.Info("Downloading source").Str("source", opts.Name).Str("downloader", d.Name()).Send()

	cacheDir, err = dlcache.New(opts.URL)
	if err != nil {
		return err
	}

	t, name, err := d.Download(Options{
		Name:        opts.Name,
		URL:         opts.URL,
		Destination: cacheDir,
		Progress:    opts.Progress,
	})
	if err != nil {
		return err
	}

	err = writeManifest(cacheDir, Manifest{t, name})
	if err != nil {
		return err
	}

	dest := filepath.Join(opts.Destination, name)
	_, err = handleCache(cacheDir, dest, t)
	return err
}

func writeManifest(cacheDir string, m Manifest) error {
	fl, err := os.Create(filepath.Join(cacheDir, manifestFileName))
	if err != nil {
		return err
	}
	defer fl.Close()
	return msgpack.NewEncoder(fl).Encode(m)
}

func getManifest(cacheDir string) (m Manifest, err error) {
	fl, err := os.Open(filepath.Join(cacheDir, manifestFileName))
	if err != nil {
		return Manifest{}, err
	}
	defer fl.Close()

	err = msgpack.NewDecoder(fl).Decode(&m)
	return
}

func handleCache(cacheDir, dest string, t Type) (bool, error) {
	switch t {
	case TypeFile:
		cd, err := os.Open(cacheDir)
		if err != nil {
			return false, err
		}

		names, err := cd.Readdirnames(2)
		if err == io.EOF {
			break
		} else if err != nil {
			return false, err
		}

		cd.Close()

		for _, name := range names {
			if name == manifestFileName {
				continue
			}

			err = os.Link(filepath.Join(cacheDir, names[0]), filepath.Join(dest, filepath.Base(names[0])))
			if err != nil {
				return false, err
			}
		}
		return true, nil
	case TypeDir:
		err := linkDir(cacheDir, dest)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

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
			return os.Mkdir(newPath, info.Mode())
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
