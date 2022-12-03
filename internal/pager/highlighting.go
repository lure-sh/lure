package pager

import (
	"bytes"
	"io"

	"github.com/alecthomas/chroma/v2/quick"
)

func SyntaxHighlightBash(r io.Reader, style string) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	w := &bytes.Buffer{}
	err = quick.Highlight(w, string(data), "bash", "terminal", style)
	return w.String(), err
}
