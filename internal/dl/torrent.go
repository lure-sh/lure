package dl

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	urlMatchRegex       = regexp.MustCompile(`(magnet|torrent\+https?):.*`)
	ErrAria2NotFound    = errors.New("aria2 must be installed for torrent functionality")
	ErrDestinationEmpty = errors.New("the destination directory is empty")
)

type TorrentDownloader struct{}

// Name always returns "file"
func (TorrentDownloader) Name() string {
	return "torrent"
}

// MatchURL returns true if the URL is a magnet link
// or an http(s) link with a "torrent+" prefix
func (TorrentDownloader) MatchURL(u string) bool {
	return urlMatchRegex.MatchString(u)
}

// Download downloads a file over the BitTorrent protocol.
func (TorrentDownloader) Download(opts Options) (Type, string, error) {
	aria2Path, err := exec.LookPath("aria2c")
	if err != nil {
		return 0, "", ErrAria2NotFound
	}

	opts.URL = strings.TrimPrefix(opts.URL, "torrent+")

	cmd := exec.Command(aria2Path, "--summary-interval=0", "--log-level=warn", "--seed-time=0", "--dir="+opts.Destination, opts.URL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return 0, "", fmt.Errorf("aria2c returned an error: %w", err)
	}

	err = removeTorrentFiles(opts.Destination)
	if err != nil {
		return 0, "", err
	}

	return determineType(opts.Destination)
}

func removeTorrentFiles(path string) error {
	filePaths, err := filepath.Glob(filepath.Join(path, "*.torrent"))
	if err != nil {
		return err
	}

	for _, filePath := range filePaths {
		err = os.Remove(filePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func determineType(path string) (Type, string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return 0, "", err
	}

	if len(files) > 1 {
		return TypeDir, "", nil
	} else if len(files) == 1 {
		if files[0].IsDir() {
			return TypeDir, files[0].Name(), nil
		} else {
			return TypeFile, files[0].Name(), nil
		}
	}

	return 0, "", ErrDestinationEmpty
}
