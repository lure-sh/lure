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

package manager

import (
	"os"
	"os/exec"
)

var Args []string

type Opts struct {
	AsRoot    bool
	NoConfirm bool
	Args      []string
}

var DefaultOpts = &Opts{
	AsRoot:    true,
	NoConfirm: false,
}

// DefaultRootCmd is the command used for privilege elevation by default
var DefaultRootCmd = "sudo"

var managers = []Manager{
	&Pacman{},
	&APT{},
	&DNF{},
	&YUM{},
	&APK{},
	&Zypper{},
}

// Register registers a new package manager
func Register(m Manager) {
	managers = append(managers, m)
}

// Manager represents a system package manager
type Manager interface {
	// Name returns the name of the manager.
	Name() string
	// Format returns the packaging format of the manager.
	// 	Examples: rpm, deb, apk
	Format() string
	// Returns true if the package manager exists on the system.
	Exists() bool
	// Sets the command used to elevate privileges. Defaults to DefaultRootCmd.
	SetRootCmd(string)
	// Sync fetches repositories without installing anything
	Sync(*Opts) error
	// Install installs packages
	Install(*Opts, ...string) error
	// Remove uninstalls packages
	Remove(*Opts, ...string) error
	// Upgrade upgrades packages
	Upgrade(*Opts, ...string) error
	// InstallLocal installs packages from local files rather than repos
	InstallLocal(*Opts, ...string) error
	// UpgradeAll upgrades all packages
	UpgradeAll(*Opts) error
	// ListInstalled returns all installed packages mapped to their versions
	ListInstalled(*Opts) (map[string]string, error)
}

// Detect returns the package manager detected on the system
func Detect() Manager {
	for _, mgr := range managers {
		if mgr.Exists() {
			return mgr
		}
	}
	return nil
}

// Get returns the package manager with the given name
func Get(name string) Manager {
	for _, mgr := range managers {
		if mgr.Name() == name {
			return mgr
		}
	}
	return nil
}

// getRootCmd returns rootCmd if it's not empty, otherwise returns DefaultRootCmd
func getRootCmd(rootCmd string) string {
	if rootCmd != "" {
		return rootCmd
	}
	return DefaultRootCmd
}

func setCmdEnv(cmd *exec.Cmd) {
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}

func ensureOpts(opts *Opts) *Opts {
	if opts == nil {
		opts = DefaultOpts
	}
	opts.Args = append(opts.Args, Args...)
	return opts
}
