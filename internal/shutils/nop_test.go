package shutils_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"go.arsenm.dev/lure/internal/shutils"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func TestNopExec(t *testing.T) {
	ctx := context.Background()

	fl, err := syntax.NewParser().Parse(strings.NewReader(`/bin/echo test`), "lure.sh")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	buf := &bytes.Buffer{}
	runner, err := interp.New(
		interp.ExecHandler(shutils.NopExec),
		interp.StdIO(os.Stdin, buf, buf),
	)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	err = runner.Run(ctx, fl)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if buf.String() != "" {
		t.Fatalf("Expected empty string, got %#v", buf.String())
	}
}
