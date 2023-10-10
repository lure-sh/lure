package gen

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"path"
	"text/template"
)

//go:embed tmpls/pip.tmpl.sh
var pipTmpl string

type PipOptions struct {
	Name        string
	Version     string
	Description string
}

func Pip(w io.Writer, opts PipOptions) error {
	tmpl, err := template.New("pip").
		Funcs(funcs).
		Parse(pipTmpl)
	if err != nil {
		return err
	}

	params := map[string]any{
		"name":        opts.Name,
		"version":     opts.Version,
		"description": opts.Description,
	}

	url := fmt.Sprintf(
		"https://files.pythonhosted.org/packages/source/%s/%s/%s-%s.tar.gz",
		opts.Name[:1],
		opts.Name,
		opts.Name,
		opts.Version,
	)

	res, err := http.Head(url)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("pip: %s", res.Status)
	}

	dir := path.Dir(res.Request.URL.Path)
	checksum := path.Base(dir)
	dir = path.Dir(dir)
	checksum = path.Base(dir) + checksum
	dir = path.Dir(dir)
	checksum = path.Base(dir) + checksum
	params["checksum"] = "blake2b-256:" + checksum

	return tmpl.Execute(w, params)
}
