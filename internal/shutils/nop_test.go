/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Arsen Musayelyan
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

package shutils_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"go.elara.ws/lure/internal/shutils"
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
