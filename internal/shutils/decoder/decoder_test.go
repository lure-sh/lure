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

package decoder_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"go.elara.ws/lure/distro"
	"go.elara.ws/lure/internal/shutils/decoder"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

type BuildVars struct {
	Name          string   `sh:"name,required"`
	Version       string   `sh:"version,required"`
	Release       int      `sh:"release,required"`
	Epoch         uint     `sh:"epoch"`
	Description   string   `sh:"desc"`
	Homepage      string   `sh:"homepage"`
	Maintainer    string   `sh:"maintainer"`
	Architectures []string `sh:"architectures"`
	Licenses      []string `sh:"license"`
	Provides      []string `sh:"provides"`
	Conflicts     []string `sh:"conflicts"`
	Depends       []string `sh:"deps"`
	BuildDepends  []string `sh:"build_deps"`
	Replaces      []string `sh:"replaces"`
}

const testScript = `
	name='test'
	version='0.0.1'
	release=1
	epoch=2
	desc="Test package"
	homepage='https://lure.arsenm.dev'
	maintainer='Arsen Musayelyan <arsen@arsenm.dev>'
	architectures=('arm64' 'amd64')
	license=('GPL-3.0-or-later')
	provides=('test')
	conflicts=('test')
	replaces=('test-old')
	replaces_test_os=('test-legacy')

	deps=('sudo')

	build_deps=('golang')
	build_deps_arch=('go')

	test() {
		echo "Test"
	}

	package() {
		install-binary test
	}
`

var osRelease = &distro.OSRelease{
	ID:   "test_os",
	Like: []string{"arch"},
}

func TestDecodeVars(t *testing.T) {
	ctx := context.Background()

	fl, err := syntax.NewParser().Parse(strings.NewReader(testScript), "lure.sh")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	runner, err := interp.New()
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	err = runner.Run(ctx, fl)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	dec := decoder.New(osRelease, runner)

	var bv BuildVars
	err = dec.DecodeVars(&bv)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	expected := BuildVars{
		Name:          "test",
		Version:       "0.0.1",
		Release:       1,
		Epoch:         2,
		Description:   "Test package",
		Homepage:      "https://lure.arsenm.dev",
		Maintainer:    "Arsen Musayelyan <arsen@arsenm.dev>",
		Architectures: []string{"arm64", "amd64"},
		Licenses:      []string{"GPL-3.0-or-later"},
		Provides:      []string{"test"},
		Conflicts:     []string{"test"},
		Replaces:      []string{"test-legacy"},
		Depends:       []string{"sudo"},
		BuildDepends:  []string{"go"},
	}

	if !reflect.DeepEqual(bv, expected) {
		t.Errorf("Expected %v, got %v", expected, bv)
	}
}

func TestDecodeVarsMissing(t *testing.T) {
	ctx := context.Background()

	const testScript = `
		name='test'
		epoch=2
		desc="Test package"
		homepage='https://lure.arsenm.dev'
		maintainer='Arsen Musayelyan <arsen@arsenm.dev>'
		architectures=('arm64' 'amd64')
		license=('GPL-3.0-or-later')
		provides=('test')
		conflicts=('test')
		replaces=('test-old')
		replaces_test_os=('test-legacy')

		deps=('sudo')

		build_deps=('golang')
		build_deps_arch=('go')

		test() {
			echo "Test"
		}

		package() {
			install-binary test
		}
	`

	fl, err := syntax.NewParser().Parse(strings.NewReader(testScript), "lure.sh")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	runner, err := interp.New()
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	err = runner.Run(ctx, fl)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	dec := decoder.New(osRelease, runner)

	var bv BuildVars
	err = dec.DecodeVars(&bv)

	var notFoundErr decoder.VarNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("Expected VarNotFoundError, got %T %v", err, err)
	}
}

func TestGetFunc(t *testing.T) {
	ctx := context.Background()

	fl, err := syntax.NewParser().Parse(strings.NewReader(testScript), "lure.sh")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	runner, err := interp.New()
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	err = runner.Run(ctx, fl)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	dec := decoder.New(osRelease, runner)
	fn, ok := dec.GetFunc("test")
	if !ok {
		t.Fatalf("Expected test() function to exist")
	}

	buf := &bytes.Buffer{}
	err = fn(ctx, interp.StdIO(os.Stdin, buf, buf))
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if buf.String() != "Test\n" {
		t.Fatalf(`Expected "Test\n", got %#v`, buf.String())
	}
}
