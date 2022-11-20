/*
 * LURE - Linux User REpository
 * Copyright (C) 2022 Arsen Musayelyan
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

package distro

import (
	"context"
	"errors"
	"os"
	"strings"

	"go.arsenm.dev/lure/internal/shutils"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var ErrParse = errors.New("could not parse os-release file")

type OSRelease struct {
	Name             string
	PrettyName       string
	ID               string
	Like             []string
	VersionID        string
	ANSIColor        string
	HomeURL          string
	DocumentationURL string
	SupportURL       string
	BugReportURL     string
	Logo             string
}

// OSReleaseName returns the NAME field of the
func ParseOSRelease(ctx context.Context) (*OSRelease, error) {
	fl, err := os.Open("/usr/lib/os-release")
	if err != nil {
		fl, err = os.Open("/etc/os-release")
		if err != nil {
			return nil, err
		}
	}

	file, err := syntax.NewParser().Parse(fl, "/usr/lib/os-release")
	if err != nil {
		return nil, err
	}

	fl.Close()

	// Create new shell interpreter with nop open, exec, readdir, and stat handlers
	// as well as no environment variables in order to prevent vulnerabilities
	// caused by changing the os-release file.
	runner, err := interp.New(
		interp.OpenHandler(shutils.NopOpen),
		interp.ExecHandler(shutils.NopExec),
		interp.ReadDirHandler(shutils.NopReadDir),
		interp.StatHandler(shutils.NopStat),
		interp.Env(expand.ListEnviron()),
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, file)
	if err != nil {
		return nil, ErrParse
	}

	out := &OSRelease{
		Name:             runner.Vars["NAME"].Str,
		PrettyName:       runner.Vars["PRETTY_NAME"].Str,
		ID:               runner.Vars["ID"].Str,
		VersionID:        runner.Vars["VERSION_ID"].Str,
		ANSIColor:        runner.Vars["ANSI_COLOR"].Str,
		HomeURL:          runner.Vars["HOME_URL"].Str,
		DocumentationURL: runner.Vars["DOCUMENTATION_URL"].Str,
		SupportURL:       runner.Vars["SUPPORT_URL"].Str,
		BugReportURL:     runner.Vars["BUG_REPORT_URL"].Str,
		Logo:             runner.Vars["LOGO"].Str,
	}

	if runner.Vars["ID_LIKE"].IsSet() {
		out.Like = strings.Split(runner.Vars["ID_LIKE"].Str, " ")
	}

	return out, nil
}
