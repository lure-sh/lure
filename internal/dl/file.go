package dl

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/schollz/progressbar/v3"
)

type FileDownloader struct{}

func (FileDownloader) Name() string {
	return "file"
}

func (FileDownloader) Type() Type {
	return TypeFile
}

func (FileDownloader) MatchURL(string) bool {
	return true
}

func (FileDownloader) Download(opts Options) error {
	res, err := http.Get(opts.URL)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	name := getFilename(res)
	fl, err := os.Create(filepath.Join(opts.Destination, name))
	if err != nil {
		return err
	}

	bar := progressbar.DefaultBytes(
		res.ContentLength,
		"downloading "+name,
	)
	defer bar.Close()

	_, err = io.Copy(io.MultiWriter(fl, bar), res.Body)
	return err
}

var cdHeaderRgx = regexp.MustCompile(`filename="(.+)"`)

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
