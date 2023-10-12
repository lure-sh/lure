package gen

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"text/template"
)

//go:embed tmpls/pip.tmpl.sh
var pipTmpl string

type PipOptions struct {
	Name        string
	Version     string
	Description string
}

type pypiAPIResponse struct {
	Info pypiInfo  `json:"info"`
	URLs []pypiURL `json:"urls"`
}

func (res pypiAPIResponse) SourceURL() (pypiURL, error) {
	for _, url := range res.URLs {
		if url.PackageType == "sdist" {
			return url, nil
		}
	}
	return pypiURL{}, errors.New("package doesn't have a source distribution")
}

type pypiInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Summary  string `json:"summary"`
	Homepage string `json:"home_page"`
	License  string `json:"license"`
}

type pypiURL struct {
	Digests     map[string]string `json:"digests"`
	Filename    string            `json:"filename"`
	PackageType string            `json:"packagetype"`
}

func Pip(w io.Writer, opts PipOptions) error {
	tmpl, err := template.New("pip").
		Funcs(funcs).
		Parse(pipTmpl)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://pypi.org/pypi/%s/%s/json",
		opts.Name,
		opts.Version,
	)

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("pypi: %s", res.Status)
	}

	var resp pypiAPIResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return err
	}

	if opts.Description != "" {
		resp.Info.Summary = opts.Description
	}

	return tmpl.Execute(w, resp)
}
